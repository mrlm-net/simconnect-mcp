//go:build windows

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/traffic"
	"github.com/mrlm-net/simconnect/pkg/traffic/world"
)

// The AI traffic and ATC tools run on the library's traffic engine,
// pkg/traffic/world (the airport map's): spawning, pushback with tugs and
// stand services, taxi, the tower, landing sequences, separation,
// schedules, real-world traffic, traffic along the user's route, and the
// radio. The engine starts on the first tool that needs it.

var worldCallsignRe = regexp.MustCompile(`^[A-Z0-9]{2,8}$`)

// worldTimeout bounds a tool's wait for the engine to come up.
const worldTimeout = 45 * time.Second

// RegisterWorldTrafficTools registers the traffic tools on w.
func RegisterWorldTrafficTools(mcp *mcpadapter.Server, w live.World) {
	registerWorldModels(mcp, w)
	registerWorldSpawnDeparture(mcp, w)
	registerWorldSpawnArrival(mcp, w)
	registerWorldListOurTraffic(mcp, w)
	registerWorldClearance(mcp, w)
	registerWorldPicture(mcp, w)
	registerWorldLandingSequence(mcp, w)
	registerWorldApproach(mcp, w)
	registerWorldATCLog(mcp, w)
	registerWorldConflicts(mcp, w)
	registerWorldStartSchedule(mcp, w)
	registerWorldStopSchedule(mcp, w)
	registerWorldGetSchedule(mcp, w)
	registerWorldAddFlights(mcp, w)
	registerWorldRealTraffic(mcp, w)
	registerWorldObserve(mcp, w)
	registerWorldCorridor(mcp, w)
	registerWorldPlayer(mcp, w)
	registerWorldStatus(mcp, w)
	registerWorldAirportInfo(mcp, w)
	registerWorldTCAS(mcp, w)
	registerGenerateSchedule(mcp)
	registerSeparationMinima(mcp)
}

// worldError turns an engine error into a tool error result. The API's
// errors read "METHOD path: CODE text".
func worldError(what string, err error) *mcpadapter.CallToolResult {
	msg := err.Error()
	code := "SIM_ERROR"
	switch {
	case errors.Is(err, live.ErrNotConnected) || strings.Contains(msg, ": 503"):
		code = "BRIDGE_DISCONNECTED"
	case errors.Is(err, live.ErrTimeout) || errors.Is(err, context.DeadlineExceeded):
		code = "TIMEOUT"
	case strings.Contains(msg, ": 404"):
		code = "NOT_FOUND"
	case strings.Contains(msg, ": 400"):
		code = "INVALID_ARGUMENT"
	case strings.Contains(msg, ": 409"), strings.Contains(msg, ": 422"), strings.Contains(msg, ": 403"):
		code = "NOT_APPLICABLE"
	}
	return mcpadapter.ErrorResult(fmt.Sprintf("%s: %s: %s", code, what, msg))
}

// ensure starts the engine (or waits for it); nil when it runs.
func ensure(ctx context.Context, w live.World) *mcpadapter.CallToolResult {
	ctx, cancel := context.WithTimeout(ctx, worldTimeout)
	defer cancel()
	if err := w.Ensure(ctx); err != nil {
		return worldError("traffic engine", err)
	}
	return nil
}

// loadAirport has the engine load icao's layout (spawning needs it).
func loadAirport(w live.World, icao string) error {
	_, err := w.Do("GET", "/api/airport?icao="+url.QueryEscape(icao), nil)
	return err
}

// ourAircraft lists the engine's aircraft.
func ourAircraft(w live.World) ([]world.ControlView, error) {
	var list []world.ControlView
	err := w.Get("/api/control", &list)
	return list, err
}

// findOurs finds one of ours by call sign (its tail) or engine id.
func findOurs(w live.World, callsign string) (world.ControlView, error) {
	list, err := ourAircraft(w)
	if err != nil {
		return world.ControlView{}, err
	}
	cs := strings.ToUpper(strings.TrimSpace(callsign))
	for _, a := range list {
		if strings.EqualFold(a.Tail, cs) || strconv.Itoa(a.ID) == cs {
			return a, nil
		}
	}
	return world.ControlView{}, fmt.Errorf("%s is not one of ours: %w", cs, live.ErrNotFound)
}

// aircraftView is one of ours as the tools show it: the engine's view
// without its routes unless detail is asked.
func aircraftView(a world.ControlView, detail bool) map[string]any {
	b, _ := json.Marshal(a)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if !detail {
		for _, k := range []string{"route", "nodes", "airRoute", "airFixes", "tug", "fuel"} {
			delete(m, k)
		}
		vs, _ := m["vehicles"].([]any)
		for _, v := range vs {
			if vm, ok := v.(map[string]any); ok {
				delete(vm, "route")
			}
		}
	}
	m["callsign"] = a.Tail
	return m
}

// pickModel is a model title for an ICAO type, from the engine's list:
// the first whose title names the type ("A320", "B738"); "" none.
func pickModel(w live.World, typ string) string {
	typ = strings.ToUpper(strings.TrimSpace(typ))
	if typ == "" {
		return ""
	}
	var models []string
	if w.Get("/api/models", &models) != nil {
		return ""
	}
	for _, m := range models {
		if strings.Contains(strings.ToUpper(m), typ) {
			return m
		}
	}
	return ""
}

// standIndex is the engine's index of the stand labelled label at icao.
func standIndex(w live.World, icao, label string) (int, error) {
	var stands []struct {
		Index int    `json:"index"`
		Label string `json:"label"`
	}
	if err := w.Get("/api/stands?icao="+url.QueryEscape(icao), &stands); err != nil {
		return 0, err
	}
	label = strings.ToUpper(strings.ReplaceAll(label, " ", ""))
	for _, s := range stands {
		if strings.ToUpper(strings.ReplaceAll(s.Label, " ", "")) == label {
			return s.Index, nil
		}
	}
	return 0, fmt.Errorf("stand %s at %s: %w", label, icao, live.ErrNotFound)
}

