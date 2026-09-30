//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// Scheduled traffic: the library's TrafficManager runs an airline schedule
// for airports in the simulator — departures board on their stands before
// their STD and push at it, arrivals appear at a STAR entry in time for
// their STA; departed and parked aircraft are removed. An arrival of an
// airline and type that departs again 40 minutes to 3 hours later turns
// around: it stays on its stand and becomes that departure. The flights are
// ours and not held for clearances, so the runtime's tower and landing
// sequences clear and sequence them. Arrivals fly en route before their
// STAR entry, and overflights cross the area (live_schedule_enroute.go).

// scheduleRunner is the TrafficManager and the Spawner it spawns through.
type scheduleRunner struct {
	src live.Source
	tr  live.Traffic

	mu      sync.Mutex
	mgr     *traffic.TrafficManager
	stop    chan struct{}
	status  map[string]traffic.FlightStatus // last reported, by call sign
	clock   func() time.Time                // time.Now; tests set it
	enroute map[string]*enrouteFlight       // en route flights by call sign
	cleared bool                            // stop_schedule removed the aircraft (until the next start)
	// settings: what the manager's sources read, under its lock (not mu:
	// start holds mu while it calls the manager).
	settings atomic.Pointer[scheduleSettings]
}

// scheduleSettings are start_schedule's: density, seed, the airports and
// the first one's position (the overflights' area; zero: none).
type scheduleSettings struct {
	density  float64
	seed     uint64
	airports []string
	centre   airport.LatLon
}

// The manager's settings for the MCP: at most maxScheduled at once (the runtime keeps 32).
const (
	maxScheduled       = 24
	scheduleTick       = time.Second
	defaultMaxPerField = 12
)

func newScheduleRunner(src live.Source, tr live.Traffic) *scheduleRunner {
	return &scheduleRunner{src: src, tr: tr, status: map[string]traffic.FlightStatus{}, clock: time.Now, enroute: map[string]*enrouteFlight{}}
}

// Spawn implements traffic.Spawner: in the background, the way
// spawn_departure and spawn_arrival would, not held for clearances; en
// route in the air on its flight plan.
func (r *scheduleRunner) Spawn(f traffic.ManagedFlight) {
	go func() {
		r.mu.Lock()
		mgr := r.mgr
		r.mu.Unlock()
		if mgr == nil {
			return
		}
		if f.Stage == "enroute" {
			v, err := r.spawnEnroute(context.Background(), f, mgr.Options().ArrivalLead)
			if err != nil {
				mgr.Failed(f.Callsign, err, r.clock())
				return
			}
			mgr.Describe(f.Callsign, v.Model, "", "")
			return
		}
		args := map[string]any{"icao": f.Airport, "callsign": f.Callsign, "aircraft_type": f.Type}
		var v live.FlightView
		var bad *mcpadapter.CallToolResult
		if f.Kind == "departure" {
			args["hold_for_clearances"] = false
			if f.TurnFrom != "" {
				args["turnaround_of"] = f.TurnFrom
			}
			v, bad = spawnDepartureFrom(context.Background(), r.src, r.tr, args)
		} else {
			args["hold_for_clearance"] = false
			v, bad = spawnArrivalFrom(context.Background(), r.src, r.tr, args)
		}
		now := r.clock()
		if bad != nil {
			mgr.Failed(f.Callsign, errors.New(resultText(bad)), now)
			return
		}
		mgr.Describe(f.Callsign, v.Model, v.Stand, v.Runway)
	}()
}

// Remove implements traffic.Spawner.
func (r *scheduleRunner) Remove(f traffic.ManagedFlight) {
	_, _ = r.tr.Clear(f.Callsign, "remove")
}

// resultText is an error result's text.
func resultText(res *mcpadapter.CallToolResult) string {
	for _, c := range res.Content {
		if c.Text != "" {
			return c.Text
		}
	}
	return "spawn failed"
}

