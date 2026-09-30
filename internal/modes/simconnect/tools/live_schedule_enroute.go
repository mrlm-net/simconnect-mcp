//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
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
	e := &enrouteFlight{f: f}
	// An overflight is planned direct: its airways would be crawled up to
	// 400 NM around, and the area crossing is a great circle's.
	p := planArgs{Type: f.Type, Direct: !f.Arrival()}
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
	now := r.clock() // after planning, which takes a while
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
		// Where its plan enters the area, on from there since it entered
		// (the schedule's Enter): the plan's route and speeds are not the
		// generator's, so its STD would put it elsewhere.
		dist = now.Sub(f.STD.Add(10*time.Minute)).Hours() * kts
		if s := r.settings.Load(); s != nil && s.centre != (airport.LatLon{}) {
			in, ok := entersArea(fp, s.centre, overflightRadiusNM)
			if !ok {
				return live.FlightView{}, fmt.Errorf("its route (%s) does not cross the area", fp.Route)
			}
			dist = in + math.Max(0, now.Sub(f.Enter).Hours())*kts
		}
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

// entersArea is how far along fp it first comes within radiusNM of centre.
func entersArea(fp *nav.FlightPlan, centre airport.LatLon, radiusNM float64) (float64, bool) {
	const step = 2.0 // NM
	for d := 0.0; d <= fp.DistanceNM; d += step {
		p, _, _ := fp.PositionAt(d)
		if calc.HaversineNM(p.Lat, p.Lon, centre.Lat, centre.Lon) <= radiusNM {
			return d, true
		}
	}
	return 0, false
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
	if r.wasCleared() {
		return // stop_schedule removed the schedule's aircraft meanwhile
	}
	args := map[string]any{"icao": e.f.Airport, "callsign": cs, "runway": e.runway, "star": e.star, "model": e.model,
		"aircraft_type": e.f.Type, "hold_for_clearance": false}
	v, bad := spawnArrivalFrom(context.Background(), r.src, r.tr, args)
	if bad != nil {
		mgr.Failed(cs, fmt.Errorf("handover at %s: %s", e.entryFix, resultText(bad)), r.clock())
		return
	}
	if r.wasCleared() {
		_, _ = r.tr.Clear(cs, "remove")
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
	pos := map[string]airport.LatLon{}
	for _, a := range cfg.Airports {
		pos[a.ICAO] = a.Position
	}
	// The generator draws a straight line in latitude and longitude; a
	// long flight follows the great circle, far from it (Dublin to Seoul
	// "crosses" Prague on the line, Norway on the great circle).
	out := traffic.Overflights(cfg, traffic.OverflightOptions{Centre: s.centre, RadiusNM: overflightRadiusNM,
		Density: s.density, Seed: s.seed, Exclude: s.airports}, from, to)
	return slices.DeleteFunc(out, func(f traffic.Flight) bool {
		return !greatCircleCrosses(pos[f.Origin], pos[f.Destination], s.centre, overflightRadiusNM)
	})
}

// greatCircleCrosses reports whether the great circle from a to b passes
// through the area for at least half its radius.
func greatCircleCrosses(a, b, centre airport.LatLon, radiusNM float64) bool {
	if a == (airport.LatLon{}) || b == (airport.LatLon{}) {
		return false
	}
	dist := calc.HaversineNM(a.Lat, a.Lon, b.Lat, b.Lon)
	in := -1.0
	for d := 0.0; d <= dist; d += 5 {
		p := greatCirclePoint(a, b, d/dist)
		if calc.HaversineNM(p.Lat, p.Lon, centre.Lat, centre.Lon) <= radiusNM {
			if in < 0 {
				in = d
			}
			if d-in >= radiusNM/2 {
				return true
			}
		}
	}
	return false
}

// greatCirclePoint is the point a fraction f of the way from a to b on the
// great circle (spherical interpolation).
func greatCirclePoint(a, b airport.LatLon, f float64) airport.LatLon {
	vec := func(p airport.LatLon) [3]float64 {
		lat, lon := p.Lat*math.Pi/180, p.Lon*math.Pi/180
		return [3]float64{math.Cos(lat) * math.Cos(lon), math.Cos(lat) * math.Sin(lon), math.Sin(lat)}
	}
	u, v := vec(a), vec(b)
	delta := math.Acos(math.Max(-1, math.Min(1, u[0]*v[0]+u[1]*v[1]+u[2]*v[2])))
	if delta < 1e-9 {
		return a
	}
	ka, kb := math.Sin((1-f)*delta)/math.Sin(delta), math.Sin(f*delta)/math.Sin(delta)
	x, y, z := ka*u[0]+kb*v[0], ka*u[1]+kb*v[1], ka*u[2]+kb*v[2]
	return airport.LatLon{Lat: math.Atan2(z, math.Hypot(x, y)) * 180 / math.Pi, Lon: math.Atan2(y, x) * 180 / math.Pi}
}