func registerWorldModels(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("list_aircraft_models").
		Description("List the installed aircraft models (titles, with their livery after \"|\") the traffic engine can spawn, "+
			"for spawn_departure and spawn_arrival's model.").
		StringParam("search", "Only titles containing this text, e.g. \"A320\" or \"Lufthansa\".").
		NumberParam("limit", "At most this many (default 100).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var models []string
		if err := w.Get("/api/models", &models); err != nil {
			return worldError("models", err), nil
		}
		search := strings.ToLower(strArg(args, "search"))
		limit := int(numArg(args, "limit", 100))
		out := []string{}
		matched := 0
		for _, m := range models {
			if search != "" && !strings.Contains(strings.ToLower(m), search) {
				continue
			}
			matched++
			if len(out) < limit {
				out = append(out, m)
			}
		}
		return mcpadapter.JSONResult(map[string]any{"count": matched, "models": out})
	})
}

// spawnRequest builds the engine's spawn request from the spawn tools'
// common arguments; bad is the tool's error result.
func spawnRequest(ctx context.Context, w live.World, args map[string]any, kind string) (world.SpawnRequest, string, *mcpadapter.CallToolResult) {
	if bad := ensure(ctx, w); bad != nil {
		return world.SpawnRequest{}, "", bad
	}
	icao, bad := icaoArg(args, "icao")
	if bad != nil {
		return world.SpawnRequest{}, "", bad
	}
	cs := strings.ToUpper(strArg(args, "callsign"))
	if !worldCallsignRe.MatchString(cs) {
		return world.SpawnRequest{}, "", mcpadapter.ErrorResult("INVALID_ARGUMENT: callsign must be 2–8 letters and digits")
	}
	if _, err := findOurs(w, cs); err == nil {
		return world.SpawnRequest{}, "", mcpadapter.ErrorResult("INVALID_ARGUMENT: " + cs + " is already one of ours")
	}
	if err := loadAirport(w, icao); err != nil {
		return world.SpawnRequest{}, "", worldError("loading "+icao, err)
	}
	req := world.SpawnRequest{Kind: kind, ICAO: icao, Tail: cs, Stand: -1, Runway: strings.ToUpper(strArg(args, "runway")),
		Model: strArg(args, "model"), Taxiways: listArg(args, "via"), Squawk: strArg(args, "squawk")}
	if req.Model == "" {
		req.Model = pickModel(w, strArg(args, "aircraft_type"))
	}
	if s := strArg(args, "stand"); s != "" {
		i, err := standIndex(w, icao, s)
		if err != nil {
			return world.SpawnRequest{}, "", worldError("stand", err)
		}
		req.Stand = i
	}
	return req, cs, nil
}

// spawn sends req and, with hold, gives the aircraft's clearances to the
// caller (manual: the engine's automation leaves it).
func spawn(w live.World, req world.SpawnRequest, hold bool) (*mcpadapter.CallToolResult, error) {
	b, err := w.Do("POST", "/api/control", req)
	if err != nil {
		return worldError(req.Kind+" "+req.Tail+" at "+req.ICAO, err), nil
	}
	var v world.ControlView
	if err := live.DecodeJSON(b, &v); err != nil {
		return worldError(req.Kind, err), nil
	}
	if hold {
		if _, err := w.Do("POST", fmt.Sprintf("/api/control/%d/manual?on=1", v.ID), nil); err != nil {
			return worldError("hold for clearances", err), nil
		}
		v.Manual = true
	}
	return mcpadapter.JSONResult(aircraftView(v, false))
}

// procedureArg sets a spawn's procedure from "auto", "none" or a name.
func procedureArg(req *world.SpawnRequest, v string) {
	switch strings.ToLower(v) {
	case "", "auto":
		req.Procedure = true
	case "none":
		req.Procedure = false
	default:
		req.Procedure, req.ProcName = true, strings.ToUpper(v)
	}
}

func boolArg(args map[string]any, key string, def bool) bool {
	if b, ok := args[key].(bool); ok {
		return b
	}
	return def
}

func registerWorldSpawnDeparture(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("spawn_departure").
		Description("Put an AI departure of ours on a stand (this adds an aircraft to the sim). The traffic engine gives it "+
			"its stand services (fuel truck; stairs and GPU at a remote stand), a pushback with a tug, the taxi route to the "+
			"runway, line-up, take-off and its SID, and its crew talks to delivery, ground and tower. With "+
			"hold_for_clearances (default true) it waits for atc_clearance at each step; false lets the engine's tower clear "+
			"it. Stand, runway in use, SID and model are chosen when not given. Returns the aircraft; follow it with "+
			"list_our_traffic.").
		StringParam("icao", "Airport ICAO code (required); the simulator must have it loaded around the user aircraft.").
		StringParam("callsign", "Call sign, e.g. \"CSA123\" (required).").
		StringParam("stand", "Stand label, e.g. \"C22\" (default: a free stand that fits).").
		StringParam("runway", "Departure runway (default: in use).").
		StringParam("entry", "Runway entry taxiway for an intersection departure, e.g. \"B\" (default: full length).").
		StringParam("sid", "SID name, \"auto\" (default: one for the runway) or \"none\".").
		StringParam("model", "Model title from list_aircraft_models.").
		StringParam("aircraft_type", "ICAO type to pick a model by, e.g. \"B738\" (when no model is given).").
		StringParam("via", "Taxiways to follow in order, e.g. \"F, L\".").
		StringParam("squawk", "SSR code (default: one is assigned).").
		NumberParam("push_in_min", "Push this many minutes from now, giving the stand services their time (default: when ready).").
		StringParam("stand_use", "Stand kind when none is given: \"ga\" or \"cargo\" (default: an airliner stand).").
		BoolParam("hold_for_clearances", "Wait for atc_clearance at every step (default true).").
		BoolParam("tug", "A pushback tug pushes it (default true).").
		BoolParam("fuel", "A fuel truck comes before the push (default true).").
		Required("icao", "callsign").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		req, _, bad := spawnRequest(ctx, w, args, "departure")
		if bad != nil {
			return bad, nil
		}
		req.Entry = strings.ToUpper(strArg(args, "entry"))
		req.StandUse = strings.ToLower(strArg(args, "stand_use"))
		req.PushInMin = numArg(args, "push_in_min", 0)
		req.Tug, req.Fuel = boolArg(args, "tug", true), boolArg(args, "fuel", true)
		procedureArg(&req, strArg(args, "sid"))
		return spawn(w, req, boolArg(args, "hold_for_clearances", true))
	})
}

