//go:build windows

// Package live gives MCP tools the mrlm-net/simconnect library's view of the
// simulator: airport layouts and taxi graphs (pkg/airport), procedures,
// fixes and airways, and the weather (pkg/nav).
//
// Source is what tools use. Runtime implements it on a live
// manager.Manager (Windows only); tests use a Source built from captured data.
package live

import (
	"context"
	"errors"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/nav"
)

// Source provides simulator data to tools. Every method blocks until the
// data has arrived, the simulator gives up, or ctx ends.
type Source interface {
	// Connected reports whether the simulator is connected.
	Connected() bool
	// Layout returns an airport's ground layout (cached after the first load).
	Layout(ctx context.Context, icao string) (*airport.Layout, error)
	// Graph returns an airport's taxi graph.
	Graph(ctx context.Context, icao string) (*airport.Graph, error)
	// Procedures returns an airport's SIDs, STARs and approaches.
	Procedures(ctx context.Context, icao string) (*airport.Procedures, error)
	// Weather returns the ambient weather at the user aircraft.
	Weather(ctx context.Context) (nav.Weather, error)
	// Fix loads a waypoint, VOR or NDB with the airways through it.
	Fix(ctx context.Context, key nav.FixKey) (nav.NavResult, error)
	// Airways crawls the airway network within radiusNM of center, starting
	// at the seeds, and returns it.
	Airways(ctx context.Context, center airport.LatLon, radiusNM float64, seeds []nav.FixKey) (*nav.AirwayGraph, error)
	// LoadFlightPlan loads an MSFS .pln into the simulator for the user aircraft.
	LoadFlightPlan(ctx context.Context, pln []byte) error
}

// Errors of a Source.
var (
	ErrNotConnected = errors.New("not connected to the simulator")
	ErrNotFound     = errors.New("not found in the simulator's data")
	ErrTimeout      = errors.New("the simulator did not answer in time")
	ErrReconnected  = errors.New("the simulator connection was reset")
)
