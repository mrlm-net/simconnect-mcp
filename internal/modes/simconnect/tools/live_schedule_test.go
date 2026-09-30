//go:build windows

package tools

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// testClock is the runner's clock in tests: the test moves it while spawn
// goroutines read it.
type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) add(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
	return c.t
}

// The schedule spawns LKPR's flights through the spawn tools' code, not
// held for clearances, and follows their states into the manager.
func TestScheduleRunner(t *testing.T) {
	fx, err := live.NewFixture("../../../live/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ft := &live.FixtureTraffic{ModelList: []string{"FSLTL A320 CSA Czech Airlines", "FSLTL B738 TVS Smartwings"}}
	r := newScheduleRunner(fx, ft)
	clock := &testClock{t: time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)} // the morning wave
	r.clock = clock.now
	now := clock.now()
	r.start([]string{"LKPR"}, airport.LatLon{}, 2, 1, 6)
	defer r.halt(false)
	// Ticks over twenty minutes, a spawn goroutine's time between them.
	deps, arrs := 0, 0
	for i := 0; i < 20*60 && deps == 0; i += 10 {
		now = clock.add(10 * time.Second)
		r.tick(now)
		time.Sleep(20 * time.Millisecond)
		deps, arrs = ft.Spawned()
	}
	if deps == 0 {
		t.Fatalf("no departure spawned in 20 minutes (arrivals %d)", arrs)
	}
	if ft.Departures[0].HoldForClearances {
		t.Error("a scheduled departure holds for clearances: the tower would never clear it")
	}
	cs := ft.Departures[0].Callsign
	ft.SetState(cs, "awaiting pushback")
	r.tick(now.Add(time.Second))
	found := false
	for _, f := range r.mgr.Flights() {
		if f.Callsign == cs {
			found = true
			if f.Status != traffic.FlightBoarding {
				t.Errorf("%s: %v, want boarding", cs, f.Status)
			}
		}
	}
	if !found {
		t.Fatalf("%s not on the manager's boards", cs)
	}
	res, err := scheduleView(r, "LKPR")
	if err != nil || res.IsError {
		t.Fatalf("view: %v %+v", err, res)
	}
	if n := r.halt(true); n == 0 {
		t.Error("stop with remove removed nothing")
	}
}

// An arrival turning around parks and becomes its departure: the departure
// adopts it instead of a fresh aircraft on another stand.
func TestScheduleTurnaround(t *testing.T) {
	fx, err := live.NewFixture("../../../live/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ft := &live.FixtureTraffic{ModelList: []string{"FSLTL A320 CSA Czech Airlines", "FSLTL B738 TVS Smartwings"}}
	r := newScheduleRunner(fx, ft)
	clock := &testClock{t: time.Date(2026, 9, 30, 6, 0, 0, 0, time.UTC)}
	r.clock = clock.now
	now := clock.now()
	r.start([]string{"LKPR"}, airport.LatLon{}, 2, 1, 24)
	defer r.halt(false)
	// Arrivals park at once, departures leave at once; four hours.
	for i := 0; i < 4*3600; i += 30 {
		now = clock.add(30 * time.Second)
		for _, v := range ft.Flights() {
			switch {
			case v.Kind == "arrival" && v.State == "spawning":
				ft.SetState(v.Callsign, "parked")
			case v.Kind == "departure" && v.State == "spawning":
				ft.SetState(v.Callsign, "complete")
			}
		}
		r.tick(now)
		time.Sleep(3 * time.Millisecond)
	}
	turned := 0
	for _, d := range ft.Departures {
		if d.Adopt == "" {
			continue
		}
		turned++
		for _, v := range ft.Flights() {
			if v.Callsign == d.Adopt {
				t.Errorf("%s turned into %s but is still one of ours", d.Adopt, d.Callsign)
			}
		}
	}
	if turned == 0 {
		t.Fatalf("no turnaround in four hours (%d departures, %d arrivals)", len(ft.Departures), len(ft.Arrivals))
	}
	t.Logf("%d of %d departures turned around", turned, len(ft.Departures))
}

// An arrival appears en route on its flight plan, flies to its STAR entry
// and is handed there to an arrival controller flying that STAR.
func TestScheduleEnrouteArrival(t *testing.T) {
	fx, err := live.NewFixture("../../../live/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ft := &live.FixtureTraffic{ModelList: []string{"FSLTL A320 DLH Lufthansa"}}
	r := newScheduleRunner(fx, ft)
	clock := &testClock{t: time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)}
	r.clock = clock.now
	now := clock.now()
	r.start([]string{"LKPR"}, airport.LatLon{}, 1, 1, 6)
	r.halt(false) // the manager only, no ticks
	lead := r.mgr.Options().ArrivalLead
	f := traffic.ManagedFlight{Flight: traffic.Flight{Callsign: "DLH1234", Airline: "DLH", Type: "A320", Origin: "EDDM", Destination: "LKPR",
		STA: now.Add(lead + 20*time.Minute)}, Kind: "arrival", Airport: "LKPR", Stage: "enroute"}
	v, err := r.spawnEnroute(t.Context(), f, lead)
	if err != nil {
		t.Fatal(err)
	}
	if v.Kind != "arrival" || len(ft.Enroute) != 1 {
		t.Fatalf("en route: %+v, %d spawned", v, len(ft.Enroute))
	}
	e := r.enroute["DLH1234"]
	s := ft.Enroute[0]
	last := s.Route[len(s.Route)-1].Position
	if e == nil || e.star == "" || e.entryFix == "" || last != e.entry {
		t.Fatalf("route ends at %v, want the STAR entry %+v", last, e)
	}
	// About 20 minutes out: over 100 NM along the plan.
	if d := calc.HaversineNM(s.Route[0].Position.Lat, s.Route[0].Position.Lon, e.entry.Lat, e.entry.Lon); d < 60 {
		t.Errorf("appears %.0f NM from the entry, want well out", d)
	}
	t.Logf("%s: %s, %d points to %s, then %s to %s", s.Callsign, s.Plan, len(s.Route), e.entryFix, e.star, e.runway)
	// At the entry: handed over.
	views := map[string]live.FlightView{"DLH1234": {Callsign: "DLH1234", Kind: "arrival", State: "enroute", Position: e.entry}}
	r.handovers(now, views)
	for i := 0; i < 100 && len(ft.Arrivals) == 0; i++ {
		time.Sleep(20 * time.Millisecond)
		_, _ = ft.Spawned()
	}
	_, arrs := ft.Spawned()
	if arrs != 1 {
		t.Fatalf("no arrival controller after the handover")
	}
	if a := ft.Arrivals[0]; !strings.HasPrefix(a.STAR, e.star) || a.Runway != e.runway || a.Callsign != "DLH1234" {
		t.Errorf("handed over to %s %s %s, want %s %s", a.Callsign, a.STAR, a.Runway, e.star, e.runway)
	}
}