func registerWorldSpawnArrival(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("spawn_arrival").
		Description("Put an AI arrival of ours into the simulator (this adds an aircraft to the sim): on its STAR, sequenced "+
			"by the engine's approach controller, flying the approach, landing, vacating and taxiing to a stand. With "+
			"hold_for_clearance (default true) it waits for atc_clearance taxi after vacating; false lets ground clear it. "+
			"With turnaround it stays on its stand and departs again after dwell_min. Runway in use, stand, STAR and model "+
			"are chosen when not given. Returns the aircraft; follow it with list_our_traffic and get_landing_sequence.").
		StringParam("icao", "Airport ICAO code (required).").
		StringParam("callsign", "Call sign, e.g. \"DLH4AB\" (required).").
		StringParam("runway", "Landing runway (default: in use).").
		StringParam("stand", "Stand label (default: a free stand that fits).").
		StringParam("star", "STAR name, \"auto\" (default) or \"none\" (straight in on final).").
		StringParam("model", "Model title from list_aircraft_models.").
		StringParam("aircraft_type", "ICAO type to pick a model by (when no model is given).").
		StringParam("via", "Taxiways to follow to the stand, in order.").
		BoolParam("hold_for_clearance", "Wait for the taxi clearance after vacating (default true).").
		BoolParam("turnaround", "Stay on the stand and depart again (default false).").
		NumberParam("dwell_min", "With turnaround: minutes on the stand before departing (default: the engine's).").
		Required("icao", "callsign").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		req, _, bad := spawnRequest(ctx, w, args, "arrival")
		if bad != nil {
			return bad, nil
		}
		req.InjectApproach = true
		req.Turnaround = boolArg(args, "turnaround", false)
		req.DwellSec = numArg(args, "dwell_min", 0) * 60
		procedureArg(&req, strArg(args, "star"))
		return spawn(w, req, boolArg(args, "hold_for_clearance", true))
	})
}

func registerWorldListOurTraffic(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("list_our_traffic").
		Description("List the AI aircraft of ours (spawned or scheduled): call sign, kind, airport, model, stand, runway, "+
			"procedure, state, the position and frequency working it, position and speed, what it holds short of, its ground "+
			"vehicles (tug, fuel truck, stairs, GPU) and their state, whether it is manual (waiting for atc_clearance) and "+
			"actions — the clearances atc_clearance takes now. Real-world aircraft (set_real_traffic) are marked real.").
		StringParam("icao", "Only this airport's aircraft.").
		BoolParam("detail", "Include routes, taxi nodes and air fixes (default false).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"count": 0, "aircraft": []any{}, "engine": "not started"})
		}
		list, err := ourAircraft(w)
		if err != nil {
			return worldError("our traffic", err), nil
		}
		icao := strings.ToUpper(strArg(args, "icao"))
		detail := boolArg(args, "detail", false)
		out := []map[string]any{}
		for _, a := range list {
			if icao != "" && a.ICAO != icao {
				continue
			}
			out = append(out, aircraftView(a, detail))
		}
		return mcpadapter.JSONResult(map[string]any{"count": len(out), "aircraft": out})
	})
}

func registerWorldClearance(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("atc_clearance").
		Description("Give one of our AI aircraft a clearance or instruction (its actions in list_our_traffic list what fits "+
			"now). Departures: pushback (facing, startup), pushstart, startup, taxi, upto (a taxi node), cross, lineup, "+
			"lineupbehind, takeoff, hold, abort, entry (an intersection, \"\" full length), rush. Arrivals: land, hold, "+
			"goaround, taxi, upto, cross, standto (another stand), depart (a turnaround now). Both: manual (on: the caller "+
			"clears it; off: the engine's ATC does), remove (out of the simulator). Any action but remove puts it under "+
			"manual control.").
		StringParam("callsign", "Call sign of one of ours (required).").
		StringParam("action", "The action (required), e.g. pushback, taxi, lineup, takeoff, goaround, remove.").
		StringParam("stand", "standto: the stand label.").
		StringParam("entry", "entry: the entry taxiway (\"\" full length).").
		NumberParam("node", "upto: the taxi node to stop at.").
		StringParam("facing", "pushback: the direction to face after the push, e.g. \"east\".").
		BoolParam("startup", "pushback: start engines during the push.").
		BoolParam("on", "manual, rush: on (default true) or off.").
		Required("callsign", "action").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_FOUND: no traffic of ours: the engine has not started"), nil
		}
		a, err := findOurs(w, strArg(args, "callsign"))
		if err != nil {
			return worldError("clearance", err), nil
		}
		action := strings.ToLower(strArg(args, "action"))
		if action == "" || strings.ContainsAny(action, "/?&") {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: action is required"), nil
		}
		q := url.Values{}
		switch action {
		case "standto":
			i, err := standIndex(w, a.ICAO, strArg(args, "stand"))
			if err != nil {
				return worldError("stand", err), nil
			}
			q.Set("stand", strconv.Itoa(i))
		case "entry":
			q.Set("entry", strings.ToUpper(strArg(args, "entry")))
		case "upto":
			q.Set("node", strconv.Itoa(int(numArg(args, "node", -1))))
		case "pushback":
			if f := strArg(args, "facing"); f != "" {
				q.Set("facing", f)
			}
			if boolArg(args, "startup", false) {
				q.Set("startup", "1")
			}
		case "manual", "rush":
			q.Set("on", map[bool]string{true: "1", false: "0"}[boolArg(args, "on", true)])
		}
		path := fmt.Sprintf("/api/control/%d/%s", a.ID, action)
		if len(q) > 0 {
			path += "?" + q.Encode()
		}
		if _, err := w.Do("POST", path, nil); err != nil {
			return worldError(action+" "+a.Tail, err), nil
		}
		if action == "remove" {
			return mcpadapter.JSONResult(map[string]any{"callsign": a.Tail, "removed": true})
		}
		if now, err := findOurs(w, a.Tail); err == nil {
			return mcpadapter.JSONResult(aircraftView(now, false))
		}
		return mcpadapter.JSONResult(map[string]any{"callsign": a.Tail, "action": action, "sent": true})
	})
}

