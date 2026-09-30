//go:build windows

package live

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/engine"
	"github.com/mrlm-net/simconnect/pkg/traffic"
	"github.com/mrlm-net/simconnect/pkg/types"
)

// Traffic controls AI aircraft of our own and sees the simulator's traffic.
type Traffic interface {
	// Models lists the installed aircraft titles ("title" or "title|livery").
	Models(ctx context.Context) ([]string, error)
	// SpawnDeparture puts an aircraft on a stand, to push, taxi and take off.
	SpawnDeparture(ctx context.Context, s DepartureSpec) (FlightView, error)
	// SpawnArrival puts an aircraft on its approach, to land and taxi in.
	SpawnArrival(ctx context.Context, s ArrivalSpec) (FlightView, error)
	// Flights lists our aircraft.
	Flights() []FlightView
	// Clear gives one of our aircraft a clearance or instruction (Actions).
	Clear(callsign, action string) (FlightView, error)
	// Picture returns the aircraft within radiusNM of the user aircraft, or
	// of centre (an airport ICAO code) when set.
	Picture(ctx context.Context, centre string, radiusNM float64) ([]traffic.TrackedAircraft, error)
	// Sequences are the landing sequences at icao ("" all), Tower who
	// uses its runways, ATCLog the latest instructions of the runtime's
	// controllers (at most limit; 0 all).
	Sequences(icao string) []RunwaySequence
	Tower(icao string) []RunwayUser
	ATCLog(limit int) []ATCMessage
	// Approach gives one of our arrivals in a landing sequence an
	// instruction (up, down, slow, hold, release, direct, goaround) and
	// returns what was said.
	Approach(callsign, action string) (string, error)
}

// DepartureSpec asks for a controlled departure. Stand, Runway and Model
// are chosen when empty (the tools choose the runway and SID).
type DepartureSpec struct {
	Graph             *airport.Graph
	Limits            *airport.Limits
	Callsign          string
	Stand             string // label; "" assigns one
	Runway            string
	Entry             string
	Model, Livery     string
	Type              string // ICAO type for choosing a model; default A320
	SID               string
	Departure         []airport.NavPoint // the SID, flown after the take-off
	Taxiways          []string
	HoldForClearances bool
	// Adopt: the call sign of a parked arrival of ours; its aircraft, on its
	// stand and in its livery, becomes this departure (a turnaround). Stand,
	// Model, Livery and Type are then ignored.
	Adopt string
}

// ArrivalSpec asks for a controlled arrival.
type ArrivalSpec struct {
	Graph            *airport.Graph
	Limits           *airport.Limits
	Callsign         string
	Stand            string
	Runway           string
	Model, Livery    string
	Type             string
	STAR             string             // name, for the view
	Procedure        []airport.NavPoint // STAR and approach; empty: straight in from SpawnNM
	MissedApproach   []airport.NavPoint // flown on a go-around; empty: a circuit
	SpawnNM          float64
	Taxiways         []string
	HoldForClearance bool
}

// FlightView is one of our aircraft as the tools show it.
type FlightView struct {
	Callsign       string         `json:"callsign"`
	Kind           string         `json:"kind"` // departure | arrival
	ICAO           string         `json:"icao"`
	Model          string         `json:"model"`
	Stand          string         `json:"stand"`
	Runway         string         `json:"runway"`
	Entry          string         `json:"entry,omitempty"`
	Procedure      string         `json:"procedure,omitempty"`
	TaxiRoute      []string       `json:"taxi_route,omitempty"`
	State          string         `json:"state"`
	Taxiway        string         `json:"taxiway,omitempty"`
	HoldingShortOf string         `json:"holding_short_of,omitempty"`
	RemainingM     float64        `json:"remaining_m,omitempty"`
	Position       airport.LatLon `json:"position"`
	Heading        float64        `json:"heading"`
	GroundSpeed    float64        `json:"ground_speed_kts"`
	AGLFt          float64        `json:"agl_ft,omitempty"`
	OnGround       bool           `json:"on_ground"`
	Lights         string         `json:"lights,omitempty"`
	Error          string         `json:"error,omitempty"`
	Actions        []string       `json:"actions"`
	Done           bool           `json:"done"`
}