// flightStatus is the manager's status for one of our flights' state.
func flightStatus(v live.FlightView) (traffic.FlightStatus, bool, error) {
	switch v.State {
	case "failed", "cancelled":
		if v.Error != "" {
			return 0, false, errors.New(v.Error)
		}
		return 0, false, errors.New(v.State)
	case "enroute":
		return traffic.FlightEnroute, true, nil
	}
	if v.Kind == "departure" {
		switch v.State {
		case "awaiting pushback":
			return traffic.FlightBoarding, true, nil
		case "pushback", "awaiting taxi", "taxiing", "holding short", "lining up", "lined up":
			return traffic.FlightTaxiing, true, nil
		case "departing":
			return traffic.FlightDeparting, true, nil
		case "complete":
			return traffic.FlightDeparted, true, nil
		}
		return 0, false, nil
	}
	switch v.State {
	case "approaching", "landing":
		return traffic.FlightApproaching, true, nil
	case "rollout", "vacating", "awaiting taxi", "taxiing", "holding short", "parking":
		return traffic.FlightLanded, true, nil
	case "parked":
		return traffic.FlightParked, true, nil
	}
	return 0, false, nil
}

// tick reports our flights' states to the manager and runs it.
func (r *scheduleRunner) tick(now time.Time) {
	r.mu.Lock()
	mgr := r.mgr
	r.mu.Unlock()
	if mgr == nil {
		return
	}
	ours := map[string]bool{}
	for _, f := range mgr.Flights() {
		ours[f.Callsign] = true
	}
	views := map[string]live.FlightView{}
	for _, v := range r.tr.Flights() {
		views[v.Callsign] = v
		if !ours[v.Callsign] {
			continue // spawned by hand
		}
		s, ok, err := flightStatus(v)
		if err != nil {
			mgr.Failed(v.Callsign, err, now)
			// Out of the way of a retry under the same call sign.
			_, _ = r.tr.Clear(v.Callsign, "remove")
			continue
		}
		r.mu.Lock()
		changed := ok && r.status[v.Callsign] != s
		if changed {
			r.status[v.Callsign] = s
		}
		r.mu.Unlock()
		if changed {
			mgr.Update(v.Callsign, s, now)
		}
	}
	r.handovers(now, views)
	mgr.Tick(now)
}

// start runs the schedule for airports, creating the manager on first use;
// centre is the first airport's position (zero: no overflights).
func (r *scheduleRunner) start(airports []string, centre airport.LatLon, density float64, seed uint64, maxAircraft int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings.Store(&scheduleSettings{density: density, seed: seed, airports: airports, centre: centre})
	r.cleared = false
	if r.mgr == nil {
		cfg := traffic.DefaultScheduleConfig()
		r.mgr = traffic.NewTrafficManager(r, traffic.ManagerOptions{
			Source: func(from, to time.Time, airports []string) []traffic.Flight {
				// Under the manager's lock: reads the runner's settings only.
				s := r.settings.Load()
				return traffic.Schedule(cfg, traffic.ScheduleOptions{Focus: airports, Density: s.density, Seed: s.seed}, from, to)
			},
			Overflights: func(from, to time.Time) []traffic.Flight {
				return r.overflights(cfg, from, to)
			},
			MaxAircraft:   maxAircraft,
			MaxPerAirport: min(maxAircraft, defaultMaxPerField),
		}, airports...)
	} else {
		r.mgr.SetAirports(airports...)
		r.mgr.SetLimits(maxAircraft, min(maxAircraft, defaultMaxPerField))
	}
	r.mgr.SetEnabled(true)
	if r.stop == nil {
		r.stop = make(chan struct{})
		go func(stop chan struct{}) {
			t := time.NewTicker(scheduleTick)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					r.tick(r.clock())
					if r.drained(stop) {
						return
					}
				}
			}
		}(r.stop)
	}
}