func registerWorldPicture(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_traffic_picture").
		Description("The traffic picture: every aircraft the simulator has around the user aircraft (or an airport) with call "+
			"sign, title, position, altitude, ground speed, heading, vertical speed, phase (parked, taxiing, runway, "+
			"departing, enroute, arriving) and the airport it belongs to; ours and the user's are marked. The first call "+
			"starts the engine's scan and can take a few seconds.").
		StringParam("centre", "Airport ICAO code to centre on (default: the user aircraft).").
		NumberParam("radius_nm", "Radius, NM (default 40, at most 40).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		centre := ""
		if strArg(args, "centre") != "" {
			var bad *mcpadapter.CallToolResult
			if centre, bad = icaoArg(args, "centre"); bad != nil {
				return bad, nil
			}
		}
		radius := numArg(args, "radius_nm", 40)
		if radius <= 0 || radius > 40 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: radius_nm must be 1–40"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		// Read the picture as the engine keeps it: its area is also where
		// it works, so the tool doesn't move it; the radius is ours.
		var v struct {
			Airports []traffic.AirportRef      `json:"airports"`
			Aircraft []traffic.TrackedAircraft `json:"aircraft"`
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			if err := w.Get("/api/world", &v); err != nil {
				return worldError("traffic picture", err), nil
			}
			if len(v.Aircraft) > 0 || time.Now().After(deadline) || ctx.Err() != nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		var at *airport.LatLon
		for _, a := range v.Airports {
			if centre != "" && a.ICAO == centre {
				p := a.Position
				at = &p
			}
		}
		for _, a := range v.Aircraft {
			if centre == "" && a.User {
				p := a.Position
				at = &p
			}
		}
		if at == nil {
			if centre != "" {
				return mcpadapter.ErrorResult("NOT_FOUND: " + centre + " is not in the engine's traffic picture: it must be loaded around the user aircraft"), nil
			}
			return mcpadapter.ErrorResult("NOT_FOUND: the user aircraft is not in the traffic picture yet: try again in a few seconds"), nil
		}
		type aircraft struct {
			Callsign  string  `json:"callsign,omitempty"`
			Title     string  `json:"title,omitempty"`
			Phase     string  `json:"phase"`
			Airport   string  `json:"airport,omitempty"`
			Lat       float64 `json:"lat"`
			Lon       float64 `json:"lon"`
			AltFt     float64 `json:"alt_ft"`
			AGLFt     float64 `json:"agl_ft"`
			GroundKts float64 `json:"ground_speed_kts"`
			Heading   float64 `json:"heading_true"`
			VSFpm     float64 `json:"vs_fpm"`
			User      bool    `json:"user,omitempty"`
			Ours      bool    `json:"ours,omitempty"`
		}
		out := make([]aircraft, 0, len(v.Aircraft))
		for _, a := range v.Aircraft {
			if calc.HaversineNM(at.Lat, at.Lon, a.Position.Lat, a.Position.Lon) > radius {
				continue
			}
			out = append(out, aircraft{a.Tail, a.Title, string(a.Phase), a.Airport, round(a.Position.Lat, 5), round(a.Position.Lon, 5),
				round(a.AltFt, -1), round(a.AGLFt, 0), round(a.GroundKts, 0), round(a.Heading, 0), round(a.VSFpm, -1), a.User, a.Ours})
		}
		return mcpadapter.JSONResult(map[string]any{"centre": centre, "radius_nm": radius, "count": len(out), "aircraft": out})
	})
}