// Traffic IDs, above the loaders' (see runtime_windows.go).
const (
	ctrlDefBase    = idBase + 10_000
	ctrlReqBase    = idBase + 20_000
	ctrlIDBlock    = 10
	ctrlBlocks     = 64
	standDefBase   = idBase + 30_000
	standReqBase   = idBase + 31_000
	standIDBlock   = 4
	scanDefID      = idBase + 40_000
	scanReqID      = idBase + 40_001
	modelsReqID    = idBase + 40_002
	removeReqBase  = idBase + 41_000
	scanRadiusM    = 80_000 // the picture's scan around the user aircraft
	defaultRadius  = 40.0   // NM
	maxFlights     = 32
	modelsSettle   = 3 * time.Second
	liverySep      = "|"
	defaultAirType = "A320"
)

type flight struct {
	ts      *trafficState
	view    FlightView
	dep     *traffic.TaxiController
	arr     *traffic.ArrivalController
	alloc   *traffic.StandAllocator
	stand   int
	left    bool
	defBase uint32
	id      uint32
	// held: spawned holding for every clearance — the runtime's tower and
	// sequencing leave it to the caller (it counts as other traffic).
	held bool
}

// trafficState is the Runtime's traffic side; its fields are guarded by
// Runtime.mu, except flight views (tmu), which event goroutines update.
type trafficState struct {
	client  engine.Client
	fleet   *traffic.Fleet
	inj     *traffic.Injector
	ids     *traffic.IDBlocks
	detail  *traffic.Detail
	picture *traffic.TrafficPicture
	stands  map[string]*traffic.StandAllocator
	scanOn  bool
	scan    []traffic.Observation
	userID  uint32

	models     map[string]bool
	modelsAt   time.Time // last enumeration message
	modelsWait []chan struct{}
	modelsReq  bool

	tmu     sync.Mutex
	flights map[string]*flight // by call sign

	atc *atcState // the towers and landing sequences (atc_windows.go)
}

func newTrafficState(client engine.Client) *trafficState {
	return &trafficState{
		client:  client,
		fleet:   traffic.NewFleet(client),
		inj:     traffic.NewInjector(client),
		ids:     traffic.NewIDBlocks(ctrlDefBase, ctrlReqBase, ctrlIDBlock, ctrlBlocks),
		detail:  traffic.NewDetail(),
		picture: traffic.NewTrafficPicture(traffic.PictureOptions{Centre: traffic.Centre{FollowUser: true}, RadiusNM: defaultRadius}),
		stands:  map[string]*traffic.StandAllocator{},
		models:  map[string]bool{},
		flights: map[string]*flight{},
	}
}

// trafficLocked returns the traffic state, created on first use with the
// connection's engine client (the traffic package drives an engine.Client).
func (r *Runtime) trafficLocked() (*trafficState, error) {
	if r.traffic == nil {
		c := r.mgr.Client()
		if c == nil {
			return nil, ErrNotConnected
		}
		r.traffic = newTrafficState(c)
	}
	return r.traffic, nil
}

// scanRaw is the by-type scan of aircraft for the traffic picture.
type scanRaw struct {
	Title    [256]byte
	AtcID    [32]byte
	Lat, Lon float64
	AGL      float64
	GS       float64
	Heading  float64
	VS       float64
	OnGround float64
	SpanFt   float64
	AltFt    float64
	IsUser   float64
}

var scanVars = []struct{ name, unit string }{
	{"PLANE LATITUDE", "degrees"}, {"PLANE LONGITUDE", "degrees"}, {"PLANE ALT ABOVE GROUND", "feet"},
	{"GROUND VELOCITY", "knots"}, {"PLANE HEADING DEGREES TRUE", "degrees"}, {"VERTICAL SPEED", "feet per minute"},
	{"SIM ON GROUND", "bool"}, {"WING SPAN", "feet"}, {"PLANE ALTITUDE", "feet"}, {"IS USER SIM", "bool"},
}

// startScanLocked defines the aircraft scan; tick requests it every second.
func (r *Runtime) startScanLocked() error {
	t := r.traffic
	if t.scanOn {
		return nil
	}
	if err := r.mgr.AddToDataDefinition(scanDefID, "TITLE", "", types.SIMCONNECT_DATATYPE_STRING256, 0, 0); err != nil {
		return err
	}
	if err := r.mgr.AddToDataDefinition(scanDefID, "ATC ID", "", types.SIMCONNECT_DATATYPE_STRING32, 0, 1); err != nil {
		return err
	}
	for i, v := range scanVars {
		if err := r.mgr.AddToDataDefinition(scanDefID, v.name, v.unit, types.SIMCONNECT_DATATYPE_FLOAT64, 0, uint32(i+2)); err != nil {
			return err
		}
	}
	t.scanOn = true
	return r.mgr.RequestDataOnSimObjectType(scanReqID, scanDefID, scanRadiusM, types.SIMCONNECT_SIMOBJECT_TYPE_AIRCRAFT)
}

