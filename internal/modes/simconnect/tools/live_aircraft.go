//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/addons"
	"github.com/mrlm-net/simconnect/pkg/gsx"
	"github.com/mrlm-net/simconnect/pkg/systems"
)

// RegisterLiveAircraftTools registers the user aircraft tools built on the
// library's pkg/systems, pkg/avionics and pkg/addons: get_aircraft_systems,
// set_aircraft_control, request_ground_service, set_radio, set_atc_callsign
// and list_addons.
func RegisterLiveAircraftTools(mcp *mcpadapter.Server, ac live.Aircraft) {
	registerGetAircraftSystems(mcp, ac)
	registerSetAircraftControl(mcp, ac)
	registerRequestGroundService(mcp, ac)
	registerSetRadio(mcp, ac)
	registerSetATCCallsign(mcp, ac)
	registerListAddons(mcp, ac)
	registerGetGSXState(mcp, ac)
	registerSetLVar(mcp, ac)
}

// lvarNameRe is an L:var name, with or without its "L:".
var lvarNameRe = regexp.MustCompile(`^(L:)?[A-Za-z_][A-Za-z0-9_:.]{0,127}$`)

func registerGetGSXState(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("get_gsx_state").
		Description("GSX Pro's state from its L:vars: services, passengers, cargo, doors, fuel, pushback, crew, gate, " +
			"de-ice.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		st, err := ac.GSXState(ctx)
		if err != nil {
			return aircraftError("GSX state", err), nil
		}
		services := map[string]string{}
		for name, s := range map[string]gsx.Service{"boarding": st.Boarding, "deboarding": st.Deboarding,
			"catering": st.Catering, "refueling": st.Refueling, "departure": st.Departure, "deice": st.Deice} {
			services[name] = s.String()
		}
		waiting := st.WaitingFor
		if waiting == nil {
			waiting = []string{}
		}
		return mcpadapter.JSONResult(map[string]any{
			"running":  st.Running,
			"services": services,
			"passengers": map[string]int{"to_board": st.Passengers, "boarding": st.PassengersBoarding,
				"boarded_total": st.PassengersBoardingTotal, "deboarding": st.PassengersDeboarding,
				"deboarded_total": st.PassengersDeboardingTotal, "max": st.MaxPassengers},
			"cargo": map[string]any{"loading": st.LoadingCargo, "loaded_pct": round(st.CargoLoadedPct, 0),
				"unloading": st.UnloadingCargo, "unloaded_pct": round(st.CargoUnloadedPct, 0)},
			"waiting_for": waiting,
			"fuel":        map[string]any{"hose": st.FuelHose, "counter": round(st.FuelCounter, 1), "counter_max": round(st.FuelCounterMax, 1)},
			"pushback":    map[string]bool{"frozen": st.Frozen, "bypass_pin": st.BypassPin},
			"on_board":    map[string]bool{"pilots": st.PilotsOnBoard, "crew": st.CrewOnBoard},
			"gate":        st.Gate,
			"deice_fluid": st.DeiceFluid,
		})
	})
}

func registerSetLVar(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("set_lvar").
		Description("Write an L:var on the user aircraft as a number; read back with get_simvar_value (unit \"number\").").
		StringParam("name", "e.g. \"L:MY_FLAG\"").
		NumberParam("value", "Value to write").
		Required("name", "value").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		name := strArg(args, "name")
		if !lvarNameRe.MatchString(name) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: name must be an L:var name, e.g. \"L:MY_FLAG\""), nil
		}
		v, ok := args["value"].(float64)
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: value must be a number"), nil
		}
		if err := ac.SetLVar(ctx, name, v); err != nil {
			return aircraftError("L:var "+name, err), nil
		}
		if !strings.HasPrefix(name, "L:") {
			name = "L:" + name
		}
		return mcpadapter.JSONResult(map[string]any{"name": name, "value": v, "set": true})
	})
}

// aircraftError turns an Aircraft error into a tool error result.
func aircraftError(what string, err error) *mcpadapter.CallToolResult {
	switch {
	case errors.Is(err, systems.ErrNoControl):
		return mcpadapter.ErrorResult(fmt.Sprintf("NOT_APPLICABLE: %s: this aircraft's profile gives no way to operate it "+
			"(get_aircraft_systems lists what it can operate under \"can\")", what))
	case live.IsInputError(err):
		return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %s: %v", what, err))
	}
	return sourceError(what, err)
}

type doorState struct {
	Control string `json:"control"`
	Name    string `json:"name"`
	Open    bool   `json:"open"`
}