// trafficAirports are the airports the engine works now: the schedule's
// and those of our aircraft.
func trafficAirports(w live.World) []string {
	seen := map[string]bool{}
	var s struct {
		Airports []string `json:"airports"`
	}
	_ = w.Get("/api/schedule", &s)
	for _, a := range s.Airports {
		seen[a] = true
	}
	if list, err := ourAircraft(w); err == nil {
		for _, a := range list {
			if a.ICAO != "" {
				seen[a.ICAO] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

func airportsArg(w live.World, args map[string]any) ([]string, *mcpadapter.CallToolResult) {
	if strArg(args, "icao") == "" {
		return trafficAirports(w), nil
	}
	icao, bad := icaoArg(args, "icao")
	if bad != nil {
		return nil, bad
	}
	return []string{icao}, nil
}

func registerWorldLandingSequence(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_landing_sequence").
		Description("The landing sequence of each runway at an airport the traffic engine works: for every arrival its "+
			"number, the one ahead, the spacing it keeps and why (wake, minimum, low visibility), its distance to go, "+
			"predicted and sequenced landing, and its delay; with the approach conditions and low-visibility procedures. "+
			"The engine's approach controller loses delays (speed, vectors, holds); approach_instruction steers it.").
		StringParam("icao", "Airport ICAO code (default: every airport the engine works).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"airports": []any{}, "engine": "not started"})
		}
		airports, bad := airportsArg(w, args)
		if bad != nil {
			return bad, nil
		}
		out := []map[string]any{}
		for _, icao := range airports {
			var seq []json.RawMessage
			if err := w.Get("/api/sequence?icao="+url.QueryEscape(icao), &seq); err != nil {
				return worldError("sequence at "+icao, err), nil
			}
			out = append(out, map[string]any{"icao": icao, "runways": seq})
		}
		return mcpadapter.JSONResult(map[string]any{"airports": out,
			"note": "delay is in nanoseconds; eta and landing are traffic time (get_schedule's now)"})
	})
}

var worldApproachActions = []string{"up", "down", "slow", "speed", "hold", "release", "direct", "joinfinal", "holdat", "goaround"}

func registerWorldApproach(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("approach_instruction").
		Description("Give one of our arrivals an approach controller's instruction: up / down (a place earlier or later in "+
			"the landing order), slow (lose another minute), speed (kts; 0 resumes normal speed), hold (at its STAR's hold "+
			"fix) and release, direct (to a point given, else to the final), joinfinal (join the final at a point), holdat "+
			"(hold at a point), goaround. Established arrivals keep their place.").
		StringParam("callsign", "Our arrival's call sign (required).").
		StringParam("instruction", strings.Join(worldApproachActions, ", ")+" (required).").
		NumberParam("kts", "speed: the speed, kts.").
		NumberParam("lat", "direct, joinfinal, holdat: the point's latitude.").
		NumberParam("lon", "direct, joinfinal, holdat: the point's longitude.").
		Required("callsign", "instruction").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_FOUND: no traffic of ours: the engine has not started"), nil
		}
		action := strings.ToLower(strArg(args, "instruction"))
		ok := false
		for _, a := range worldApproachActions {
			ok = ok || a == action
		}
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: instruction must be one of " + strings.Join(worldApproachActions, ", ")), nil
		}
		a, err := findOurs(w, strArg(args, "callsign"))
		if err != nil {
			return worldError("approach", err), nil
		}
		body := map[string]any{"kts": numArg(args, "kts", 0)}
		lat, hasLat := args["lat"].(float64)
		lon, hasLon := args["lon"].(float64)
		if hasLat && hasLon {
			body["lat"], body["lon"] = lat, lon
		} else if action == "joinfinal" || action == "holdat" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: " + action + " needs lat and lon"), nil
		}
		path := fmt.Sprintf("/api/approach/%s/%s/%s", url.PathEscape(a.ICAO), url.PathEscape(a.Tail), action)
		if _, err := w.Do("POST", path, body); err != nil {
			return worldError(action+" "+a.Tail, err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"callsign": a.Tail, "icao": a.ICAO, "instruction": action, "sent": true})
	})
}

func registerWorldATCLog(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_atc_log").
		Description("The radio: the latest transmissions of the traffic engine's controllers and crews, oldest first, with "+
			"time, airport, frequency, position (delivery, ground, tower, approach), controller or pilot, call sign, intent "+
			"and text.").
		StringParam("icao", "Only this airport's.").
		NumberParam("limit", "At most this many (default 30).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"count": 0, "transmissions": []any{}, "engine": "not started"})
		}
		q := url.Values{"n": {strconv.Itoa(int(numArg(args, "limit", 30)))}}
		if icao := strings.ToUpper(strArg(args, "icao")); icao != "" {
			q.Set("icao", icao)
		}
		var list []traffic.Transmission
		if err := w.Get("/api/radio?"+q.Encode(), &list); err != nil {
			return worldError("radio", err), nil
		}
		if list == nil {
			list = []traffic.Transmission{}
		}
		return mcpadapter.JSONResult(map[string]any{"count": len(list), "transmissions": list})
	})
}

func registerWorldConflicts(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_conflicts").
		Description("Airborne separation as the traffic engine sees it: the minimum in force, the closest pairs, open and " +
			"past losses of separation, predicted conflicts, and the resolutions given to ours with what was said.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"engine": "not started"})
		}
		var v json.RawMessage
		if err := w.Get("/api/separation", &v); err != nil {
			return worldError("separation", err), nil
		}
		return rawResult(v), nil
	})
}

func registerWorldStartSchedule(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("start_schedule").
		Description("Run scheduled traffic at airports with the traffic engine's ATC: an airline timetable (and light "+
			"aircraft in daylight), departures with stand services, pushback, taxi and their SID, arrivals en route and on "+
			"their STAR, turnarounds, overflights, sequencing and separation. Calling it again changes the settings. The "+
			"engine keeps it across a simulator reconnect.").
		StringParam("airports", "Airports, e.g. \"LKPR\" or \"LKPR, LKTB\" (required); loaded around the user aircraft.").
		NumberParam("density", "Traffic density (default 1: the timetable as it is).").
		NumberParam("max_aircraft", "Most of the schedule's aircraft at once (default: no limit).").
		NumberParam("seed", "Random seed.").
		BoolParam("ifr", "Airline flights (default true).").
		BoolParam("vfr", "Light aircraft in the circuit (default true).").
		BoolParam("generator", "The generated timetable (default true); false: only flights from add_flights.").
		NumberParam("offset_min", "Fly the timetable this many minutes later now, e.g. 600 puts a morning wave into an evening.").
		StringParam("others", "\"respect\" (default) or \"ignore\" the traffic that is not ours.").
		Required("airports").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		var airports []string
		for _, a := range listArg(args, "airports") {
			a = strings.ToUpper(a)
			if !airportICAORe.MatchString(a) {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: airports must be ICAO codes, e.g. \"LKPR, LKTB\""), nil
			}
			airports = append(airports, a)
		}
		if len(airports) == 0 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: airports is required"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		s := world.ScheduleSettings{Enabled: true, ICAO: airports[0], Airports: airports, Density: numArg(args, "density", 0),
			MaxAircraft: int(numArg(args, "max_aircraft", 0)), Seed: uint64(numArg(args, "seed", 0)), Others: strArg(args, "others")}
		for key, p := range map[string]**bool{"ifr": &s.IFR, "vfr": &s.VFR, "generator": &s.Generator} {
			if b, ok := args[key].(bool); ok {
				*p = &b
			}
		}
		if v, ok := args["offset_min"].(float64); ok {
			s.OffsetMin = &v
		}
		if err := w.SetSchedule(s); err != nil {
			return worldError("schedule", err), nil
		}
		return scheduleStatus(w)
	})
}