// handleTrafficLocked passes a message to the traffic components.
func (r *Runtime) handleTrafficLocked(msg engine.Message) bool {
	t := r.traffic
	if t == nil {
		return false
	}
	for _, a := range t.stands {
		if a.Handle(msg) {
			return true
		}
	}
	switch types.SIMCONNECT_RECV_ID(msg.DwID) {
	case types.SIMCONNECT_RECV_ID_ENUMERATE_SIMOBJECT_AND_LIVERY_LIST:
		if e := msg.AsSimObjectAndLiveryEnumeration(); uint32(e.DwRequestID) == modelsReqID {
			r.addModelsLocked(msg)
			return true
		}
	case types.SIMCONNECT_RECV_ID_SIMOBJECT_DATA_BYTYPE:
		if d := msg.AsSimObjectDataBType(); uint32(d.DwRequestID) == scanReqID {
			r.addScanLocked(d)
			return true
		}
	}
	if ok, _ := t.inj.Handle(msg); ok {
		return true
	}
	t.tmu.Lock()
	fl := make([]*flight, 0, len(t.flights))
	for _, f := range t.flights {
		fl = append(fl, f)
	}
	t.tmu.Unlock()
	for _, f := range fl {
		if f.dep != nil && f.dep.Handle(msg) || f.arr != nil && f.arr.Handle(msg) {
			return true
		}
	}
	return false
}

func (r *Runtime) addScanLocked(d *types.SIMCONNECT_RECV_SIMOBJECT_DATA_BTYPE) {
	t := r.traffic
	s := engine.CastDataAs[scanRaw](&d.DwData)
	id := uint32(d.DwObjectID)
	user := s.IsUser != 0
	if user {
		t.userID = id
		t.detail.SetViewer(airport.LatLon{Lat: s.Lat, Lon: s.Lon})
	}
	t.scan = append(t.scan, traffic.Observation{ObjectID: id, Title: engine.BytesToString(s.Title[:]), Tail: engine.BytesToString(s.AtcID[:]),
		Position: airport.LatLon{Lat: s.Lat, Lon: s.Lon}, AltFt: s.AltFt, AGLFt: s.AGL, GroundKts: s.GS, Heading: s.Heading,
		VSFpm: s.VS, OnGround: s.OnGround != 0, User: user, SpanM: s.SpanFt * 0.3048})
	if uint32(d.DwEntryNumber) >= uint32(d.DwOutOf) || uint32(d.DwOutOf) == 0 {
		t.picture.Observe(time.Now(), t.scan)
		t.scan = nil
	}
}

func (r *Runtime) tickTrafficLocked(now time.Time) {
	t := r.traffic
	if t == nil {
		return
	}
	r.tickATCLocked(now)
	if t.scanOn {
		t.scan = t.scan[:0]
		_ = r.mgr.RequestDataOnSimObjectType(scanReqID, scanDefID, scanRadiusM, types.SIMCONNECT_SIMOBJECT_TYPE_AIRCRAFT)
	}
	if len(t.modelsWait) > 0 && !t.modelsAt.IsZero() && time.Since(t.modelsAt) > modelsSettle {
		for _, w := range t.modelsWait {
			close(w)
		}
		t.modelsWait = nil
	}
}

// ── Models ──────────────────────────────────────────────────────────────────

func (r *Runtime) addModelsLocked(msg engine.Message) {
	t := r.traffic
	e := msg.AsSimObjectAndLiveryEnumeration()
	n := uint32(e.DwArraySize)
	header := uint32(unsafe.Sizeof(types.SIMCONNECT_RECV_LIST_TEMPLATE{}))
	size := uint32(unsafe.Sizeof(types.SIMCONNECT_ENUMERATE_SIMOBJECT_LIVERY{}))
	if n > 0 && n*size <= uint32(msg.DwSize)-header {
		base := unsafe.Add(unsafe.Pointer(e), header)
		for i := uint32(0); i < n; i++ {
			entry := (*types.SIMCONNECT_ENUMERATE_SIMOBJECT_LIVERY)(unsafe.Add(base, i*size))
			if title := engine.BytesToString(entry.AircraftTitle[:]); title != "" {
				if l := engine.BytesToString(entry.LiveryName[:]); l != "" {
					title += liverySep + l
				}
				t.models[title] = true
			}
		}
	}
	t.modelsAt = time.Now()
	if uint32(e.DwEntryNumber)+1 >= uint32(e.DwOutOf) {
		for _, w := range t.modelsWait {
			close(w)
		}
		t.modelsWait = nil
	}
}

