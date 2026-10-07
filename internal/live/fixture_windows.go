//go:build windows

package live

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/nav"
)

// Fixture is a Source serving captured data, for tests: raw airport records
// (<ICAO>.json) or built layouts (<ICAO>-layout.json), procedures (<ICAO>-procedures.json) and an airway graph
// (airways.json or any *-airways.json) from a directory, with fixed weather.
type Fixture struct {
	WeatherValue nav.Weather
	// Loaded collects the .pln files LoadFlightPlan received.
	Loaded [][]byte

	cache   *airport.Cache
	procs   map[string]*airport.Procedures
	graph   *nav.AirwayGraph
	runways RunwayMemory
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

func (f *Fixture) RunwaysInUse(l *airport.Layout, w nav.Weather, lim nav.RunwayLimits) nav.RunwayUse {
	return f.runways.Use(l, w, lim)
}

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