// scheduleStatus is the schedule's settings without its flights.
func scheduleStatus(w live.World) (*mcpadapter.CallToolResult, error) {
	var v map[string]any
	if err := w.Get("/api/schedule", &v); err != nil {
		return worldError("schedule", err), nil
	}
	if f, ok := v["flights"].([]any); ok {
		v["flight_count"] = len(f)
	}
	delete(v, "flights")
	return mcpadapter.JSONResult(v)
}

func registerWorldStopSchedule(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("stop_schedule").
		Description("Stop the scheduled traffic: no more aircraft appear. Those flying finish their flights; with remove=true "+
			"every aircraft of ours on the engine's list is taken out of the simulator now.").
		BoolParam("remove", "Take our aircraft out now (default false).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"running": false, "removed": 0})
		}
		if err := w.SetSchedule(world.ScheduleSettings{Enabled: false}); err != nil {
			return worldError("schedule", err), nil
		}
		removed := 0
		if boolArg(args, "remove", false) {
			list, err := ourAircraft(w)
			if err != nil {
				return worldError("our traffic", err), nil
			}
			for _, a := range list {
				if _, err := w.Do("POST", fmt.Sprintf("/api/control/%d/remove", a.ID), nil); err == nil {
					removed++
				}
			}
		}
		return mcpadapter.JSONResult(map[string]any{"running": false, "removed": removed})
	})
}

func registerWorldGetSchedule(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_schedule").
		Description("The scheduled traffic: its settings, traffic time now (what add_flights' std and sta are in), and per "+
			"airport the departure and arrival boards — call sign, type, origin, destination, STD/STA, status, stand, "+
			"runway, estimate, note.").
		StringParam("icao", "Only this airport's boards (default: every scheduled airport).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"enabled": false, "engine": "not started"})
		}
		var status map[string]any
		if err := w.Get("/api/schedule", &status); err != nil {
			return worldError("schedule", err), nil
		}
		delete(status, "flights")
		airports, bad := airportsArg(w, args)
		if bad != nil {
			return bad, nil
		}
		boards := map[string]json.RawMessage{}
		for _, icao := range airports {
			var b json.RawMessage
			if err := w.Get("/api/boards?icao="+url.QueryEscape(icao), &b); err != nil {
				return worldError("boards at "+icao, err), nil
			}
			boards[icao] = b
		}
		status["boards"] = boards
		return mcpadapter.JSONResult(status)
	})
}

// objectsArg reads an array-of-objects argument given as an array or as
// a JSON string.
func objectsArg(args map[string]any, key string, v any) error {
	raw := args[key]
	if s, ok := raw.(string); ok {
		return json.Unmarshal([]byte(s), v)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func registerWorldAddFlights(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("add_flights").
		Description("Add flights to the running schedule at chosen times, e.g. an arrival just before the user's ETA or a "+
			"departure just after their off-block (start_schedule with generator=false runs only these). Each flight: "+
			"callsign, origin, destination (one of them a scheduled airport), type (default A320), airline, and its times as "+
			"std_in_min / sta_in_min (minutes from traffic time now) or std / sta (RFC 3339, traffic time). A departure "+
			"appears on its stand before its STD, an arrival in time for its STA; an STA too soon is refused.").
		ObjectArrayParam("flights", "The flights (required): [{\"callsign\":\"CSA7\",\"origin\":\"EDDM\",\"destination\":\"LKPR\",\"type\":\"A320\",\"sta_in_min\":30}].").
		Required("flights").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		type in struct {
			traffic.Flight
			STDInMin *float64 `json:"std_in_min"`
			STAInMin *float64 `json:"sta_in_min"`
		}
		var list []in
		if err := objectsArg(args, "flights", &list); err != nil || len(list) == 0 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: flights must be an array of flight objects"), nil
		}
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_APPLICABLE: no schedule runs: start_schedule first"), nil
		}
		var now struct {
			Now time.Time `json:"now"`
		}
		if err := w.Get("/api/schedule", &now); err != nil {
			return worldError("schedule", err), nil
		}
		flights := make([]traffic.Flight, 0, len(list))
		for _, f := range list {
			fl := f.Flight
			fl.Callsign = strings.ToUpper(fl.Callsign)
			fl.Origin, fl.Destination = strings.ToUpper(fl.Origin), strings.ToUpper(fl.Destination)
			if f.STDInMin != nil {
				fl.STD = now.Now.Add(time.Duration(*f.STDInMin * float64(time.Minute)))
			}
			if f.STAInMin != nil {
				fl.STA = now.Now.Add(time.Duration(*f.STAInMin * float64(time.Minute)))
			}
			flights = append(flights, fl)
		}
		b, err := w.Do("POST", "/api/flights", flights)
		if err != nil {
			return worldError("flights", err), nil
		}
		var added []traffic.Flight
		_ = live.DecodeJSON(b, &added)
		return mcpadapter.JSONResult(map[string]any{"now": now.Now, "added": added})
	})
}