// halt stops spawning. With remove the schedule's aircraft go now and it
// stops; otherwise it runs on until they have departed, parked or left
// (en route arrivals are still handed over), and flying is how many that is.
func (r *scheduleRunner) halt(remove bool) (removed, flying int) {
	r.mu.Lock()
	mgr := r.mgr
	if remove {
		r.stopLocked()
		r.cleared = true
	}
	r.mu.Unlock()
	if mgr == nil {
		return 0, 0
	}
	mgr.SetEnabled(false)
	if !remove {
		return 0, mgr.Active()
	}
	ours := map[string]bool{}
	for _, v := range r.tr.Flights() {
		ours[v.Callsign] = true
	}
	now := r.clock()
	for _, f := range mgr.Flights() {
		if f.Status >= traffic.FlightSpawning && f.Status <= traffic.FlightParked { // in the simulator
			if ours[f.Callsign] {
				removed++
			}
			mgr.Remove(f.Callsign, now) // done; its aircraft goes through Remove above
		}
	}
	return removed, 0
}

// stopLocked stops the ticker.
func (r *scheduleRunner) stopLocked() {
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
}

// stopped reports whether the ticker is stopped.
func (r *scheduleRunner) stopped() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stop == nil
}

// wasCleared reports whether stop_schedule removed the schedule's aircraft.
func (r *scheduleRunner) wasCleared() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cleared
}

// drained stops the ticker (stop) once spawning is off and none of the
// schedule's aircraft is left.
func (r *scheduleRunner) drained(stop chan struct{}) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stop != stop {
		return true // stopped, or another ticker runs
	}
	if r.mgr.Enabled() || r.mgr.Active() > 0 {
		return false
	}
	r.stopLocked()
	return true
}

// shutdown stops the ticker; the runtime removes our aircraft.
func (r *scheduleRunner) shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopLocked()
}

// RegisterLiveScheduleTools registers start_schedule, stop_schedule and
// get_schedule.
func RegisterLiveScheduleTools(mcp *mcpadapter.Server, src live.Source, tr live.Traffic) func() {
	r := newScheduleRunner(src, tr)
	registerStartSchedule(mcp, r)
	registerStopSchedule(mcp, r)
	registerGetSchedule(mcp, r)
	return r.shutdown
}

func registerStartSchedule(mcp *mcpadapter.Server, r *scheduleRunner) {
	tool := mcpadapter.NewTool("start_schedule").
		Description("Run a realistic airline schedule at airports in the simulator (this adds and removes aircraft): " +
			"departures appear on their stands 10 minutes before their STD and push at it, arrivals appear in the air on their " +
			"flight plan 45 minutes before their STA and are handed to an arrival controller at their STAR entry (at the entry " +
			"25 minutes before when they cannot fly en route), and overflights cross within 100 NM of the first airport; the tower and landing sequence clear and sequence them (see get_landing_sequence), " +
			"and departed and parked aircraft are removed. An arrival whose airline and type depart again 40 min to 3 h " +
			"later turns around on its stand into that departure. Airlines, types, routes and time-of-day waves as generate_schedule. " +
			"Calling it again changes the airports and settings. The airports must be loaded around the user aircraft.").
		StringParam("airports", "Airports, e.g. \"LKPR\" or \"LKPR, LKTB\" (required).").
		NumberParam("density", "Traffic density, 0.1–3 (default 1).").
		NumberParam("max_aircraft", "Most aircraft of the schedule at once, 1–24 (default 12).").
		NumberParam("seed", "Random seed (default 1).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		var airports []string
		for _, a := range listArg(args, "airports") {
			a = strings.ToUpper(a)
			if !airportICAORe.MatchString(a) {
				return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %q is not an airport ICAO code", a)), nil
			}
			airports = append(airports, a)
		}
		if len(airports) == 0 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: airports is required"), nil
		}
		density, maxAc := numArg(args, "density", 1), int(numArg(args, "max_aircraft", 12))
		if density < 0.1 || density > 3 || maxAc < 1 || maxAc > maxScheduled {
			return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: density 0.1–3, max_aircraft 1–%d", maxScheduled)), nil
		}
		var centre airport.LatLon
		if l, err := r.src.Layout(ctx, airports[0]); err == nil {
			centre = airport.LatLon{Lat: l.Latitude, Lon: l.Longitude}
		}
		r.start(airports, centre, density, uint64(numArg(args, "seed", 1)), maxAc)
		return scheduleView(r, "")
	})
}

