//go:build windows

package tools

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

func newLiveServer(t *testing.T) (*httptest.Server, *live.Fixture) {
	t.Helper()
	fx, err := live.NewFixture("../../../live/testdata")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterLiveAirportTools(mcp, fx)
	RegisterLiveNavTools(mcp, fx)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, fx
}

func TestLiveTools(t *testing.T) {
	srv, fx := newLiveServer(t)
	cases := []struct {
		tool  string
		args  map[string]any
		isErr bool
		check func(t *testing.T, got map[string]any)
	}{
		{"get_airport_procedures", map[string]any{"icao": "lkpr", "runway": "24"}, false, func(t *testing.T, got map[string]any) {
			if n := len(got["sids"].([]any)); n != 9 {
				t.Errorf("SIDs for 24: %d, want 9", n)
			}
			if n := len(got["approaches"].([]any)); n != 2 {
				t.Errorf("approaches for 24: %d, want 2", n)
			}
		}},
		{"get_airport_procedures", map[string]any{"icao": "LKPR", "name": "ILS 24"}, false, func(t *testing.T, got map[string]any) {
			pts := got["points"].([]any)
			last := pts[len(pts)-1].(map[string]any)
			if got["kind"] != "APPROACH" || last["ident"] != "RW24" || last["map"] != true || got["missed_approach"] == nil {
				t.Errorf("ILS 24: %v", got)
			}
		}},
		{"get_airport_procedures", map[string]any{"icao": "LKPR", "name": "OMNI"}, true, nil}, // several runways: needs runway
		{"get_airport_procedures", map[string]any{"icao": "LKPR", "name": "VOZ5M"}, false, func(t *testing.T, got map[string]any) {
			if got["kind"] != "SID" || len(got["points"].([]any)) < 3 {
				t.Errorf("VOZ5M: %v", got)
			}
		}},
		{"get_airport_procedures", map[string]any{"icao": "LKXX"}, true, nil},
		{"plan_taxi_route", map[string]any{"icao": "LKPR", "parking": "C22", "runway": "24"}, false, func(t *testing.T, got map[string]any) {
			if got["instruction"] != "taxi to holding point runway 24 via H1 H A" || got["length_m"] != 1595.0 {
				t.Errorf("C22 → 24: %v", got)
			}
		}},
		{"plan_taxi_route", map[string]any{"icao": "LKPR", "parking": "C22", "runway": "24", "entry": "B"}, false, func(t *testing.T, got map[string]any) {
			if got["entry"] != "B" {
				t.Errorf("C22 → 24 at B: %v", got)
			}
		}},
		{"plan_taxi_route", map[string]any{"icao": "LKPR", "parking": "C22", "runway": "24", "direction": "arrival", "exit": "D"}, false, func(t *testing.T, got map[string]any) {
			if got["exit"] != "D" || len(got["taxiways"].([]any)) == 0 {
				t.Errorf("24 via D → C22: %v", got)
			}
		}},
		{"plan_taxi_route", map[string]any{"icao": "LKPR", "parking": "C22", "runway": "24", "direction": "arrival", "exit": "Q"}, true, nil},
		{"plan_taxi_route", map[string]any{"icao": "LKPR", "parking": "ZZ99", "runway": "24"}, true, nil},
		{"get_runway_entries_exits", map[string]any{"icao": "LKPR", "runway": "24"}, false, func(t *testing.T, got map[string]any) {
			entries := got["entries"].([]any)
			if len(entries) != 3 || entries[0].(map[string]any)["taxiway"] != "A" {
				t.Errorf("entries onto 24: %v", entries)
			}
		}},
		{"find_stands", map[string]any{"icao": "LKPR", "wingspan_m": 64.8}, false, func(t *testing.T, got map[string]any) {
			if got["total"] != 4.0 {
				t.Errorf("stands for a 777: %v", got["total"])
			}
		}},
		{"get_weather", nil, false, func(t *testing.T, got map[string]any) {
			if got["wind_dir_true"] != 240.0 || got["qnh_inhg"] != 29.91 {
				t.Errorf("weather: %v", got)
			}
		}},
		{"get_active_runway", map[string]any{"icao": "LKPR"}, false, func(t *testing.T, got map[string]any) {
			if got["departure_runway"] != "24" || got["approach"] != "ILS 24" || got["transition_level"] != 70.0 {
				t.Errorf("runway in use: %v", got)
			}
		}},
		{"get_atis", map[string]any{"icao": "LKPR", "letter": "k"}, false, func(t *testing.T, got map[string]any) {
			if !strings.Contains(got["text"].(string), "information Kilo") || !strings.Contains(got["spoken"].(string), "two four") {
				t.Errorf("ATIS: %v", got)
			}
		}},
		{"get_atis", map[string]any{"icao": "LKPR", "letter": "KK"}, true, nil},
		{"get_fix", map[string]any{"ident": "VOZ", "region": "LK"}, false, func(t *testing.T, got map[string]any) {
			f := got["fixes"].([]any)[0].(map[string]any)
			if f["kind"] != "VOR" || f["freq"] != 116.95 || f["name"] != "VOZICE" {
				t.Errorf("VOZ: %v", f)
			}
		}},
		{"get_fix", map[string]any{"ident": "NOSUCH"}, true, nil},
		{"find_airway_route", map[string]any{"from": "VOZ.LK.V", "to": "OKF.LK.V"}, false, func(t *testing.T, got map[string]any) {
			if got["route"] != "VOZ M725 OKF" {
				t.Errorf("VOZ → OKF: %v", got["route"])
			}
		}},
		{"find_airway_route", map[string]any{"from": "VOZ.LK.X", "to": "OKF"}, true, nil},
		{"plan_flight", map[string]any{"departure": "LKPR", "arrival": "EDDM", "aircraft_type": "A20N", "load_into_sim": true}, false, func(t *testing.T, got map[string]any) {
			if !strings.HasPrefix(got["route"].(string), "DOBE4A DOBEN ") || got["departure_runway"] != "24" || got["loaded_into_sim"] != true {
				t.Errorf("LKPR → EDDM: %v", got)
			}
			if len(fx.Loaded) != 1 || !strings.Contains(string(fx.Loaded[0]), "<ATCWaypoint") {
				t.Errorf("no .pln loaded")
			}
		}},
		{"plan_flight", map[string]any{"departure": "LKPR", "arrival": "EDDM", "airways": false, "cruise_fl": 240.0, "include_pln": true}, false, func(t *testing.T, got map[string]any) {
			if got["cruise_fl"] != 240.0 || !strings.Contains(got["pln"].(string), "SimBase.Document") {
				t.Errorf("direct LKPR → EDDM FL240: %v", got["cruise_fl"])
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			resp := callToolEvent(t, srv.URL, c.tool, c.args)
			isErr := false
			if r, ok := resp["result"].(map[string]any); ok {
				isErr, _ = r["isError"].(bool)
			}
			text := contentTextEvent(t, resp)
			if isErr != c.isErr {
				t.Fatalf("%v: isError=%v, want %v: %s", c.args, isErr, c.isErr, text)
			}
			if c.check != nil {
				var got map[string]any
				if err := json.Unmarshal([]byte(text), &got); err != nil {
					t.Fatalf("%v: %v", err, text)
				}
				c.check(t, got)
			}
		})
	}
}

func TestLiveTrafficTools(t *testing.T) {
	fx, err := live.NewFixture("../../../live/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ft := &live.FixtureTraffic{ModelList: []string{"FSLTL A320 CSA Czech Airlines", "FSLTL B738 TVS Smartwings"}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterLiveTrafficTools(mcp, fx, ft)
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
		text := contentTextEvent(t, resp)
		var got map[string]any
		if json.Unmarshal([]byte(text), &got) != nil {
			got = map[string]any{"text": text}
		}
		return got, isErr
	}

	got, isErr := call("list_aircraft_models", map[string]any{"filter": "a320 csa"})
	if isErr || got["count"] != 1.0 {
		t.Errorf("models: %v", got)
	}

	// Runway in use for the fixture's wind (240/10) and the first SID for it.
	got, isErr = call("spawn_departure", map[string]any{"icao": "LKPR", "callsign": "csa123", "stand": "C22"})
	if isErr || got["runway"] != "24" || got["procedure"] != "ARTU5A" || got["callsign"] != "CSA123" {
		t.Fatalf("departure: %v", got)
	}
	d := ft.Departures[0]
	if !d.HoldForClearances || len(d.Departure) < 2 || d.Limits == nil || d.Limits.TransitionAltitudeFt != 5000 {
		t.Errorf("departure spec: hold %v, SID points %d, limits %+v", d.HoldForClearances, len(d.Departure), d.Limits)
	}
	if _, isErr = call("spawn_departure", map[string]any{"icao": "LKPR", "callsign": "CSA124", "sid": "NOPE"}); !isErr {
		t.Error("unknown SID accepted")
	}
	if _, isErr = call("spawn_departure", map[string]any{"icao": "LKPR", "callsign": "C", "stand": "C22"}); !isErr {
		t.Error("bad call sign accepted")
	}

	got, isErr = call("spawn_arrival", map[string]any{"icao": "LKPR", "callsign": "DLH4AB", "runway": "24", "hold_for_clearance": false})
	if isErr || !strings.Contains(got["procedure"].(string), "→ ILS 24") {
		t.Fatalf("arrival: %v", got)
	}
	a := ft.Arrivals[0]
	if a.HoldForClearance || len(a.Procedure) < 3 {
		t.Errorf("arrival spec: hold %v, procedure points %d", a.HoldForClearance, len(a.Procedure))
	}
	if last := a.Procedure[len(a.Procedure)-1]; last.Ident != "RW24" {
		t.Errorf("arrival ends at %q, want RW24", last.Ident)
	}
	if _, isErr = call("spawn_arrival", map[string]any{"icao": "LKPR", "callsign": "DLH4AC", "runway": "24", "star": "none"}); isErr {
		t.Error("straight-in arrival refused")
	}
	if ft.Arrivals[1].Procedure != nil {
		t.Error("star=none still flies a procedure")
	}

	got, _ = call("list_our_traffic", nil)
	if got["count"] != 3.0 {
		t.Errorf("our traffic: %v", got)
	}
	if got, isErr = call("atc_clearance", map[string]any{"callsign": "XXX1", "action": "taxi"}); !isErr || !strings.Contains(got["text"].(string), "NOT_FOUND") {
		t.Errorf("clearance for a stranger: %v", got)
	}

	got, isErr = call("generate_schedule", map[string]any{"airports": "LKPR", "hours": 3.0, "start": "2026-10-01T06:00:00Z"})
	if isErr || got["count"].(float64) < 5 {
		t.Errorf("schedule: %v", got)
	}
	got2, _ := call("generate_schedule", map[string]any{"airports": "LKPR", "hours": 3.0, "start": "2026-10-01T06:00:00Z"})
	b1, _ := json.Marshal(got)
	b2, _ := json.Marshal(got2)
	if string(b1) != string(b2) {
		t.Error("schedule not deterministic for a seed")
	}
}
