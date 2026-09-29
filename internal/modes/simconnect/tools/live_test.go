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
