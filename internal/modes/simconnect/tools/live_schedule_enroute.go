//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/nav"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// En route scheduled traffic: an arrival appears 20 minutes before it is
// due at its STAR entry, airborne on its flight plan (planned as
// plan_flight would, to the runway in use) and flown there by MSFS AI; at
// the entry it is handed to an arrival controller — the MSFS AI aircraft
// makes way for a controlled one flying the same STAR. Overflights cross
// the area around the schedule's first airport between airports outside
// it and leave with it.

// Handover: this close to the STAR entry, or this long after the arrival
// was due there.
const (
	handoverNM         = 8.0
	handoverAfter      = 2 * time.Minute
	overflightRadiusNM = 100
	enrouteMinNM       = 15 // an arrival closer than this to its entry appears there
)

// enrouteFlight is a scheduled flight flying en route.
type enrouteFlight struct {
	f traffic.ManagedFlight
	// Arrival: its STAR entry, when it is due there, and what the arrival
	// controller flies from it.
	entry               airport.LatLon
	entryFix            string
	due                 time.Time
	star, runway, model string
	handing             bool
}

// spawnEnroute puts a scheduled arrival or overflight in the air where its
// flight is now.
func (r *scheduleRunner) spawnEnroute(ctx context.Context, f traffic.ManagedFlight, arrivalLead time.Duration) (live.FlightView, error) {
	now := r.clock()
	e := &enrouteFlight{f: f}
	p := planArgs{Type: f.Type}
	var procs *airport.Procedures
	if f.Arrival() {
		g, err := r.src.Graph(ctx, f.Airport)
		if err != nil {
			return live.FlightView{}, err
		}
		w, err := r.src.Weather(ctx)
		if err != nil {
			return live.FlightView{}, err
		}
		procs, _ = r.src.Procedures(ctx, f.Airport)
		if procs == nil {
			return live.FlightView{}, errors.New("no procedures: it appears on final")
		}
		e.runway = r.src.RunwaysInUse(g.Layout, w, nav.RunwayLimitsFrom(airport.LimitsFor(g.Layout, procs))).Arrival.Name
		p.ArrivalRunway = e.runway
	}
	fp, _, bad := planBetween(ctx, r.src, f.Origin, f.Destination, p)
	if bad != nil {
		return live.FlightView{}, fmt.Errorf("flight plan %s → %s: %s", f.Origin, f.Destination, resultText(bad))
	}
	kts := fp.Performance.CruiseTASKts
	if kts < 200 {
		kts = 420
	}
	var dist float64
	entryDist := fp.DistanceNM
	if f.Arrival() {
		// The STAR's first fix, where spawn_arrival starts it.
		for _, s := range procs.STARsFor(e.runway) {
			if strings.EqualFold(s.Name, fp.STAR) {
				e.entryFix = starEntry(s, e.runway)
			}
		}
		found := false
		for _, w := range fp.Waypoints {
			if e.entryFix != "" && w.Ident == e.entryFix && (w.Phase == nav.PhaseSTAR || w.Phase == nav.PhaseApproach) {
				e.entry, entryDist, found = w.Position, w.DistanceNM, true
				break
			}
		}
		if !found {
			return live.FlightView{}, fmt.Errorf("no STAR entry on the plan (%q): it appears there", fp.STAR)
		}
		// Where it must be now to reach the entry when it is due there.
		e.due, e.star = f.STA.Add(-arrivalLead), fp.STAR
		dist = entryDist - e.due.Sub(now).Hours()*kts
		if dist > entryDist-enrouteMinNM {
			return live.FlightView{}, fmt.Errorf("only %.0f NM before its STAR entry: it appears there", entryDist-dist)
		}
	} else {
		dist = now.Sub(f.STD.Add(10*time.Minute)).Hours() * kts
	}
	dist = math.Max(10, math.Min(dist, fp.DistanceNM-20))
	pos, altFt, _ := fp.PositionAt(dist)
	// The rest of the flight: an arrival to its STAR entry, an overflight
	// up to its destination's STAR.
	route := []traffic.RoutePoint{{Position: pos, AltFt: altFt, Kts: traffic.EnrouteSpeedKts(altFt, kts)}}
	for _, w := range fp.Waypoints {
		if w.DistanceNM <= dist+2 || w.Kind == nav.PointRunway || w.Kind == nav.PointAirport {
			continue
		}
		if f.Arrival() && w.DistanceNM > entryDist || !f.Arrival() && (w.Phase == nav.PhaseSTAR || w.Phase == nav.PhaseApproach) {
			break
		}
		route = append(route, traffic.RoutePoint{Position: w.Position, AltFt: w.AltFt, Kts: traffic.EnrouteSpeedKts(w.AltFt, kts)})
	}
	if len(route) < 2 {
		return live.FlightView{}, errors.New("no route left to fly en route")
	}
	s := live.EnrouteSpec{Callsign: f.Callsign, Kind: f.Kind, Type: f.Type, Route: route, Plan: fp.Route}
	if f.Arrival() {
		s.ICAO = f.Airport
	}
	v, err := r.tr.SpawnEnroute(ctx, s)
	if err != nil {
		return live.FlightView{}, err
	}
	e.model = v.Model
	r.mu.Lock()
	r.enroute[f.Callsign] = e
	r.mu.Unlock()
	return v, nil
}

