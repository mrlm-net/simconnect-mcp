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
	specs, _, _ := ft.Specs()
	if specs[0].HoldForClearances {
		t.Error("a scheduled departure holds for clearances: the tower would never clear it")
	}
	cs := specs[0].Callsign
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
	deps, arrs, _ := ft.Specs()
	for _, d := range deps {
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
		t.Fatalf("no turnaround in four hours (%d departures, %d arrivals)", len(deps), len(arrs))
	}
	t.Logf("%d of %d departures turned around", turned, len(deps))
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
	_, _, enroute := ft.Specs()
	if v.Kind != "arrival" || len(enroute) != 1 {
		t.Fatalf("en route: %+v, %d spawned", v, len(enroute))
	}
	e := r.enroute["DLH1234"]
	s := enroute[0]
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
	_, arrs := ft.Spawned()
	for i := 0; i < 100 && arrs == 0; i++ {
		time.Sleep(20 * time.Millisecond)
		_, arrs = ft.Spawned()
	}
	if arrs != 1 {
		t.Fatalf("no arrival controller after the handover")
	}
	_, handed, _ := ft.Specs()
	if a := handed[0]; !strings.HasPrefix(a.STAR, e.star) || a.Runway != e.runway || a.Callsign != "DLH1234" {
		t.Errorf("handed over to %s %s %s, want %s %s", a.Callsign, a.STAR, a.Runway, e.star, e.runway)
	}
}

// An overflight appears where its plan enters the area at its entry time,
// not where its STD would put it.
func TestScheduleOverflightEntersArea(t *testing.T) {
	fx, err := live.NewFixture("../../../live/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ft := &live.FixtureTraffic{ModelList: []string{"FSLTL A320 DLH Lufthansa"}}
	r := newScheduleRunner(fx, ft)
	clock := &testClock{t: time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)}
	r.clock = clock.now
	now := clock.now()
	// The area: 100 NM around a point west of Prague, which EDDM → LKPR crosses.
	centre := airport.LatLon{Lat: 49.6, Lon: 13.0}
	r.start([]string{"LKTB"}, centre, 1, 1, 6)
	r.halt(false)
	f := traffic.ManagedFlight{Flight: traffic.Flight{Callsign: "DLH9", Airline: "DLH", Type: "A320", Origin: "EDDM", Destination: "LKPR",
		STD: now.Add(-3 * time.Hour), Enter: now}, Kind: "overflight", Stage: "enroute"}
	if _, err := r.spawnEnroute(t.Context(), f, 0); err != nil {
		t.Fatal(err)
	}
	_, _, enroute := ft.Specs()
	p := enroute[0].Route[0].Position
	if d := calc.HaversineNM(p.Lat, p.Lon, centre.Lat, centre.Lon); d > overflightRadiusNM+5 || d < overflightRadiusNM-40 {
		t.Errorf("appears %.0f NM from the centre, want at the %d NM edge", d, overflightRadiusNM)
	}
}

// Dublin to Seoul crosses Prague on a straight lat/lon line, not on its
// great circle; Frankfurt to Kraków does both.
func TestGreatCircleCrosses(t *testing.T) {
	prague := airport.LatLon{Lat: 50.10, Lon: 14.26}
	dublin, seoul := airport.LatLon{Lat: 53.42, Lon: -6.27}, airport.LatLon{Lat: 37.46, Lon: 126.44}
	frankfurt, krakow := airport.LatLon{Lat: 50.03, Lon: 8.57}, airport.LatLon{Lat: 50.08, Lon: 19.78}
	if greatCircleCrosses(dublin, seoul, prague, overflightRadiusNM) {
		t.Error("Dublin–Seoul crosses Prague")
	}
	if !greatCircleCrosses(frankfurt, krakow, prague, overflightRadiusNM) {
		t.Error("Frankfurt–Kraków misses Prague")
	}
	if p := greatCirclePoint(dublin, seoul, 1); calc.HaversineNM(p.Lat, p.Lon, seoul.Lat, seoul.Lon) > 0.1 {
		t.Errorf("the great circle ends at %v, not Seoul", p)
	}
}
