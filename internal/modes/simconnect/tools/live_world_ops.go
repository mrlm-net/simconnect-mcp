//go:build windows

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/traffic/world"
)

// A supervisor's controls over the traffic engine: emergencies, the
// runway in use, stations a player staffs, what a clearance would say, and
// whether the ground vehicles talk.

func registerWorldOperations(mcp *mcpadapter.Server, w live.World) {
	emergency := mcpadapter.NewTool("declare_emergency").
		Description("One of our aircraft declares an emergency (squawk 7700, MAYDAY or PAN PAN, priority and rescue vehicles), or cancels it.").
		StringParam("callsign", "Our aircraft's call sign (required)").
		StringParam("kind", "engine-failure, medical, fuel, pressurisation, fire or hydraulic (default engine-failure)").
		StringParam("level", "mayday or panpan (default by the kind)").
		BoolParam("on", "false cancels the emergency (default true)").
		Required("callsign").
		Build()
	mcp.AddTool(emergency, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		cs := strings.ToUpper(strings.TrimSpace(strArg(args, "callsign")))
		if cs == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: callsign is required"), nil
		}
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_FOUND: no traffic of ours: the engine has not started"), nil
		}
		kind := strings.ToLower(strings.TrimSpace(strArg(args, "kind")))
		if kind == "" {
			kind = "engine-failure"
		}
		on := boolArg(args, "on", true)
		body := map[string]any{"tail": cs, "kind": kind, "level": strArg(args, "level"), "on": on}
		if _, err := w.Do("POST", "/api/emergency", body); err != nil {
			return worldError("emergency "+cs, err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"callsign": cs, "kind": kind, "emergency": on})
	})

	use := mcpadapter.NewTool("set_runway_in_use").
		Description("Make a runway end the runway in use at an airport whatever the wind, or give the choice back to the wind; the change is planned like any runway change.").
		StringParam("icao", "Airport ICAO (required)").
		StringParam("runway", "Runway end, e.g. 24; empty gives it back to the wind").
		Required("icao").
		Build()
	mcp.AddTool(use, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao := strings.ToUpper(strings.TrimSpace(strArg(args, "icao")))
		if icao == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao is required"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		b, err := w.Do("POST", "/api/runway-use", map[string]any{"icao": icao, "runway": strings.ToUpper(strings.TrimSpace(strArg(args, "runway")))})
		if err != nil {
			return worldError("runway in use", err), nil
		}
		return mcpadapter.TextResult(string(b)), nil
	})

	getStaffed := mcpadapter.NewTool("get_staffed_stations").
		Description("The ATC stations a player works; the AI gives no clearances to aircraft on them.").
		Build()
	mcp.AddTool(getStaffed, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var list []world.StaffedStation
		if err := w.Get("/api/stations/staffed", &list); err != nil {
			return worldError("staffed stations", err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"staffed": list})
	})

	setStaffed := mcpadapter.NewTool("set_staffed_stations").
		Description("Set the ATC stations a player works (replaces the list; empty: all automatic). Aircraft on them are the player's to clear.").
		ObjectArrayParam("stations", "Each {icao, position (tower, ground, approach, …) or station (its name), mode (assist, silent, safetynet), autoHandoff}").
		Required("stations").
		Build()
	mcp.AddTool(setStaffed, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		raw, ok := args["stations"].([]any)
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: stations (a list, empty for none) is required"), nil
		}
		b, _ := json.Marshal(raw)
		var list []world.StaffedStation
		if err := json.Unmarshal(b, &list); err != nil {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: stations: " + err.Error()), nil
		}
		for i, s := range list {
			if strings.TrimSpace(s.ICAO) == "" || strings.TrimSpace(s.Position)+strings.TrimSpace(s.Station) == "" {
				return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: stations[%d] needs icao and a position or station", i)), nil
			}
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		out, err := w.Do("PUT", "/api/stations/staffed", list)
		if err != nil {
			return worldError("staffed stations", err), nil
		}
		return mcpadapter.TextResult(string(out)), nil
	})

	phrase := mcpadapter.NewTool("preview_clearance").
		Description("What giving one of our aircraft a clearance would say, controller and readback, without giving it.").
		StringParam("callsign", "Our aircraft's call sign (required)").
		StringParam("action", "A clearance from its actions, e.g. pushback, taxi, lineup, takeoff, land (required)").
		Required("callsign", "action").
		Build()
	mcp.AddTool(phrase, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		action := strings.ToLower(strings.TrimSpace(strArg(args, "action")))
		if action == "" || strings.ContainsAny(action, "/?&") {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: action is required"), nil
		}
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_FOUND: no traffic of ours: the engine has not started"), nil
		}
		a, err := findOurs(w, strArg(args, "callsign"))
		if err != nil {
			return worldError("clearance", err), nil
		}
		var p world.ClearancePhrase
		if err := w.Get(fmt.Sprintf("/api/control/%d/%s/phrase", a.ID, action), &p); err != nil {
			return worldError(action+" "+a.Tail, err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"callsign": a.Tail, "action": action, "phrase": p})
	})

	getRadio := mcpadapter.NewTool("get_vehicle_radio").
		Description("Whether the ground vehicles (tugs, fuel trucks, rescue) talk on the radio.").
		Build()
	mcp.AddTool(getRadio, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var st struct {
			On bool `json:"on"`
		}
		if err := w.Get("/api/vehicles/radio", &st); err != nil {
			return worldError("vehicle radio", err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"on": st.On})
	})

	setRadio := mcpadapter.NewTool("set_vehicle_radio").
		Description("Turn the ground vehicles' radio on or off; off they keep quiet, and a runway crossing still waits for the tower.").
		BoolParam("on", "true: they talk, false: quiet (required)").
		Required("on").
		Build()
	mcp.AddTool(setRadio, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		on, ok := args["on"].(bool)
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: on (true or false) is required"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		b, err := w.Do("POST", "/api/vehicles/radio", map[string]any{"on": on})
		if err != nil {
			return worldError("vehicle radio", err), nil
		}
		return rawResult(b), nil
	})
}