func registerGetAircraftSystems(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("get_aircraft_systems").
		Description("The user aircraft's systems through its profile: power, radios, engines, brakes, lights, doors, " +
			"transponder, gear, ground services, take-off speeds, and what can be operated.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		s, err := ac.Systems(ctx)
		if err != nil {
			return aircraftError("aircraft systems", err), nil
		}
		st := s.State
		doors := []doorState{}
		for i, name := range st.DoorNames {
			doors = append(doors, doorState{systems.Door(i), name, i < len(st.DoorsOpen) && st.DoorsOpen[i]})
		}
		engines := []map[string]any{}
		for i := 0; i < st.Engines && i < 4; i++ {
			engines = append(engines, map[string]any{"engine": i + 1, "running": st.Running[i], "starter": st.Starter[i]})
		}
		xpdr := map[int]string{0: "off", 1: "standby", 2: "test", 3: "on", 4: "alt"}[st.XPDRState]
		if xpdr == "" {
			xpdr = fmt.Sprint(st.XPDRState)
		}
		values := map[string]float64{}
		for k, v := range st.Values {
			values[k] = round(v, 3)
		}
		can, cannot := []string{}, []string{}
		for k, ok := range s.Can {
			if ok {
				can = append(can, k)
			} else {
				cannot = append(cannot, k)
			}
		}
		sort.Strings(can)
		sort.Strings(cannot)
		aircraft := map[string]any{"title": s.Title, "atc_type": s.ATCType}
		if s.Path != "" {
			aircraft["path"] = s.Path
		}
		if s.Package != nil {
			aircraft["package"] = s.Package.Folder
			aircraft["package_source"] = string(s.Package.Source)
		}
		profile := map[string]any{"name": s.Profile.Name}
		if s.Profile.Measured != "" {
			profile["measured"] = s.Profile.Measured
		}
		if len(s.Local) > 0 {
			profile["local_overrides"] = s.Local
		}
		if len(s.LocalErr) > 0 {
			profile["local_override_errors"] = s.LocalErr
		}
		out := map[string]any{
			"aircraft": aircraft,
			"profile":  profile,
			"power": map[string]any{"battery": st.Battery, "powered": st.Powered, "bus_volts": round(st.Volts, 1),
				"avionics": st.Avionics, "external_available": st.ExtAvailable, "external_on": st.ExtOn},
			"radios": map[string]any{
				"com1": map[string]any{"working": st.COM1, "active_mhz": mhz(st.COM1Active), "standby_mhz": mhz(st.COM1Standby)},
				"com2": map[string]any{"working": st.COM2, "active_mhz": mhz(st.COM2Active), "standby_mhz": mhz(st.COM2Standby)},
			},
			"transponder":   map[string]any{"state": xpdr, "squawk": st.Squawk},
			"engines":       engines,
			"parking_brake": st.ParkingBrake,
			"lights": map[string]bool{"beacon": st.Beacon, "nav": st.Nav, "strobe": st.Strobe,
				"landing": st.Landing, "taxi": st.Taxi},
			"doors":     doors,
			"flaps_pct": round(st.FlapsPct, 1),
			"gear_down": st.GearDown,
			"ground": map[string]any{"chocks": st.Chocks, "has_chocks": st.HasChocks, "gpu": st.GPU, "has_gpu": st.HasGPU,
				"pushback_attached": st.PushbackAttached, "pushback_available": st.PushbackAvailable,
				"pushback_wait": st.PushbackWait},
			"values": values,
			"can":    can,
			"cannot": cannot,
		}
		if st.SpeedsFrom != "" || st.VRKt > 0 {
			ts := map[string]any{"v1_kt": round(st.V1Kt, 0), "vr_kt": round(st.VRKt, 0), "v2_kt": round(st.V2Kt, 0),
				"source": st.SpeedsFrom, "speed_check_kt": st.SpeedCheckKt}
			if st.DAFt > 0 {
				ts["da_ft"] = round(st.DAFt, 0)
			}
			if st.MDAFt > 0 {
				ts["mda_ft"] = round(st.MDAFt, 0)
			}
			out["takeoff_speeds"] = ts
		}
		if v, ok := st.Values[systems.Seatbelts]; ok {
			out["seatbelts"] = v != 0
		}
		if v, ok := st.Values[systems.NoSmoking]; ok {
			out["no_smoking"] = int(math.Round(v))
		}
		return mcpadapter.JSONResult(out)
	})
}