// Models implements Traffic. The simulator enumerates its aircraft once.
func (r *Runtime) Models(ctx context.Context) ([]string, error) {
	ch := make(chan struct{})
	r.mu.Lock()
	if !r.connected {
		r.mu.Unlock()
		return nil, ErrNotConnected
	}
	t, err := r.trafficLocked()
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	if t.modelsReq && len(t.modelsWait) == 0 {
		out := r.modelListLocked()
		r.mu.Unlock()
		return out, nil
	}
	if !t.modelsReq {
		if err := r.mgr.EnumerateSimObjectsAndLiveries(modelsReqID, types.SIMCONNECT_SIMOBJECT_TYPE_AIRCRAFT); err != nil {
			r.mu.Unlock()
			return nil, err
		}
		t.modelsReq = true
	}
	t.modelsWait = append(t.modelsWait, ch)
	r.mu.Unlock()

	select {
	case <-ch:
	case <-ctx.Done():
		return nil, fmt.Errorf("aircraft models: %w", ErrTimeout)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.modelListLocked(), nil
}

func (r *Runtime) modelListLocked() []string {
	out := make([]string, 0, len(r.traffic.models))
	for m := range r.traffic.models {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// pickModel returns the title and livery for a flight: the given model, or
// an installed one of the type in the airline's livery.
func (r *Runtime) pickModel(ctx context.Context, model, livery, typ, callsign string) (string, string, error) {
	if model != "" {
		if m, l, ok := strings.Cut(model, liverySep); ok && livery == "" {
			return m, l, nil
		}
		return model, livery, nil
	}
	models, err := r.Models(ctx)
	if err != nil {
		return "", "", err
	}
	if typ == "" {
		typ = defaultAirType
	}
	airline, name := airlineOf(callsign)
	picks := traffic.ModelsFor(models, airline, name, typ, 1)
	if len(picks) == 0 {
		return "", "", fmt.Errorf("no installed aircraft of type %s: give model (list_aircraft_models)", typ)
	}
	m, l, _ := strings.Cut(picks[0], liverySep)
	return m, l, nil
}

// airlineOf is the airline ICAO code and name of a call sign ("CSA123").
func airlineOf(callsign string) (string, string) {
	if len(callsign) < 3 {
		return "", ""
	}
	code := strings.ToUpper(callsign[:3])
	for _, a := range traffic.DefaultScheduleConfig().Airlines {
		if a.ICAO == code {
			return code, a.Name
		}
	}
	return code, ""
}

// ── Spawning ────────────────────────────────────────────────────────────────

func (r *Runtime) allocatorLocked(g *airport.Graph) *traffic.StandAllocator {
	t := r.traffic
	if a := t.stands[g.Layout.ICAO]; a != nil {
		return a
	}
	k := uint32(len(t.stands))
	a := traffic.NewStandAllocator(t.client, g, traffic.StandWithIDs(standDefBase+k*standIDBlock, standReqBase+k*standIDBlock))
	t.stands[g.Layout.ICAO] = a
	t.picture.Allocate(g.Layout.ICAO, a) // fed from the scans
	return a
}

// prepare checks the call sign, starts the scan, takes an ID block and the
// stand; undo gives them back if the spawn fails.
func (r *Runtime) prepareLocked(g *airport.Graph, callsign, standLabel, runway, kind string, halfSpan float64) (alloc *traffic.StandAllocator, stand int, defBase, reqBase uint32, undo func(), err error) {
	t, err := r.trafficLocked()
	if err != nil {
		return nil, 0, 0, 0, nil, err
	}
	t.tmu.Lock()
	_, dup := t.flights[callsign]
	n := len(t.flights)
	t.tmu.Unlock()
	if dup {
		return nil, 0, 0, 0, nil, fmt.Errorf("%s is already one of ours", callsign)
	}
	if n >= maxFlights {
		return nil, 0, 0, 0, nil, fmt.Errorf("at most %d aircraft of ours at once; remove some", maxFlights)
	}
	if err := r.startScanLocked(); err != nil {
		return nil, 0, 0, 0, nil, err
	}
	alloc = r.allocatorLocked(g)
	if standLabel == "" {
		req := traffic.StandRequirements{Owner: callsign, HalfSpan: halfSpan}
		req.Airline, _ = airlineOf(callsign)
		if kind == "arrival" {
			req.Runway = runway
		}
		if stand, err = alloc.Assign(req); err != nil {
			return nil, 0, 0, 0, nil, err
		}
	} else {
		if stand, err = g.Layout.ParkingIndex(standLabel); err != nil {
			return nil, 0, 0, 0, nil, err
		}
		if err = alloc.Occupy(stand, callsign, halfSpan); err != nil {
			return nil, 0, 0, 0, nil, err
		}
	}
	if defBase, reqBase, err = t.ids.Acquire(); err != nil {
		alloc.ReleaseOwner(callsign)
		return nil, 0, 0, 0, nil, err
	}
	undo = func() {
		alloc.ReleaseOwner(callsign)
		t.ids.Release(defBase)
	}
	return alloc, stand, defBase, reqBase, undo, nil
}

// SpawnDeparture implements Traffic.
func (r *Runtime) SpawnDeparture(ctx context.Context, s DepartureSpec) (FlightView, error) {
	var model, livery string
	if s.Adopt == "" {
		var err error
		if model, livery, err = r.pickModel(ctx, s.Model, s.Livery, s.Type, s.Callsign); err != nil {
			return FlightView{}, err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.connected {
		return FlightView{}, ErrNotConnected
	}
	g := s.Graph
	var adopt *flight
	var objectID uint32
	if s.Adopt != "" {
		var err error
		if adopt, err = r.adoptableLocked(s.Adopt, g); err != nil {
			return FlightView{}, err
		}
		model, livery, _ = strings.Cut(adopt.view.Model, liverySep)
		s.Stand, objectID = adopt.view.Stand, adopt.id
		adopt.alloc.ReleaseOwner(s.Adopt) // the stand passes to the departure
	}
	ac := traffic.ProfileFor(model)
	// A failed turnaround leaves the arrival on its stand.
	keep := func() {
		if adopt != nil {
			_ = adopt.alloc.Occupy(adopt.stand, s.Adopt, ac.Motion.SpanMeters/2)
		}
	}
	alloc, stand, defBase, reqBase, undo, err := r.prepareLocked(g, s.Callsign, s.Stand, s.Runway, "departure", ac.Motion.SpanMeters/2)
	if err != nil {
		keep()
		return FlightView{}, err
	}
	t := r.traffic
	ctl := traffic.NewTaxiController(t.fleet, traffic.TaxiWithIDs(defBase, reqBase), traffic.TaxiWithInjector(t.inj),
		traffic.TaxiWithDetail(t.detail), traffic.TaxiWithGroundPicture(t.picture.Ground(g.Layout.ICAO)))
	if err := ctl.Start(traffic.TaxiRequest{Graph: g, Parking: stand, Runway: s.Runway, Entry: s.Entry,
		Options: airport.RouteOptions{Taxiways: s.Taxiways}, Model: model, Livery: livery, Tail: s.Callsign,
		HoldForClearances: s.HoldForClearances, HoldForRunway: !s.HoldForClearances, Profile: ac.Motion, Aircraft: &ac,
		Departure: s.Departure, Airport: s.Limits, ObjectID: objectID}); err != nil {
		undo()
		keep()
		return FlightView{}, err
	}
	if adopt != nil {
		t.tmu.Lock()
		delete(t.flights, s.Adopt) // the same aircraft flies on as the departure
		t.tmu.Unlock()
	}
	f := &flight{ts: t, dep: ctl, alloc: alloc, stand: stand, defBase: defBase, held: s.HoldForClearances, view: FlightView{Callsign: s.Callsign, Kind: "departure",
		ICAO: g.Layout.ICAO, Model: joinModel(model, livery), Stand: g.Layout.Parking[stand].Label(), Runway: s.Runway, Entry: s.Entry,
		Procedure: s.SID, State: "spawning", Actions: []string{"remove"}}}
	if rt := ctl.Route(); rt != nil {
		f.view.TaxiRoute = rt.Taxiways
		alloc.ReserveRoute(s.Callsign, rt.Nodes)
	}
	r.addFlight(f, func(yield func(FlightView) bool) {
		for ev := range ctl.Events() {
			if !yield(f.departureEvent(r, ev)) {
				return
			}
		}
	})
	return f.view, nil
}

// adoptableLocked returns our arrival parked at g's airport, for a turnaround.
func (r *Runtime) adoptableLocked(callsign string, g *airport.Graph) (*flight, error) {
	t, err := r.trafficLocked()
	if err != nil {
		return nil, err
	}
	t.tmu.Lock()
	defer t.tmu.Unlock()
	f := t.flights[callsign]
	switch {
	case f == nil:
		return nil, fmt.Errorf("turnaround of %s: %w", callsign, ErrUnknownFlight)
	case f.arr == nil || f.view.State != "parked" || f.id == 0:
		return nil, fmt.Errorf("turnaround of %s: it is a %s, %s, not an arrival parked on its stand", callsign, f.view.Kind, f.view.State)
	case f.view.ICAO != g.Layout.ICAO:
		return nil, fmt.Errorf("turnaround of %s: it is parked at %s, not %s", callsign, f.view.ICAO, g.Layout.ICAO)
	}
	return f, nil
}

// SpawnArrival implements Traffic.
func (r *Runtime) SpawnArrival(ctx context.Context, s ArrivalSpec) (FlightView, error) {
	model, livery, err := r.pickModel(ctx, s.Model, s.Livery, s.Type, s.Callsign)
	if err != nil {
		return FlightView{}, err
	}
	ac := traffic.ProfileFor(model)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.connected {
		return FlightView{}, ErrNotConnected
	}
	g := s.Graph
	alloc, stand, defBase, reqBase, undo, err := r.prepareLocked(g, s.Callsign, s.Stand, s.Runway, "arrival", ac.Motion.SpanMeters/2)
	if err != nil {
		return FlightView{}, err
	}
	t := r.traffic
	ctl := traffic.NewArrivalController(t.fleet, traffic.ArrivalWithIDs(defBase, reqBase), traffic.ArrivalWithInjector(t.inj),
		traffic.ArrivalWithDetail(t.detail), traffic.ArrivalWithGroundPicture(t.picture.Ground(g.Layout.ICAO)))
	if err := ctl.Start(traffic.ArrivalRequest{Graph: g, Runway: s.Runway, Parking: stand, Model: model, Livery: livery, Tail: s.Callsign,
		SpawnNm: s.SpawnNM, Options: airport.RouteOptions{Taxiways: s.Taxiways}, HoldForClearance: s.HoldForClearance,
		HoldAtCrossings: true, InjectApproach: true, Profile: ac.Motion, Procedure: s.Procedure, MissedApproach: s.MissedApproach,
		Aircraft: &ac, Airport: s.Limits}); err != nil {
		undo()
		return FlightView{}, err
	}
	f := &flight{ts: t, arr: ctl, alloc: alloc, stand: stand, defBase: defBase, held: s.HoldForClearance, view: FlightView{Callsign: s.Callsign, Kind: "arrival",
		ICAO: g.Layout.ICAO, Model: joinModel(model, livery), Stand: g.Layout.Parking[stand].Label(), Runway: s.Runway,
		Procedure: s.STAR, State: "spawning", Actions: []string{"remove"}}}
	if p := ctl.Plan(); p != nil && p.Route != nil {
		f.view.TaxiRoute = p.Route.Taxiways
		alloc.ReserveRoute(s.Callsign, p.Route.Nodes)
	}
	r.addFlight(f, func(yield func(FlightView) bool) {
		for ev := range ctl.Events() {
			if !yield(f.arrivalEvent(r, ev)) {
				return
			}
		}
	})
	return f.view, nil
}

func joinModel(model, livery string) string {
	if livery == "" {
		return model
	}
	return model + liverySep + livery
}

// addFlight registers f and follows its events until the controller ends.
func (r *Runtime) addFlight(f *flight, events func(yield func(FlightView) bool)) {
	t := f.ts
	t.tmu.Lock()
	t.flights[f.view.Callsign] = f
	t.tmu.Unlock()
	go func() {
		for v := range events {
			t.tmu.Lock()
			f.view = v
			t.tmu.Unlock()
		}
		t.tmu.Lock()
		f.view.Done, f.view.Actions = true, []string{"remove"}
		t.tmu.Unlock()
		r.mu.Lock()
		t.ids.Release(f.defBase)
		r.mu.Unlock()
	}()
}

// departureEvent turns a TaxiEvent into the flight's view.
func (f *flight) departureEvent(r *Runtime, e traffic.TaxiEvent) FlightView {
	t := f.ts
	t.tmu.Lock()
	v := f.view
	t.tmu.Unlock()
	v.State, v.Taxiway, v.HoldingShortOf, v.RemainingM = e.State.String(), e.Taxiway, e.HoldingShortOf, e.Remaining
	v.Position, v.Heading, v.GroundSpeed, v.OnGround, v.Lights = e.Position, e.Heading, e.GroundSpeed, e.OnGround, e.Lights.String()
	v.AGLFt = e.HeightFt
	if e.Err != nil {
		v.Error = e.Err.Error()
	}
	v.Actions = departureActions(e.State, e.HoldingShortOf, f.view.Runway)
	if e.ObjectID != 0 {
		f.id = e.ObjectID
	}
	// Off the stand once pushed or taxiing; the route is released when airborne.
	if !f.left && e.State >= traffic.TaxiPushback && e.State != traffic.TaxiAwaitingTaxi {
		f.left = true
		f.alloc.Release(f.stand)
	}
	if e.State.Terminal() {
		f.alloc.ReleaseRoute(v.Callsign)
	}
	t.reportOwn(f.id, e.State.Terminal(), departurePhase(e), v.ICAO)
	return v
}

func (f *flight) arrivalEvent(r *Runtime, e traffic.ArrivalEvent) FlightView {
	t := f.ts
	t.tmu.Lock()
	v := f.view
	t.tmu.Unlock()
	v.State, v.Taxiway, v.HoldingShortOf, v.RemainingM = e.State.String(), e.Taxiway, e.HoldingShortOf, e.Remaining
	v.Position, v.Heading, v.GroundSpeed, v.OnGround, v.Lights = e.Position, e.Heading, e.GroundSpeed, e.OnGround, e.Lights.String()
	v.AGLFt = e.AGL
	if e.Err != nil {
		v.Error = e.Err.Error()
	}
	v.Actions = arrivalActions(e.State)
	if e.ObjectID != 0 {
		f.id = e.ObjectID
	}
	switch e.State {
	case traffic.ArrivalParked:
		f.alloc.ReleaseRoute(v.Callsign)
	case traffic.ArrivalCancelled, traffic.ArrivalFailed:
		f.alloc.ReleaseOwner(v.Callsign)
	}
	t.reportOwn(f.id, e.State.Terminal() && e.State != traffic.ArrivalParked, arrivalPhase(e), v.ICAO)
	return v
}

// reportOwn tells the picture about one of ours; done hands it back to
// what the scans see.
func (t *trafficState) reportOwn(id uint32, done bool, phase traffic.Phase, icao string) {
	if id == 0 {
		return
	}
	if done {
		t.picture.ForgetOwn(id)
		return
	}
	t.picture.SetOwn(id, phase, icao)
}

// currentTraffic returns the traffic state, nil before the first use.
func (r *Runtime) currentTraffic() *trafficState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.traffic
}

func departurePhase(e traffic.TaxiEvent) traffic.Phase {
	switch e.State {
	case traffic.TaxiSpawning, traffic.TaxiAwaitingPushback:
		return traffic.PhaseParked
	case traffic.TaxiLiningUp, traffic.TaxiLinedUp:
		return traffic.PhaseRunway
	case traffic.TaxiDeparting:
		if !e.OnGround {
			return traffic.PhaseDeparting
		}
		return traffic.PhaseRunway
	}
	return traffic.PhaseTaxiing
}

func arrivalPhase(e traffic.ArrivalEvent) traffic.Phase {
	switch e.State {
	case traffic.ArrivalSpawning, traffic.ArrivalApproaching:
		return traffic.PhaseArriving
	case traffic.ArrivalLanding, traffic.ArrivalRollout:
		return traffic.PhaseRunway
	case traffic.ArrivalParked:
		return traffic.PhaseParked
	}
	return traffic.PhaseTaxiing
}

// ── Clearances ─────────────────────────────────────────────────────────────

func departureActions(s traffic.TaxiState, holdingShortOf, runway string) []string {
	var a []string
	switch s {
	case traffic.TaxiAwaitingPushback:
		a = []string{"pushback", "taxi"}
	case traffic.TaxiPushback, traffic.TaxiAwaitingTaxi:
		a = []string{"taxi", "takeoff"}
	case traffic.TaxiTaxiing:
		a = []string{"hold", "taxi", "takeoff"}
	case traffic.TaxiHoldingShort:
		if holdingShortOf != "" && !strings.Contains(holdingShortOf, runway) {
			a = []string{"cross", "taxi"}
		} else {
			a = []string{"lineup", "takeoff"}
		}
	case traffic.TaxiLiningUp, traffic.TaxiLinedUp:
		a = []string{"takeoff", "abort"}
	case traffic.TaxiDeparting:
		a = []string{"abort"}
	}
	return append(a, "remove")
}

func arrivalActions(s traffic.ArrivalState) []string {
	var a []string
	switch s {
	case traffic.ArrivalApproaching, traffic.ArrivalLanding:
		a = []string{"goaround", "taxi"}
	case traffic.ArrivalRollout, traffic.ArrivalVacating, traffic.ArrivalAwaitingTaxi:
		a = []string{"taxi"}
	case traffic.ArrivalTaxiing:
		a = []string{"hold", "taxi"}
	case traffic.ArrivalHoldingShort:
		a = []string{"cross", "taxi"}
	}
	return append(a, "remove")
}

// ErrUnknownFlight is returned for a call sign that is not one of ours.
var ErrUnknownFlight = errors.New("not one of our aircraft")

// Clear implements Traffic.
func (r *Runtime) Clear(callsign, action string) (FlightView, error) {
	t := r.currentTraffic()
	if t == nil {
		return FlightView{}, fmt.Errorf("%s: %w", callsign, ErrUnknownFlight)
	}
	t.tmu.Lock()
	f := t.flights[callsign]
	t.tmu.Unlock()
	if f == nil {
		return FlightView{}, fmt.Errorf("%s: %w", callsign, ErrUnknownFlight)
	}
	var err error
	switch d, a := f.dep, f.arr; {
	case action == "remove":
		err = r.removeFlight(f)
	case d != nil && action == "pushback":
		d.ClearPushback()
	case d != nil && action == "taxi":
		d.ClearToTaxi()
	case d != nil && action == "cross":
		d.ClearToCross()
	case d != nil && action == "lineup":
		d.ClearToLineUp()
	case d != nil && action == "takeoff":
		err = d.ClearForTakeoff()
	case d != nil && action == "hold":
		err = d.HoldPosition()
	case d != nil && action == "abort":
		err = d.AbortTakeoff()
	case a != nil && action == "goaround":
		err = a.GoAround()
	case a != nil && action == "taxi":
		a.ClearToTaxi()
	case a != nil && action == "cross":
		a.ClearToCross()
	case a != nil && action == "hold":
		err = a.HoldPosition()
	default:
		t.tmu.Lock()
		v := f.view
		t.tmu.Unlock()
		return v, fmt.Errorf("%s (%s, %s) takes %s, not %q", callsign, v.Kind, v.State, strings.Join(v.Actions, ", "), action)
	}
	t.tmu.Lock()
	v := f.view
	t.tmu.Unlock()
	return v, err
}

// removeFlight takes the aircraft out of the simulator and forgets it.
func (r *Runtime) removeFlight(f *flight) error {
	var err error
	if f.dep != nil {
		err = f.dep.Cancel()
	} else {
		err = f.arr.Cancel()
	}
	f.alloc.ReleaseOwner(f.view.Callsign)
	t := f.ts
	if f.id != 0 {
		t.picture.ForgetOwn(f.id) // gone now, not at the next scan
	}
	t.tmu.Lock()
	delete(t.flights, f.view.Callsign)
	t.tmu.Unlock()
	return err
}

// Flights implements Traffic.
func (r *Runtime) Flights() []FlightView {
	t := r.currentTraffic()
	if t == nil {
		return []FlightView{}
	}
	t.tmu.Lock()
	defer t.tmu.Unlock()
	out := make([]FlightView, 0, len(t.flights))
	for _, f := range t.flights {
		out = append(out, f.view)
	}
	slices.SortFunc(out, func(a, b FlightView) int { return strings.Compare(a.Callsign, b.Callsign) })
	return out
}

// RemoveAll takes every aircraft of ours out of the simulator and returns
// how many there were.
func (r *Runtime) RemoveAll() int {
	t := r.currentTraffic()
	if t == nil {
		return 0
	}
	t.tmu.Lock()
	fl := make([]*flight, 0, len(t.flights))
	for _, f := range t.flights {
		fl = append(fl, f)
	}
	t.tmu.Unlock()
	for _, f := range fl {
		_ = r.removeFlight(f)
	}
	return len(fl)
}

// ── Picture ────────────────────────────────────────────────────────────────

// Picture implements Traffic. The first call starts the scan and waits a
// couple of seconds for it.
func (r *Runtime) Picture(ctx context.Context, centre string, radiusNM float64) ([]traffic.TrackedAircraft, error) {
	r.mu.Lock()
	if !r.connected {
		r.mu.Unlock()
		return nil, ErrNotConnected
	}
	t, err := r.trafficLocked()
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}
	first := !t.scanOn
	if err := r.startScanLocked(); err != nil {
		r.mu.Unlock()
		return nil, err
	}
	r.mu.Unlock()
	if radiusNM <= 0 {
		radiusNM = defaultRadius
	}
	c := traffic.Centre{FollowUser: true}
	if centre != "" {
		l, err := r.Layout(ctx, centre)
		if err != nil {
			return nil, err
		}
		c = traffic.Centre{ICAO: l.ICAO, Position: airport.LatLon{Lat: l.Latitude, Lon: l.Longitude}}
	}
	t.picture.SetCentre(c)
	t.picture.SetRadius(radiusNM)
	if first {
		select {
		case <-time.After(2500 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return t.picture.Aircraft(), nil
}
