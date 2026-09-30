//go:build windows

package live

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mrlm-net/simconnect/pkg/engine"
	"github.com/mrlm-net/simconnect/pkg/traffic"
	"github.com/mrlm-net/simconnect/pkg/types"
)

// En route aircraft — scheduled arrivals before their STAR entry, and
// overflights — appear airborne where their flight is now and fly the rest
// of it as MSFS AI on a waypoint chain: Fleet.RequestNonATC at
// traffic.EnrouteStart's position, then ReleaseControl and SetWaypoints.
// No controller flies them: they are ours to follow and remove, and an
// arrival is handed to an arrival controller at its STAR entry (by the
// caller: remove it, spawn the arrival there). (MSFS 2024 puts an enroute
// ATC aircraft on the ground at its plan's departure airport whatever the
// phase, so a flight plan cannot start one mid-route.)

// En route IDs, above the traffic IDs (traffic_windows.go).
const (
	enrouteReqBase      = idBase + 42_000
	enrouteReqCount     = 1000
	enrouteDefWaypoints = idBase + 43_000
	enrouteReleaseReq   = idBase + 43_001
	enrouteRemoveReq    = idBase + 43_002
	// The simulator answers a creation within a second or two; after this
	// the aircraft is taken as not created.
	enrouteCreateTimeout = 15 * time.Second
)

// EnrouteSpec asks for an en route aircraft.
type EnrouteSpec struct {
	Callsign string
	// Kind: "arrival" (handed to an arrival controller later) or
	// "overflight".
	Kind string
	ICAO string // the arrival airport; "" for an overflight
	// Model, Livery and Type as for a departure.
	Model, Livery, Type string
	// Route: from where it appears (the first point, heading for the
	// second) along its flight plan, at its levels and speeds.
	Route []traffic.RoutePoint
	Plan  string // its route, for the view
}

