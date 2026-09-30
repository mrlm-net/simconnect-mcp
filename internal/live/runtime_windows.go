//go:build windows

package live

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/engine"
	"github.com/mrlm-net/simconnect/pkg/manager"
	"github.com/mrlm-net/simconnect/pkg/nav"
	"github.com/mrlm-net/simconnect/pkg/types"
)

// SimConnect IDs of the library components. They sit far above the IDs the
// bridge allocates and the library's defaults, and below the manager's
// reserved range (999,999,850+). airport.ProcedureLoader has fixed IDs
// (8400+/8500+) the bridge keeps clear of.
const (
	idBase           uint32 = 900_000_000
	layoutDefBase           = idBase           // 6 definitions
	layoutReqBase           = idBase + 100     // 96 requests
	fixDefBase              = idBase + 1_000   // 3 definitions
	fixReqBase              = idBase + 1_100   // 2 per slot
	crawlDefBase            = idBase + 2_000   // 3 definitions
	crawlReqBase            = idBase + 2_100   // 2 per slot
	weatherDefID            = idBase + 3_000   //
	weatherReqID            = idBase + 3_001   //
	fixSlots                = 16               //
	crawlSlots              = 32               //
	layoutTimeout           = 20 * time.Second // also how long an unknown ICAO code takes to fail
	procedureTimeout        = 30 * time.Second // the ProcedureLoader has no timeout of its own
	weatherMaxAge           = 30 * time.Second // older without an update: request it again
	maxCrawlRequests        = 6000             //
)

// Runtime is the live Source: the library's loaders fed from a manager's
// messages. Messages are handled in the manager's dispatch goroutine, under
// mu, because their buffers are only valid during the callback; tool calls
// take mu to queue requests and wait on channels outside it.
type Runtime struct {
	mgr manager.Manager

	mu        sync.Mutex
	connected bool
	cache     *airport.Cache
	loader    *airport.Loader
	procs     *airport.ProcedureLoader
	fixes     *nav.NavLoader
	crawlNav  *nav.NavLoader
	weather   *nav.WeatherReader
	traffic   *trafficState
	runways   RunwayMemory

	layoutWait map[string][]chan layoutResult
	procWait   map[string][]chan procResult
	procSince  map[string]time.Time
	procCache  map[string]*airport.Procedures
	fixWait    map[nav.FixKey][]chan nav.NavResult
	fixQueue   []nav.FixKey
	wxWait     []chan nav.Weather
	wxSubbed   bool
	wxAt       time.Time
	crawl      *crawlJob
	crawlQueue []*crawlJob
	graphs     []*crawlJob // finished crawls, newest last

	stop chan struct{}
	once sync.Once
}

type layoutResult struct {
	layout *airport.Layout
	err    error
}

type procResult struct {
	procs *airport.Procedures
	err   error
}

type crawlJob struct {
	center   airport.LatLon
	radiusNM float64
	seeds    []nav.FixKey
	crawler  *nav.AirwayCrawler
	graph    *nav.AirwayGraph
	err      error
	done     chan struct{}
}

// NewRuntime attaches a Runtime to mgr. Call Close to detach it.
// RunwaysInUse implements Source.
func (r *Runtime) RunwaysInUse(l *airport.Layout, w nav.Weather, lim nav.RunwayLimits) nav.RunwayUse {
	return r.runways.Use(l, w, lim)
}

func NewRuntime(mgr manager.Manager) *Runtime {
	r := &Runtime{
		mgr:        mgr,
		cache:      airport.NewCache(),
		layoutWait: map[string][]chan layoutResult{},
		procWait:   map[string][]chan procResult{},
		procSince:  map[string]time.Time{},
		procCache:  map[string]*airport.Procedures{},
		fixWait:    map[nav.FixKey][]chan nav.NavResult{},
		stop:       make(chan struct{}),
	}
	r.resetLocked()
	r.connected = mgr.ConnectionState() == manager.StateConnected || mgr.ConnectionState() == manager.StateAvailable
	mgr.OnOpen(func(types.ConnectionOpenData) {
		r.mu.Lock()
		defer r.mu.Unlock()
		// A new connection has none of the old definitions or requests.
		r.failAllLocked(ErrReconnected)
		r.resetLocked()
		r.connected = true
	})
	mgr.OnConnectionStateChange(func(_, state manager.ConnectionState) {
		r.mu.Lock()
		defer r.mu.Unlock()
		up := state == manager.StateConnected || state == manager.StateAvailable
		if r.connected && !up {
			r.failAllLocked(ErrNotConnected)
		}
		r.connected = up
	})
	mgr.OnMessage(r.handle)
	go r.tickLoop()
	return r
}

