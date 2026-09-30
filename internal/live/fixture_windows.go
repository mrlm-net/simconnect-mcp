//go:build windows

package live

import (
	"sync"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/nav"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// Fixture is a Source serving captured data, for tests: raw airport records
// (<ICAO>.json) or built layouts (<ICAO>-layout.json), procedures (<ICAO>-procedures.json) and an airway graph
// (airways.json or any *-airways.json) from a directory, with fixed weather.
type Fixture struct {
	WeatherValue nav.Weather
	// Loaded collects the .pln files LoadFlightPlan received.
	Loaded [][]byte

	cache *airport.Cache
	procs map[string]*airport.Procedures
	graph *nav.AirwayGraph
}

// NewFixture reads the fixtures in dir.
func NewFixture(dir string) (*Fixture, error) {
	f := &Fixture{
		cache:        airport.NewCache(),
		procs:        map[string]*airport.Procedures{},
		WeatherValue: nav.StaticWeather(240, 10, 9999, 15, 5, 1013),
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		switch {
		case strings.HasSuffix(name, "airways"):
			if f.graph, err = nav.ReadAirwayGraph(strings.NewReader(string(b))); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
		case strings.HasSuffix(name, "-layout"):
			var l airport.Layout
			if err := json.Unmarshal(b, &l); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			f.cache.Put(&l)
		case strings.HasSuffix(name, "-procedures"):
			var p airport.Procedures
			if err := json.Unmarshal(b, &p); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			f.procs[strings.ToUpper(strings.TrimSuffix(name, "-procedures"))] = &p
		default:
			var raw airport.RawAirport
			if err := json.Unmarshal(b, &raw); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			l, err := airport.BuildLayout(raw)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			f.cache.Put(l)
		}
	}
	return f, nil
}

func (f *Fixture) Connected() bool { return true }

func (f *Fixture) Layout(_ context.Context, icao string) (*airport.Layout, error) {
	if l, ok := f.cache.Layout(icao); ok {
		return l, nil
	}
	return nil, fmt.Errorf("airport %s: %w", icao, ErrNotFound)
}

func (f *Fixture) Graph(ctx context.Context, icao string) (*airport.Graph, error) {
	if _, err := f.Layout(ctx, icao); err != nil {
		return nil, err
	}
	return f.cache.Graph(icao)
}

func (f *Fixture) Procedures(_ context.Context, icao string) (*airport.Procedures, error) {
	if p, ok := f.procs[strings.ToUpper(icao)]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("procedures of %s: %w", icao, ErrNotFound)
}

func (f *Fixture) Weather(context.Context) (nav.Weather, error) { return f.WeatherValue, nil }

func (f *Fixture) Fix(_ context.Context, key nav.FixKey) (nav.NavResult, error) {
	if f.graph != nil {
		for _, fx := range f.graph.Find(key.Ident) {
			if fx.Kind == key.Kind && (key.Region == "" || fx.Region == key.Region) {
				k := nav.Key(fx.Ident, fx.Region, fx.Kind)
				return nav.NavResult{Key: k, Fix: fx, Found: true}, nil
			}
		}
	}
	return nav.NavResult{Key: key}, fmt.Errorf("fix %s: %w", key, ErrNotFound)
}

func (f *Fixture) Airways(context.Context, airport.LatLon, float64, []nav.FixKey) (*nav.AirwayGraph, error) {
	if f.graph == nil {
		return nil, fmt.Errorf("airways: %w", ErrNotFound)
	}
	return f.graph, nil
}

func (f *Fixture) LoadFlightPlan(_ context.Context, pln []byte) error {
	f.Loaded = append(f.Loaded, pln)
	return nil
}

// FixtureTraffic is a Traffic for tests: it records spawns and clearances.
type FixtureTraffic struct {
	mu sync.Mutex // spawns come from goroutines (scheduled traffic)

	ModelList    []string
	PictureList  []traffic.TrackedAircraft
	SequenceList []RunwaySequence
	TowerList    []RunwayUser
	Log          []ATCMessage
	Instructions []string // "CS action", as given to Approach
	ApproachSaid string
	ApproachErr  error
	Departures   []DepartureSpec
	Arrivals     []ArrivalSpec
	flights      []FlightView
}

func (f *FixtureTraffic) Models(context.Context) ([]string, error) { return f.ModelList, nil }

func (f *FixtureTraffic) SpawnDeparture(_ context.Context, s DepartureSpec) (FlightView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.Adopt != "" { // a turnaround: the parked arrival becomes the departure
		i := slices.IndexFunc(f.flights, func(v FlightView) bool { return v.Callsign == s.Adopt })
		if i < 0 || f.flights[i].Kind != "arrival" || f.flights[i].State != "parked" {
			return FlightView{}, fmt.Errorf("turnaround of %s: not an arrival parked on its stand", s.Adopt)
		}
		s.Stand = f.flights[i].Stand
		f.flights = slices.Delete(f.flights, i, i+1)
	}
	f.Departures = append(f.Departures, s)
	v := FlightView{Callsign: s.Callsign, Kind: "departure", ICAO: s.Graph.Layout.ICAO, Stand: s.Stand, Runway: s.Runway,
		Entry: s.Entry, Procedure: s.SID, State: "spawning", Actions: []string{"remove"}}
	f.flights = append(f.flights, v)
	return v, nil
}

func (f *FixtureTraffic) SpawnArrival(_ context.Context, s ArrivalSpec) (FlightView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Arrivals = append(f.Arrivals, s)
	v := FlightView{Callsign: s.Callsign, Kind: "arrival", ICAO: s.Graph.Layout.ICAO, Stand: s.Stand, Runway: s.Runway,
		Procedure: s.STAR, State: "spawning", Actions: []string{"remove"}}
	f.flights = append(f.flights, v)
	return v, nil
}

func (f *FixtureTraffic) Flights() []FlightView {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]FlightView(nil), f.flights...)
}

// SetState sets a flight's state, as its controller would.
func (f *FixtureTraffic) SetState(callsign, state string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.flights {
		if f.flights[i].Callsign == callsign {
			f.flights[i].State = state
		}
	}
}

// Spawned counts the departures and arrivals spawned.
func (f *FixtureTraffic) Spawned() (departures, arrivals int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Departures), len(f.Arrivals)
}

func (f *FixtureTraffic) Clear(callsign, action string) (FlightView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range f.flights {
		if v.Callsign == callsign {
			return v, nil
		}
	}
	return FlightView{}, fmt.Errorf("%s: %w", callsign, ErrUnknownFlight)
}

func (f *FixtureTraffic) Picture(context.Context, string, float64) ([]traffic.TrackedAircraft, error) {
	return f.PictureList, nil
}

// Sequences, Tower and ATCLog return the fixture's; Approach records the
// instruction and answers from ApproachSaid.
func (f *FixtureTraffic) Sequences(string) []RunwaySequence { return f.SequenceList }
func (f *FixtureTraffic) Tower(string) []RunwayUser         { return f.TowerList }
func (f *FixtureTraffic) ATCLog(limit int) []ATCMessage {
	if limit > 0 && len(f.Log) > limit {
		return f.Log[len(f.Log)-limit:]
	}
	return f.Log
}
func (f *FixtureTraffic) Approach(callsign, action string) (string, error) {
	f.Instructions = append(f.Instructions, callsign+" "+action)
	return f.ApproachSaid, f.ApproachErr
}