func registerWorldRealTraffic(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("set_real_traffic").
		Description("Fly real-world traffic at an airport instead of the generated timetable: the engine flies the aircraft "+
			"observe_traffic reports (from a feed such as ADS-B), each with stand services, push, taxi, ATC and radio. "+
			"Generated flights not yet in the simulator go at once; those flying finish. on=false brings the timetable back.").
		BoolParam("on", "On (default true) or off.").
		StringParam("icao", "The airport (required when on).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		on := boolArg(args, "on", true)
		icao := ""
		if on {
			var bad *mcpadapter.CallToolResult
			if icao, bad = icaoArg(args, "icao"); bad != nil {
				return bad, nil
			}
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		if on {
			if err := loadAirport(w, icao); err != nil {
				return worldError("loading "+icao, err), nil
			}
		}
		if err := w.SetRealTraffic(on, icao); err != nil {
			return worldError("real traffic", err), nil
		}
		var v json.RawMessage
		if err := w.Get("/api/realtraffic", &v); err != nil {
			return worldError("real traffic", err), nil
		}
		return rawResult(v), nil
	})
}

func registerWorldObserve(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("observe_traffic").
		Description("Feed real-world sightings to the engine (after set_real_traffic): each aircraft seen, by its ICAO 24-bit "+
			"address. Parked and departing aircraft go on a stand near where they are seen, arrivals appear on their "+
			"projected track and join a STAR; overflights are not flown. An ID is spawned once; later sightings only refresh "+
			"it. drop ends aircraft the feed no longer sees (one in progress finishes first). Unknown origins and "+
			"destinations stay unknown.").
		ObjectArrayParam("sightings", "Sightings: [{\"id\":\"49d2a1\",\"callsign\":\"CSA7\",\"registration\":\"OK-TVR\",\"type\":\"B738\",\"lat\":50.1,\"lon\":14.26,\"altFt\":1200,\"groundKts\":150,\"trackDeg\":240,\"vsFpm\":-700,\"onGround\":false,\"seenAt\":\"2026-10-07T13:00:00Z\",\"origin\":\"EGLL\",\"destination\":\"LKPR\"}].").
		StringParam("drop", "IDs to drop, e.g. \"49d2a1, 4ca7b2\".").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_APPLICABLE: real traffic is off: set_real_traffic first"), nil
		}
		out := map[string]any{}
		if args["sightings"] != nil {
			var obs []traffic.Observed
			if err := objectsArg(args, "sightings", &obs); err != nil {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: sightings must be an array of sighting objects: " + err.Error()), nil
			}
			for i := range obs {
				if obs[i].SeenAt.IsZero() {
					obs[i].SeenAt = time.Now().UTC()
				}
			}
			b, err := w.Do("POST", "/api/realflights", obs)
			if err != nil {
				return worldError("sightings", err), nil
			}
			var res []world.ObserveResult
			_ = live.DecodeJSON(b, &res)
			out["results"] = res
		}
		var dropped []string
		for _, id := range listArg(args, "drop") {
			if _, err := w.Do("DELETE", "/api/realflights/"+url.PathEscape(id), nil); err != nil {
				return worldError("drop "+id, err), nil
			}
			dropped = append(dropped, id)
		}
		if dropped != nil {
			out["dropped"] = dropped
		}
		if len(out) == 0 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: give sightings, drop or both"), nil
		}
		return mcpadapter.JSONResult(out)
	})
}

// routeArg reads "lat,lon; lat,lon; …".
func routeArg(s string) ([]airport.LatLon, error) {
	var out []airport.LatLon
	for _, p := range strings.Split(s, ";") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		ll := strings.Split(p, ",")
		if len(ll) != 2 {
			return nil, fmt.Errorf("%q is not lat,lon", p)
		}
		lat, e1 := strconv.ParseFloat(strings.TrimSpace(ll[0]), 64)
		lon, e2 := strconv.ParseFloat(strings.TrimSpace(ll[1]), 64)
		if e1 != nil || e2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return nil, fmt.Errorf("%q is not lat,lon", p)
		}
		out = append(out, airport.LatLon{Lat: lat, Lon: lon})
	}
	return out, nil
}

func registerWorldCorridor(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("set_traffic_corridor").
		Description("Keep a few airliners around the user's flight in cruise: same (ahead on the route, 25–45 NM, same "+
			"direction, 2000 ft above or below), opposite (70–100 NM ahead, coming the other way, 1000 ft above or below) "+
			"and crossing (45–70 NM ahead, across the route). They are flown by MSFS AI and replaced once far behind. "+
			"enabled=false takes them all out. Returns the corridor and its aircraft.").
		BoolParam("enabled", "On (default true) or off.").
		StringParam("route", "The user's route ahead, in its direction: \"lat,lon; lat,lon; …\" (at least two points; required when on).").
		NumberParam("level_ft", "The user's cruise level, feet (required when on).").
		NumberParam("kts", "The user's cruise speed, kts.").
		NumberParam("same", "How many going the same way (default 1).").
		NumberParam("opposite", "How many coming the other way (default 1).").
		NumberParam("crossing", "How many crossing (default 1).").
		NumberParam("despawn_nm", "Take one out this far from the user aircraft, moving away (default 80).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		c := world.CorridorSettings{Enabled: boolArg(args, "enabled", true), LevelFt: numArg(args, "level_ft", 0),
			Kts: numArg(args, "kts", 0), DespawnNM: numArg(args, "despawn_nm", 0)}
		if c.Enabled {
			route, err := routeArg(strArg(args, "route"))
			if err != nil || len(route) < 2 || c.LevelFt <= 0 {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: route needs at least two \"lat,lon\" points and level_ft a level"), nil
			}
			c.Route = route
		}
		for key, p := range map[string]**int{"same": &c.Same, "opposite": &c.Opposite, "crossing": &c.Crossing} {
			if v, ok := args[key].(float64); ok {
				n := int(v)
				*p = &n
			}
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		if err := w.SetCorridor(c); err != nil {
			return worldError("corridor", err), nil
		}
		var v json.RawMessage
		if err := w.Get("/api/corridor", &v); err != nil {
			return worldError("corridor", err), nil
		}
		return rawResult(v), nil
	})
}

