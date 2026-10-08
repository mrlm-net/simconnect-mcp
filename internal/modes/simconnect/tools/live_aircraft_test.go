//go:build windows

package tools

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/addons"
	"github.com/mrlm-net/simconnect/pkg/avionics"
	"github.com/mrlm-net/simconnect/pkg/gsx"
	"github.com/mrlm-net/simconnect/pkg/systems"
)

// fakeAircraft is a live.Aircraft that records what was asked.
type fakeAircraft struct {
	sys      live.AircraftSystems
	set      map[string]float64
	pressed  []string
	services []string
	radio    []any
	airline  string
	number   string
	pkgs     []addons.Package
	gsxState gsx.State
	lvars    map[string]float64
}

func (f *fakeAircraft) Connected() bool { return true }
func (f *fakeAircraft) Systems(context.Context) (live.AircraftSystems, error) {
	return f.sys, nil
}
func (f *fakeAircraft) SetControl(_ context.Context, name string, v float64) error {
	if !f.sys.Can[name] {
		return systems.ErrNoControl
	}
	f.set[name] = v
	return nil
}
func (f *fakeAircraft) PressControl(_ context.Context, name string) error {
	f.pressed = append(f.pressed, name)
	return nil
}
func (f *fakeAircraft) RequestService(_ context.Context, name string) error {
	f.services = append(f.services, name)
	return nil
}
func (f *fakeAircraft) SetCOM(_ context.Context, n int, standby bool, mhz float64) error {
	if mhz < 118 || mhz > 136.99 {
		return avionics.ErrBadFrequency
	}
	f.radio = append(f.radio, n, standby, mhz)
	return nil
}
func (f *fakeAircraft) SwapCOM(_ context.Context, n int) error {
	f.radio = append(f.radio, "swap", n)
	return nil
}
func (f *fakeAircraft) SetSquawk(_ context.Context, code string) error {
	f.radio = append(f.radio, code)
	return nil
}
func (f *fakeAircraft) SetCallsign(_ context.Context, airline, number string) error {
	f.airline, f.number = airline, number
	return nil
}
func (f *fakeAircraft) Addons(bool) (addons.Install, []addons.Package, error) {
	return addons.Install{Sim: "2024", Store: "steam", Packages: `C:\MSFS\Packages`}, f.pkgs, nil
}