// handovers passes en route arrivals to an arrival controller at their
// STAR entry, and forgets the en route flights no longer ours.
func (r *scheduleRunner) handovers(now time.Time, views map[string]live.FlightView) {
	r.mu.Lock()
	var due []*enrouteFlight
	for cs, e := range r.enroute {
		v, ok := views[cs]
		switch {
		case e.handing:
		case !ok || v.Kind != "arrival" && v.Kind != "overflight":
			delete(r.enroute, cs) // removed, or already handed over
		case e.f.Arrival() && v.State == "enroute" &&
			(calc.HaversineNM(v.Position.Lat, v.Position.Lon, e.entry.Lat, e.entry.Lon) < handoverNM || now.After(e.due.Add(handoverAfter))):
			e.handing = true
			due = append(due, e)
		}
	}
	mgr := r.mgr
	r.mu.Unlock()
	for _, e := range due {
		go r.handover(mgr, e)
	}
}

// handover replaces an en route arrival at its STAR entry with an arrival
// controller flying the STAR, the same model and call sign.
func (r *scheduleRunner) handover(mgr *traffic.TrafficManager, e *enrouteFlight) {
	cs := e.f.Callsign
	defer func() {
		r.mu.Lock()
		delete(r.enroute, cs)
		r.mu.Unlock()
	}()
	_, _ = r.tr.Clear(cs, "remove")
	args := map[string]any{"icao": e.f.Airport, "callsign": cs, "runway": e.runway, "star": e.star, "model": e.model,
		"aircraft_type": e.f.Type, "hold_for_clearance": false}
	v, bad := spawnArrivalFrom(context.Background(), r.src, r.tr, args)
	if bad != nil {
		mgr.Failed(cs, fmt.Errorf("handover at %s: %s", e.entryFix, resultText(bad)), r.clock())
		return
	}
	mgr.Describe(cs, v.Model, v.Stand, v.Runway)
}

// overflights are the flights crossing the area around the schedule's
// first airport in an hour; none before its position is known.
func (r *scheduleRunner) overflights(cfg traffic.ScheduleConfig, from, to time.Time) []traffic.Flight {
	s := r.settings.Load()
	if s == nil || s.centre == (airport.LatLon{}) {
		return nil
	}
	return traffic.Overflights(cfg, traffic.OverflightOptions{Centre: s.centre, RadiusNM: overflightRadiusNM,
		Density: s.density, Seed: s.seed, Exclude: s.airports}, from, to)
}