// controlAliases are the friendly names set_aircraft_control takes besides
// the profile's own (door0, chocks, parkingBrake, …).
var controlAliases = map[string]string{
	"parking_brake": systems.ParkingBrake, "brake": systems.ParkingBrake,
	"seatbelts": systems.Seatbelts, "seat_belts": systems.Seatbelts,
	"no_smoking": systems.NoSmoking, "nosmoking": systems.NoSmoking,
	"ext_power": systems.ExtPower, "external_power": systems.ExtPower,
	"cabin_call": systems.CabinCall,
	"chocks":     systems.Chocks, "gpu": systems.GPU,
}

// controlName resolves name to a profile control: an alias, "door0"…, a
// door's name ("L1", "Door 2"), or a profile name as is.
func controlName(name string, doors []string) (string, bool) {
	n := strings.TrimSpace(name)
	if c, ok := controlAliases[strings.ToLower(n)]; ok {
		return c, true
	}
	for i, d := range doors {
		if strings.EqualFold(d, n) || strings.EqualFold(systems.Door(i), n) {
			return systems.Door(i), true
		}
	}
	for _, c := range []string{systems.ParkingBrake, systems.Seatbelts, systems.NoSmoking, systems.ExtPower,
		systems.CabinCall, systems.Chocks, systems.GPU} {
		if strings.EqualFold(c, n) {
			return c, true
		}
	}
	return "", false
}

func registerSetAircraftControl(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("set_aircraft_control").
		Description("Operate a user aircraft control via its profile; returns the state read back.").
		StringParam("control", "Door name or door0…, chocks, gpu, parking_brake, seatbelts, ext_power, no_smoking, cabin_call").
		StringParam("state", "on/open, off/closed (default on); no_smoking also auto").
		Required("control").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		s, err := ac.Systems(ctx)
		if err != nil {
			return aircraftError("aircraft systems", err), nil
		}
		name, ok := controlName(strArg(args, "control"), s.State.DoorNames)
		if !ok {
			return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: unknown control %q; doors here: %s",
				strArg(args, "control"), strings.Join(s.State.DoorNames, ", "))), nil
		}
		if name == systems.CabinCall {
			if err := ac.PressControl(ctx, name); err != nil {
				return aircraftError(name, err), nil
			}
			return mcpadapter.JSONResult(map[string]any{"control": name, "pressed": true})
		}
		var v float64
		switch strings.ToLower(strArg(args, "state")) {
		case "", "on", "open", "set", "connected", "true", "1":
			v = 1
			if name == systems.NoSmoking {
				v = 2
			}
		case "off", "closed", "close", "removed", "false", "0":
			v = 0
		case "auto":
			if name != systems.NoSmoking {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: state auto is for no_smoking only"), nil
			}
			v = 1
		default:
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: state must be on/open, off/closed or auto"), nil
		}
		if err := ac.SetControl(ctx, name, v); err != nil {
			return aircraftError(name, err), nil
		}
		out := map[string]any{"control": name, "requested": v}
		if after, err := ac.Systems(ctx); err == nil {
			if now, ok := after.State.Values[name]; ok {
				out["state_now"] = round(now, 3)
				out["note"] = "state_now is read right after the command; doors and tablet controls take a few seconds"
			}
		}
		return mcpadapter.JSONResult(out)
	})
}

func registerRequestGroundService(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("request_ground_service").
		Description("Ask for a sim ground service for the user aircraft; jetway, stairs and pushback toggle.").
		StringParam("service", "jetway, stairs, baggage, catering, powerSupply, fuelTruck or pushback").
		Required("service").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		want := strings.ReplaceAll(strings.ToLower(strArg(args, "service")), "_", "")
		name := ""
		for _, s := range live.ServiceNames {
			if strings.ToLower(s) == want {
				name = s
			}
		}
		if name == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: service must be one of " + strings.Join(live.ServiceNames, ", ")), nil
		}
		if err := ac.RequestService(ctx, name); err != nil {
			return aircraftError(name, err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"service": name, "requested": true})
	})
}