func TestLiveAircraftTools(t *testing.T) {
	fa := &fakeAircraft{
		sys: live.AircraftSystems{
			Title: "FenixA319 CSA", ATCType: "A319",
			Package: &addons.Package{Folder: "fnx-aircraft-319-321", Source: addons.Community},
			Profile: systems.Profile{Name: "Fenix A320 family"},
			State: systems.State{Battery: true, Powered: true, Volts: 28.04, Engines: 2, Squawk: "2000", XPDRState: 1,
				DoorNames: []string{"L1", "L2"}, DoorsOpen: []bool{true, false}, COM1: true, COM1Active: 118.1,
				Values: map[string]float64{systems.Seatbelts: 1, systems.Door(1): 0}},
			Can: map[string]bool{systems.Door(0): true, systems.Door(1): true, systems.Chocks: false},
		},
		set: map[string]float64{},
		pkgs: []addons.Package{
			{Source: addons.Community, Folder: "fsdreamteam-gsx-pro", Title: "GSX Pro", Creator: "FSDreamTeam"},
			{Source: addons.Streamed, Folder: "fs20-orbx-airport-lkpr-prague", ICAO: "LKPR", Publisher: "orbx"},
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterLiveAircraftTools(mcp, fa)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	call := func(tool string, args map[string]any) (map[string]any, bool) {
		t.Helper()
		resp := callToolEvent(t, srv.URL, tool, args)
		isErr := false
		if r, ok := resp["result"].(map[string]any); ok {
			isErr, _ = r["isError"].(bool)
		}
		var got map[string]any
		if json.Unmarshal([]byte(contentTextEvent(t, resp)), &got) != nil {
			got = map[string]any{"text": contentTextEvent(t, resp)}
		}
		return got, isErr
	}

	got, isErr := call("get_aircraft_systems", nil)
	doors, _ := got["doors"].([]any)
	if isErr || len(doors) != 2 || got["seatbelts"] != true {
		t.Fatalf("systems: %v", got)
	}
	if ac, _ := got["aircraft"].(map[string]any); ac["package"] != "fnx-aircraft-319-321" {
		t.Errorf("package: %v", got["aircraft"])
	}
	if cannot, _ := got["cannot"].([]any); len(cannot) != 1 || cannot[0] != systems.Chocks {
		t.Errorf("cannot: %v", got["cannot"])
	}

	// A door by its name, closed.
	if got, isErr := call("set_aircraft_control", map[string]any{"control": "l2", "state": "closed"}); isErr || got["control"] != "door1" {
		t.Fatalf("door: %v", got)
	}
	if v, ok := fa.set["door1"]; !ok || v != 0 {
		t.Errorf("door1 set %v", fa.set)
	}
	if got, isErr := call("set_aircraft_control", map[string]any{"control": "chocks"}); !isErr {
		t.Errorf("chocks the profile can't operate: %v", got)
	}
	if got, isErr := call("set_aircraft_control", map[string]any{"control": "R9"}); !isErr {
		t.Errorf("unknown door: %v", got)
	}
	if _, isErr := call("set_aircraft_control", map[string]any{"control": "cabin_call"}); isErr || len(fa.pressed) != 1 {
		t.Errorf("cabin call: %v", fa.pressed)
	}

	if _, isErr := call("request_ground_service", map[string]any{"service": "fuel_truck"}); isErr || len(fa.services) != 1 || fa.services[0] != systems.FuelTruck {
		t.Errorf("service: %v", fa.services)
	}
	if _, isErr := call("request_ground_service", map[string]any{"service": "deicing"}); !isErr {
		t.Error("unknown service accepted")
	}

	if _, isErr := call("set_radio", map[string]any{"action": "com_standby", "com": 1, "frequency_mhz": 134.56}); isErr {
		t.Errorf("standby: %v", fa.radio)
	}
	if got, isErr := call("set_radio", map[string]any{"action": "com_active", "frequency_mhz": 99.5}); !isErr {
		t.Errorf("bad frequency accepted: %v", got)
	}
	if _, isErr := call("set_radio", map[string]any{"action": "squawk", "squawk": "4521"}); isErr || fa.radio[len(fa.radio)-1] != "4521" {
		t.Errorf("squawk: %v", fa.radio)
	}

	if _, isErr := call("set_atc_callsign", map[string]any{"airline": "Czech Air Force", "flight_number": "007"}); isErr || fa.airline != "Czech Air Force" || fa.number != "007" {
		t.Errorf("callsign: %q %q", fa.airline, fa.number)
	}
	if _, isErr := call("set_atc_callsign", nil); !isErr {
		t.Error("empty call sign accepted")
	}

	got, isErr = call("list_addons", map[string]any{"source": "all", "search": "lkpr"})
	pkgs, _ := got["packages"].([]any)
	if isErr || len(pkgs) != 1 {
		t.Fatalf("addons: %v", got)
	}
	if p, _ := pkgs[0].(map[string]any); p["icao"] != "LKPR" {
		t.Errorf("streamed airport: %v", p)
	}
	got, _ = call("list_addons", nil)
	if pkgs, _ := got["packages"].([]any); len(pkgs) != 1 {
		t.Errorf("default Community only: %v", got)
	}
}

func (f *fakeAircraft) GSXState(context.Context) (gsx.State, error) { return f.gsxState, nil }
func (f *fakeAircraft) SetLVar(_ context.Context, name string, v float64) error {
	f.lvars[name] = v
	return nil
}

func TestGSXAndLVarTools(t *testing.T) {
	fa := &fakeAircraft{lvars: map[string]float64{}, gsxState: gsx.State{Running: true, Boarding: gsx.Performing,
		Passengers: 150, PassengersBoardingTotal: 42, Gate: "C19", WaitingFor: []string{"exit 1"}}}
	fa.sys.State.V1Kt, fa.sys.State.VRKt, fa.sys.State.V2Kt, fa.sys.State.SpeedsFrom = 114, 115, 121, "table"
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterLiveAircraftTools(mcp, fa)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	call := func(tool string, args map[string]any) (map[string]any, bool) {
		t.Helper()
		resp := callToolEvent(t, srv.URL, tool, args)
		isErr := false
		if r, ok := resp["result"].(map[string]any); ok {
			isErr, _ = r["isError"].(bool)
		}
		var got map[string]any
		if json.Unmarshal([]byte(contentTextEvent(t, resp)), &got) != nil {
			got = map[string]any{"text": contentTextEvent(t, resp)}
		}
		return got, isErr
	}

	got, isErr := call("get_gsx_state", nil)
	svc, _ := got["services"].(map[string]any)
	if isErr || got["running"] != true || got["gate"] != "C19" || svc["boarding"] != gsx.Performing.String() {
		t.Fatalf("gsx: %v", got)
	}

	if got, isErr := call("set_lvar", map[string]any{"name": "MY_FLAG", "value": 2.0}); isErr || got["name"] != "L:MY_FLAG" || fa.lvars["MY_FLAG"] != 2 {
		t.Errorf("lvar: %v %v", got, fa.lvars)
	}
	if _, isErr := call("set_lvar", map[string]any{"name": "bad name!", "value": 1.0}); !isErr {
		t.Error("bad L:var name accepted")
	}

	got, _ = call("get_aircraft_systems", nil)
	if ts, _ := got["takeoff_speeds"].(map[string]any); ts["vr_kt"] != float64(115) || ts["source"] != "table" {
		t.Errorf("takeoff speeds: %v", got["takeoff_speeds"])
	}
}