// Close takes our AI aircraft out of the simulator and stops the Runtime's
// timer; it returns how many aircraft it removed. The simulator keeps AI
// objects after their client disconnects, so call it before closing the
// connection. Its handlers stay registered with the manager, which is closed
// with it.
func (r *Runtime) Close() int {
	n := r.RemoveAll()
	r.once.Do(func() { close(r.stop) })
	return n
}

// resetLocked creates the loaders anew: after a reconnect their facility
// definitions must be registered again, and the ProcedureLoader has no Reset.
func (r *Runtime) resetLocked() {
	r.loader = airport.NewLoader(r.mgr, airport.LoaderWithIDs(layoutDefBase, layoutReqBase),
		airport.LoaderWithCache(r.cache), airport.LoaderWithTimeout(layoutTimeout))
	r.procs = airport.NewProcedureLoader(r.mgr)
	r.fixes = nav.NewNavLoaderWithIDs(r.mgr, fixDefBase, fixReqBase, fixSlots)
	r.crawlNav = nav.NewNavLoaderWithIDs(r.mgr, crawlDefBase, crawlReqBase, crawlSlots)
	r.weather = nav.NewWeatherReader(r.mgr, weatherDefID, weatherReqID)
	// The simulator removed our aircraft with the old connection; the
	// traffic state is made again for the new one when needed.
	r.traffic = nil
	r.wxSubbed = false
}

// failAllLocked ends every waiter with err.
func (r *Runtime) failAllLocked(err error) {
	for icao, ws := range r.layoutWait {
		for _, w := range ws {
			w <- layoutResult{err: err}
		}
		delete(r.layoutWait, icao)
	}
	for icao, ws := range r.procWait {
		for _, w := range ws {
			w <- procResult{err: err}
		}
		delete(r.procWait, icao)
		delete(r.procSince, icao)
	}
	for k, ws := range r.fixWait {
		for _, w := range ws {
			close(w)
		}
		delete(r.fixWait, k)
	}
	r.fixQueue = nil
	for _, w := range r.wxWait {
		close(w)
	}
	r.wxWait = nil
	for _, j := range append(r.crawlQueue, r.crawl) {
		if j != nil {
			j.err = err
			close(j.done)
		}
	}
	r.crawl, r.crawlQueue = nil, nil
}

// Connected implements Source.
func (r *Runtime) Connected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connected
}

// handle runs in the manager's dispatch goroutine for every message.
func (r *Runtime) handle(msg engine.Message) {
	if msg.Err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if types.SIMCONNECT_RECV_ID(msg.DwID) == types.SIMCONNECT_RECV_ID_OPEN {
		r.connected = true
	}
	if res, done := r.loader.Handle(msg); done {
		r.finishLayoutLocked(res)
		return
	}
	if p, done := r.procs.Handle(msg); done {
		r.finishProcsLocked(p)
		return
	}
	if res, done := r.fixes.Handle(msg); done {
		r.finishFixLocked(res)
		return
	}
	if w, ok := r.weather.Handle(msg); ok {
		r.wxAt = time.Now()
		for _, ch := range r.wxWait {
			ch <- w
		}
		r.wxWait = nil
		return
	}
	if r.crawl != nil {
		done, err := r.crawl.crawler.Handle(msg)
		if done || err != nil {
			r.finishCrawlLocked(err)
		}
	}
	r.handleTrafficLocked(msg)
}

func (r *Runtime) tickLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-r.stop:
			return
		case now := <-t.C:
			r.tick(now)
		}
	}
}

func (r *Runtime) tick(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, res := range r.loader.Expire(now) {
		r.finishLayoutLocked(res)
	}
	for _, res := range r.fixes.Expire(now) {
		r.finishFixLocked(res)
	}
	// The ProcedureLoader has no timeout: an airport it never hears back
	// about keeps its slot. Start it afresh and fail what was in flight.
	for _, since := range r.procSince {
		if now.Sub(since) > procedureTimeout {
			for icao := range r.procWait {
				for _, w := range r.procWait[icao] {
					w <- procResult{err: fmt.Errorf("procedures of %s: %w", icao, ErrTimeout)}
				}
			}
			r.procWait, r.procSince = map[string][]chan procResult{}, map[string]time.Time{}
			r.procs = airport.NewProcedureLoader(r.mgr)
			break
		}
	}
	if r.crawl != nil {
		done, err := r.crawl.crawler.Tick(now)
		if done || err != nil {
			r.finishCrawlLocked(err)
		}
	}
	r.tickTrafficLocked(now)
}

// ── Layouts ─────────────────────────────────────────────────────────────────