func registerSetRadio(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("set_radio").
		Description("Set a COM frequency, swap a COM, or set the squawk on the (powered) user aircraft.").
		StringParam("action", "com_active, com_standby, com_swap or squawk").
		NumberParam("com", "1–3 (default 1)").
		NumberParam("frequency_mhz", "MHz, e.g. 134.56").
		StringParam("squawk", "Four octal digits").
		Required("action").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		action := strings.ToLower(strArg(args, "action"))
		com := int(numArg(args, "com", 1))
		out := map[string]any{"action": action}
		var err error
		switch action {
		case "com_active", "com_standby":
			mhz := numArg(args, "frequency_mhz", 0)
			if mhz == 0 {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: frequency_mhz is required for " + action), nil
			}
			err = ac.SetCOM(ctx, com, action == "com_standby", mhz)
			out["com"], out["frequency_mhz"] = com, mhz
		case "com_swap":
			err = ac.SwapCOM(ctx, com)
			out["com"] = com
		case "squawk":
			code := strArg(args, "squawk")
			err = ac.SetSquawk(ctx, code)
			out["squawk"] = code
		default:
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: action must be com_active, com_standby, com_swap or squawk"), nil
		}
		if err != nil {
			return aircraftError(action, err), nil
		}
		out["sent"] = true
		return mcpadapter.JSONResult(out)
	})
}

func registerSetATCCallsign(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("set_atc_callsign").
		Description("Set the user aircraft's ATC airline call sign and flight number; omitted ones are kept.").
		StringParam("airline", "Spoken call sign, e.g. \"Speedbird\"").
		StringParam("flight_number", "Up to 7, e.g. \"123\"").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		airline, number := strArg(args, "airline"), strArg(args, "flight_number")
		if airline == "" && number == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: give airline, flight_number or both"), nil
		}
		if len(airline) > 63 || len(number) > 7 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: airline is at most 63 characters, flight_number at most 7"), nil
		}
		if err := ac.SetCallsign(ctx, airline, number); err != nil {
			return aircraftError("call sign", err), nil
		}
		out := map[string]any{"set": true}
		if airline != "" {
			out["airline"] = airline
		}
		if number != "" {
			out["flight_number"] = number
		}
		return mcpadapter.JSONResult(out)
	})
}

func registerListAddons(mcp *mcpadapter.Server, ac live.Aircraft) {
	tool := mcpadapter.NewTool("list_addons").
		Description("Installed MSFS packages (Community, Official, Streamed) from disk; no SimConnect needed.").
		StringParam("source", "Community (default), Community2024, Official, Streamed or all").
		StringParam("search", "Text in folder, title, creator or ICAO").
		BoolParam("refresh", "Rescan instead of the 5-min cache").
		NumberParam("limit", "Default 200").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		refresh, _ := args["refresh"].(bool)
		in, pkgs, err := ac.Addons(refresh)
		if errors.Is(err, addons.ErrNotFound) {
			return mcpadapter.ErrorResult("NOT_FOUND: no MSFS 2024 or 2020 installation found on this machine"), nil
		}
		if err != nil && len(pkgs) == 0 {
			return mcpadapter.ErrorResult(fmt.Sprintf("SIM_ERROR: add-ons: %v", err)), nil
		}
		source := strings.ToLower(strArg(args, "source"))
		if source == "" {
			source = "community"
		}
		search := strings.ToLower(strArg(args, "search"))
		limit := int(numArg(args, "limit", 200))
		if limit < 1 {
			limit = 200
		}
		type pkg struct {
			Source      string `json:"source"`
			Folder      string `json:"folder"`
			Title       string `json:"title,omitempty"`
			Creator     string `json:"creator,omitempty"`
			ContentType string `json:"content_type,omitempty"`
			Version     string `json:"version,omitempty"`
			ICAO        string `json:"icao,omitempty"`
			Publisher   string `json:"publisher,omitempty"`
			Cached      int    `json:"cached_archives,omitempty"`
		}
		counts := map[string]int{}
		list := []pkg{}
		matched := 0
		for _, p := range pkgs {
			src := string(p.Source)
			counts[src]++
			if source != "all" && !strings.HasPrefix(strings.ToLower(src), source) {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(p.Folder+" "+p.Title+" "+p.Creator+" "+p.ICAO), search) {
				continue
			}
			matched++
			if len(list) < limit {
				list = append(list, pkg{src, p.Folder, p.Title, p.Creator, p.ContentType, p.Version, p.ICAO, p.Publisher, p.Cached})
			}
		}
		return mcpadapter.JSONResult(map[string]any{
			"install":     map[string]string{"sim": in.Sim, "store": in.Store, "packages_path": in.Packages},
			"fingerprint": addons.Fingerprint(pkgs),
			"by_source":   counts,
			"matched":     matched,
			"packages":    list,
		})
	})
}

// mhz is a COM frequency as read, 0 when the radio gives none (a dark
// Fenix RMP reads -0.001).
func mhz(v float64) float64 {
	if v < 1 {
		return 0
	}
	return round(v, 3)
}