func registerStopSchedule(mcp *mcpadapter.Server, r *scheduleRunner) {
	tool := mcpadapter.NewTool("stop_schedule").
		Description("Stop the airline schedule: no more aircraft appear. With remove=true the schedule's aircraft are also " +
			"taken out of the simulator now; otherwise those in the simulator fly on (en route arrivals are still handed to " +
			"the arrival controller) and are removed as they depart, park or leave the area. Returns removed and flying_on.").
		BoolParam("remove", "Also remove the schedule's aircraft now (default false).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		remove, _ := args["remove"].(bool)
		removed, flying := r.halt(remove)
		return mcpadapter.JSONResult(map[string]any{"running": false, "removed": removed, "flying_on": flying})
	})
}

func registerGetSchedule(mcp *mcpadapter.Server, r *scheduleRunner) {
	tool := mcpadapter.NewTool("get_schedule").
		Description("The running schedule's boards: for each airport, departures and arrivals with call sign, type, route, " +
			"STD/STA (UTC), the estimate, status (scheduled, spawning, boarding, taxiing, departing, departed, approaching, " +
			"landed, parked, done, cancelled), stand and runway, and why a flight is delayed or cancelled.").
		StringParam("icao", "Only this airport (default: all the schedule's).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		return scheduleView(r, strings.ToUpper(strings.TrimSpace(strArg(args, "icao"))))
	})
}

// scheduleView is the schedule's state and boards.
func scheduleView(r *scheduleRunner, icao string) (*mcpadapter.CallToolResult, error) {
	r.mu.Lock()
	mgr, running := r.mgr, r.stop != nil
	r.mu.Unlock()
	if mgr == nil {
		return mcpadapter.JSONResult(map[string]any{"running": false, "airports": []string{}, "boards": []any{}})
	}
	type row struct {
		Callsign string `json:"callsign"`
		Type     string `json:"type"`
		From     string `json:"from"`
		To       string `json:"to"`
		Time     string `json:"scheduled"`
		Estimate string `json:"estimate,omitempty"`
		Status   string `json:"status"`
		Stand    string `json:"stand,omitempty"`
		Runway   string `json:"runway,omitempty"`
		Note     string `json:"note,omitempty"`
	}
	hhmm := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format("15:04Z")
	}
	conv := func(fs []traffic.ManagedFlight, dep bool) []row {
		out := []row{}
		for _, f := range fs {
			t := f.STA
			if dep {
				t = f.STD
			}
			est := ""
			if !f.Estimated.IsZero() && !f.Estimated.Equal(t) {
				est = hhmm(f.Estimated)
			}
			out = append(out, row{f.Callsign, f.Type, f.Origin, f.Destination, hhmm(t), est, f.Status.String(), f.Stand, f.Runway, f.Note})
		}
		return out
	}
	type board struct {
		ICAO       string `json:"icao"`
		Departures []row  `json:"departures"`
		Arrivals   []row  `json:"arrivals"`
	}
	boards := []board{}
	for _, a := range mgr.Airports() {
		if icao != "" && a != icao {
			continue
		}
		d, ar := mgr.Board(a)
		boards = append(boards, board{a, conv(d, true), conv(ar, false)})
	}
	over := []row{}
	for _, f := range mgr.Flights() {
		if f.Overflight() {
			over = append(over, row{f.Callsign, f.Type, f.Origin, f.Destination, hhmm(f.Enter), "", f.Status.String(), "", "", f.Note})
		}
	}
	return mcpadapter.JSONResult(map[string]any{"running": running && mgr.Enabled(), "airports": mgr.Airports(),
		"active": mgr.Active(), "max_aircraft": mgr.Options().MaxAircraft, "boards": boards, "overflights": over})
}