// SpawnEnroute implements Traffic.
func (r *Runtime) SpawnEnroute(ctx context.Context, s EnrouteSpec) (FlightView, error) {
	if s.Kind != "arrival" && s.Kind != "overflight" {
		return FlightView{}, fmt.Errorf("en route kind %q: arrival or overflight", s.Kind)
	}
	model, livery, err := r.pickModel(ctx, s.Model, s.Livery, s.Type, s.Callsign)
	if err != nil {
		return FlightView{}, err
	}
	spawn, wps, err := traffic.EnrouteStart(s.Route)
	if err != nil {
		return FlightView{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.connected {
		return FlightView{}, ErrNotConnected
	}
	t, err := r.trafficLocked()
	if err != nil {
		return FlightView{}, err
	}
	t.tmu.Lock()
	_, dup := t.flights[s.Callsign]
	n := len(t.flights)
	t.tmu.Unlock()
	if dup {
		return FlightView{}, fmt.Errorf("%s is already one of ours", s.Callsign)
	}
	if n >= maxFlights {
		return FlightView{}, fmt.Errorf("at most %d aircraft of ours at once; remove some", maxFlights)
	}
	if err := r.startScanLocked(); err != nil {
		return FlightView{}, err
	}
	if !t.waypointDef {
		if err := r.mgr.AddToDataDefinition(enrouteDefWaypoints, "AI Waypoint List", "number", types.SIMCONNECT_DATATYPE_WAYPOINT, 0, 0); err != nil {
			return FlightView{}, err
		}
		t.waypointDef = true
	}
	t.nextReq = (t.nextReq + 1) % enrouteReqCount
	req := enrouteReqBase + t.nextReq
	if err := t.fleet.RequestNonATC(traffic.NonATCOpts{Model: model, Livery: livery, Tail: s.Callsign, Position: spawn}, req); err != nil {
		return FlightView{}, err
	}
	first := s.Route[0]
	f := &flight{ts: t, enroute: true, reqID: req, asked: time.Now(), waypoints: wps, view: FlightView{Callsign: s.Callsign, Kind: s.Kind,
		ICAO: s.ICAO, Model: joinModel(model, livery), Procedure: s.Plan, State: "spawning", Position: first.Position,
		Heading: spawn.Heading, GroundSpeed: first.Kts, AltFt: first.AltFt, Actions: []string{"remove"}}}
	t.pending[req] = f
	t.tmu.Lock()
	t.flights[s.Callsign] = f
	t.tmu.Unlock()
	return f.view, nil
}

// enrouteCreatedLocked takes the simulator's answer to an en route
// creation: released to MSFS AI, the aircraft flies its waypoints.
func (r *Runtime) enrouteCreatedLocked(msg engine.Message) bool {
	t := r.traffic
	m := msg.AsAssignedObjectID()
	f := t.pending[uint32(m.DwRequestID)]
	if f == nil {
		return false
	}
	delete(t.pending, f.reqID)
	id := uint32(m.DwObjectID)
	t.fleet.Acknowledge(f.reqID, id)
	t.tmu.Lock()
	gone := t.flights[f.view.Callsign] != f // removed while it was being created
	f.id = id
	if !gone {
		f.view.State = "enroute"
	}
	t.tmu.Unlock()
	if gone {
		_ = t.fleet.Remove(id, enrouteRemoveReq)
		return true
	}
	_ = t.fleet.ReleaseControl(id, enrouteReleaseReq)
	if err := t.fleet.SetWaypoints(id, enrouteDefWaypoints, f.waypoints); err != nil {
		t.tmu.Lock()
		f.view.Error = "waypoints: " + err.Error()
		t.tmu.Unlock()
	}
	t.picture.SetOwn(id, traffic.PhaseEnroute, f.view.ICAO)
	return true
}

// tickEnrouteLocked fails creations the simulator never answered and
// follows the en route aircraft in the picture.
func (r *Runtime) tickEnrouteLocked(now time.Time) {
	t := r.traffic
	for req, f := range t.pending {
		if now.Sub(f.asked) > enrouteCreateTimeout {
			delete(t.pending, req)
			t.tmu.Lock()
			f.view.State, f.view.Error, f.view.Done = "failed", "the simulator did not create the aircraft (model, or position out of range)", true
			t.tmu.Unlock()
		}
	}
	t.tmu.Lock()
	any := false
	for _, f := range t.flights {
		any = any || f.enroute
	}
	t.tmu.Unlock()
	if !any {
		return
	}
	// Seen only within the picture's radius: farther out a view keeps its
	// last position.
	seen := map[uint32]traffic.TrackedAircraft{}
	for _, a := range t.picture.Aircraft() {
		if a.Ours {
			seen[a.ObjectID] = a
		}
	}
	t.tmu.Lock()
	defer t.tmu.Unlock()
	for _, f := range t.flights {
		if a, ok := seen[f.id]; ok && f.enroute {
			f.view.Position, f.view.Heading, f.view.GroundSpeed = a.Position, a.Heading, a.GroundKts
			f.view.AltFt, f.view.AGLFt, f.view.OnGround = a.AltFt, a.AGLFt, a.OnGround
		}
	}
}

// removeEnroute takes an en route aircraft out of the simulator; one still
// being created goes when the simulator answers.
func (r *Runtime) removeEnroute(f *flight) error {
	t := f.ts
	// Gone first: a creation answered from now on removes its aircraft
	// (enrouteCreatedLocked); one answered before has set the ID.
	t.tmu.Lock()
	if t.flights[f.view.Callsign] == f {
		delete(t.flights, f.view.Callsign)
	}
	id := f.id
	t.tmu.Unlock()
	if id == 0 {
		return nil
	}
	t.picture.ForgetOwn(id)
	if err := t.fleet.Remove(id, enrouteRemoveReq); err != nil {
		return errors.Join(fmt.Errorf("removing %s", f.view.Callsign), err)
	}
	return nil
}