// Layout implements Source.
func (r *Runtime) Layout(ctx context.Context, icao string) (*airport.Layout, error) {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	if l, ok := r.cache.Layout(icao); ok {
		return l, nil
	}
	ch := make(chan layoutResult, 1)
	r.mu.Lock()
	if !r.connected {
		r.mu.Unlock()
		return nil, ErrNotConnected
	}
	first := len(r.layoutWait[icao]) == 0
	r.layoutWait[icao] = append(r.layoutWait[icao], ch)
	if first {
		if err := r.loader.Request(icao); err != nil {
			delete(r.layoutWait, icao)
			r.mu.Unlock()
			return nil, err
		}
	}
	r.mu.Unlock()

	select {
	case res := <-ch:
		return res.layout, res.err
	case <-ctx.Done():
		r.mu.Lock()
		r.layoutWait[icao] = removeChan(r.layoutWait[icao], ch)
		r.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (r *Runtime) finishLayoutLocked(res airport.Result) {
	err := res.Err
	// SimConnect sends nothing for an unknown airport: the load times out.
	if errors.Is(err, airport.ErrTimeout) {
		err = fmt.Errorf("airport %s: %w (unknown ICAO code?)", res.ICAO, ErrNotFound)
	}
	for _, w := range r.layoutWait[res.ICAO] {
		w <- layoutResult{layout: res.Layout, err: err}
	}
	delete(r.layoutWait, res.ICAO)
}

// Graph implements Source.
func (r *Runtime) Graph(ctx context.Context, icao string) (*airport.Graph, error) {
	if _, err := r.Layout(ctx, icao); err != nil {
		return nil, err
	}
	return r.cache.Graph(icao)
}

// ── Procedures ──────────────────────────────────────────────────────────────

// Procedures implements Source. The layout is loaded first: the simulator
// never answers for an unknown airport, and the layout loader times out.
func (r *Runtime) Procedures(ctx context.Context, icao string) (*airport.Procedures, error) {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	if _, err := r.Layout(ctx, icao); err != nil {
		return nil, err
	}
	ch := make(chan procResult, 1)
	r.mu.Lock()
	if p, ok := r.procCache[icao]; ok {
		r.mu.Unlock()
		return p, nil
	}
	first := len(r.procWait[icao]) == 0
	r.procWait[icao] = append(r.procWait[icao], ch)
	if first {
		if err := r.procs.Request(icao); err != nil {
			delete(r.procWait, icao)
			r.mu.Unlock()
			return nil, err
		}
		r.procSince[icao] = time.Now()
	}
	r.mu.Unlock()

	select {
	case res := <-ch:
		return res.procs, res.err
	case <-ctx.Done():
		r.mu.Lock()
		r.procWait[icao] = removeChan(r.procWait[icao], ch)
		r.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (r *Runtime) finishProcsLocked(p airport.Procedures) {
	pp := &p
	r.procCache[p.ICAO] = pp
	for _, w := range r.procWait[p.ICAO] {
		w <- procResult{procs: pp}
	}
	delete(r.procWait, p.ICAO)
	delete(r.procSince, p.ICAO)
}

// ── Weather ─────────────────────────────────────────────────────────────────

// Weather implements Source. The first call subscribes to the weather at
// the user aircraft; later calls return the last value.
func (r *Runtime) Weather(ctx context.Context) (nav.Weather, error) {
	ch := make(chan nav.Weather, 1)
	r.mu.Lock()
	if !r.connected {
		r.mu.Unlock()
		return nav.Weather{}, ErrNotConnected
	}
	if w, ok := r.weather.Last(); ok && r.wxSubbed && time.Since(r.wxAt) < weatherMaxAge {
		r.mu.Unlock()
		return w, nil
	}
	if !r.wxSubbed {
		if err := r.weather.Subscribe(); err != nil {
			r.mu.Unlock()
			return nav.Weather{}, err
		}
		r.wxSubbed = true
	} else if err := r.weather.Request(); err != nil {
		r.mu.Unlock()
		return nav.Weather{}, err
	}
	r.wxWait = append(r.wxWait, ch)
	r.mu.Unlock()

	select {
	case w, ok := <-ch:
		if !ok {
			return nav.Weather{}, ErrReconnected
		}
		return w, nil
	case <-ctx.Done():
		r.mu.Lock()
		r.wxWait = removeChan(r.wxWait, ch)
		r.mu.Unlock()
		return nav.Weather{}, fmt.Errorf("weather: %w", ErrTimeout)
	}
}

// ── Fixes ───────────────────────────────────────────────────────────────────

// Fix implements Source.
func (r *Runtime) Fix(ctx context.Context, key nav.FixKey) (nav.NavResult, error) {
	ch := make(chan nav.NavResult, 1)
	r.mu.Lock()
	if !r.connected {
		r.mu.Unlock()
		return nav.NavResult{}, ErrNotConnected
	}
	first := len(r.fixWait[key]) == 0
	r.fixWait[key] = append(r.fixWait[key], ch)
	if first {
		if r.fixes.Free() == 0 {
			r.fixQueue = append(r.fixQueue, key)
		} else if err := r.fixes.Request(key); err != nil {
			delete(r.fixWait, key)
			r.mu.Unlock()
			return nav.NavResult{}, err
		}
	}
	r.mu.Unlock()

	select {
	case res, ok := <-ch:
		if !ok {
			return nav.NavResult{}, ErrReconnected
		}
		if !res.Found {
			return res, fmt.Errorf("fix %s: %w", key, ErrNotFound)
		}
		return res, nil
	case <-ctx.Done():
		r.mu.Lock()
		r.fixWait[key] = removeChan(r.fixWait[key], ch)
		r.mu.Unlock()
		return nav.NavResult{}, ctx.Err()
	}
}

func (r *Runtime) finishFixLocked(res nav.NavResult) {
	for _, w := range r.fixWait[res.Key] {
		w <- res
	}
	delete(r.fixWait, res.Key)
	for len(r.fixQueue) > 0 && r.fixes.Free() > 0 {
		k := r.fixQueue[0]
		r.fixQueue = r.fixQueue[1:]
		if err := r.fixes.Request(k); err != nil {
			r.finishFixLocked(nav.NavResult{Key: k})
		}
	}
}

// ── Airways ─────────────────────────────────────────────────────────────────

// Airways implements Source. A finished crawl covering the requested
// circle is reused; otherwise a crawl is queued (one runs at a time) and
// finishes even when ctx ends, for the next caller.
func (r *Runtime) Airways(ctx context.Context, center airport.LatLon, radiusNM float64, seeds []nav.FixKey) (*nav.AirwayGraph, error) {
	r.mu.Lock()
	if !r.connected {
		r.mu.Unlock()
		return nil, ErrNotConnected
	}
	for i := len(r.graphs) - 1; i >= 0; i-- {
		j := r.graphs[i]
		if calc.HaversineNM(j.center.Lat, j.center.Lon, center.Lat, center.Lon)+radiusNM <= j.radiusNM {
			r.mu.Unlock()
			return j.graph, nil
		}
	}
	job := &crawlJob{center: center, radiusNM: radiusNM, seeds: seeds, done: make(chan struct{})}
	r.crawlQueue = append(r.crawlQueue, job)
	if r.crawl == nil {
		r.startCrawlLocked()
	}
	r.mu.Unlock()

	select {
	case <-job.done:
		return job.graph, job.err
	case <-ctx.Done():
		return nil, fmt.Errorf("airway crawl: %w", ErrTimeout)
	}
}

func (r *Runtime) startCrawlLocked() {
	for len(r.crawlQueue) > 0 {
		job := r.crawlQueue[0]
		r.crawlQueue = r.crawlQueue[1:]
		job.crawler = nav.NewAirwayCrawler(r.crawlNav, nav.CrawlOptions{Center: job.center, RadiusNM: job.radiusNM, MaxRequests: maxCrawlRequests})
		if err := job.crawler.Start(job.seeds...); err != nil {
			job.err = err
			close(job.done)
			continue
		}
		r.crawl = job
		return
	}
}

func (r *Runtime) finishCrawlLocked(err error) {
	job := r.crawl
	r.crawl = nil
	job.graph, job.err = job.crawler.Graph(), err
	if err == nil {
		r.graphs = append(r.graphs, job)
		if len(r.graphs) > 4 {
			r.graphs = r.graphs[1:]
		}
	}
	close(job.done)
	r.startCrawlLocked()
}

// ── Flight plans ────────────────────────────────────────────────────────────

// LoadFlightPlan implements Source: it writes pln to a file in the temp
// directory and has the simulator load it.
func (r *Runtime) LoadFlightPlan(_ context.Context, pln []byte) error {
	if !r.Connected() {
		return ErrNotConnected
	}
	dir := filepath.Join(os.TempDir(), "simconnect-mcp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := filepath.Join(dir, fmt.Sprintf("plan-%d", time.Now().UnixNano()))
	if err := os.WriteFile(name+".pln", pln, 0o644); err != nil {
		return err
	}
	// SimConnect_FlightPlanLoad takes the path without the .pln extension.
	return r.mgr.FlightPlanLoad(name)
}

func removeChan[T any](s []chan T, ch chan T) []chan T {
	for i, c := range s {
		if c == ch {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