var playerPhases = []string{"pushback", "taxi", "holding_short", "lineup", "takeoff", "landing", "vacated"}

func registerWorldPlayer(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("set_player_clearance").
		Description("Tell the traffic engine what the user's own ATC cleared the user aircraft to do (the engine never "+
			"controls or calls it): while the user lines up, takes off or lands on a runway, none of our traffic is cleared "+
			"onto it; landing, the user is put in that runway's landing sequence and our traffic fits around it. vacated "+
			"ends it. Returns the user's place in the sequence when there is one.").
		StringParam("icao", "The airport (required).").
		StringParam("runway", "The runway end, e.g. \"24\" (required).").
		StringParam("phase", strings.Join(playerPhases, ", ")+" (required).").
		StringParam("callsign", "The user's call sign as said (default \"Player\").").
		StringParam("model", "The user's model, for its wake and speed.").
		Required("icao", "runway", "phase").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		phase := strings.ToLower(strArg(args, "phase"))
		ok := false
		for _, p := range playerPhases {
			ok = ok || p == phase
		}
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: phase must be one of " + strings.Join(playerPhases, ", ")), nil
		}
		c := world.PlayerClearance{ICAO: icao, Runway: strings.ToUpper(strArg(args, "runway")), Phase: world.PlayerPhase(phase),
			Callsign: strArg(args, "callsign"), Model: strArg(args, "model")}
		if _, err := w.Do("POST", "/api/player", c); err != nil {
			return worldError("player clearance", err), nil
		}
		out := map[string]any{"clearance": c}
		if p := w.Snapshot().Player; p != nil {
			out["place"] = p
		}
		return mcpadapter.JSONResult(out)
	})
}

func registerWorldStatus(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_traffic_status").
		Description("The traffic engine's state: whether it runs, how many aircraft of ours, the schedule, real traffic and " +
			"corridor settings, the user's place in a landing sequence, messages dropped, and the last error.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		out := map[string]any{"running": w.Running()}
		if e := w.Error(); e != "" {
			out["error"] = e
		}
		if !w.Running() {
			return mcpadapter.JSONResult(out)
		}
		snap := w.Snapshot()
		out["connected"], out["aircraft"], out["dropped_messages"] = snap.Connected, len(snap.Aircraft), snap.Dropped
		if snap.Player != nil {
			out["player"] = snap.Player
		}
		for key, path := range map[string]string{"schedule": "/api/schedule", "real_traffic": "/api/realtraffic", "corridor": "/api/corridor"} {
			var v map[string]any
			if w.Get(path, &v) == nil {
				delete(v, "flights")
				out[key] = v
			}
		}
		return mcpadapter.JSONResult(out)
	})
}

func registerWorldAirportInfo(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_traffic_airport_info").
		Description("An airport as the traffic engine works it: runways, the runways in use with head- and crosswind, the "+
			"ATIS (letter, text, spoken) the engine's controllers refer to, the ILS, limits and weather. While our traffic "+
			"runs at an airport, this is what its ATC uses.").
		StringParam("icao", "Airport ICAO code (required).").
		Required("icao").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		if err := loadAirport(w, icao); err != nil {
			return worldError("loading "+icao, err), nil
		}
		var v json.RawMessage
		if err := w.Get("/api/airportinfo?icao="+url.QueryEscape(icao), &v); err != nil {
			return worldError("airport info", err), nil
		}
		return rawResult(v), nil
	})
}

// rawResult passes the engine's JSON answer on; an empty answer is {}.
func rawResult(v json.RawMessage) *mcpadapter.CallToolResult {
	if len(v) == 0 {
		v = json.RawMessage("{}")
	}
	return mcpadapter.TextResult(string(v))
}

func registerWorldTCAS(mcp *mcpadapter.Server, w live.World) {
	tool := mcpadapter.NewTool("get_tcas").
		Description("TCAS II for our airborne traffic: each of ours sees every aircraft around it (ours, the user's, other "+
			"traffic) within 12 NM, gets traffic (TA) and resolution advisories (RA), flies an RA after the crew's reaction "+
			"time and reports it on the frequency. Returns how many TAs and RAs there are now (ta, ra) and the latest "+
			"advisories as they happened (events: at, callsign, intruder, advisory TA/RA/clear, aural, rangeNM, dzFt). Each "+
			"aircraft's advisory now is also on list_our_traffic as tcas.").
		NumberParam("limit", "At most this many latest events (default 30).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if !w.Running() {
			return mcpadapter.JSONResult(map[string]any{"ta": 0, "ra": 0, "events": []any{}, "engine": "not started"})
		}
		var v struct {
			TA     int               `json:"ta"`
			RA     int               `json:"ra"`
			Events []json.RawMessage `json:"events"`
		}
		if err := w.Get("/api/tcas", &v); err != nil {
			return worldError("tcas", err), nil
		}
		limit := int(numArg(args, "limit", 30))
		if limit > 0 && len(v.Events) > limit {
			v.Events = v.Events[len(v.Events)-limit:]
		}
		if v.Events == nil {
			v.Events = []json.RawMessage{}
		}
		return mcpadapter.JSONResult(v)
	})
}
