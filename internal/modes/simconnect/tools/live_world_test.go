//go:build windows

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/traffic"
	"github.com/mrlm-net/simconnect/pkg/traffic/world"
)

// fakeWorld is a live.World answering from canned responses and
// recording the calls.
type fakeWorld struct {
	running  bool
	answers  map[string]any // "METHOD path" (path without query) → answer
	calls    []string
	bodies   map[string]any
	schedule *world.ScheduleSettings
	corridor *world.CorridorSettings
}

func (f *fakeWorld) Ensure(context.Context) error { f.running = true; return nil }
func (f *fakeWorld) Running() bool                { return f.running }
func (f *fakeWorld) Error() string                { return "" }
func (f *fakeWorld) Snapshot() world.Snapshot     { return world.Snapshot{Connected: f.running} }

func (f *fakeWorld) Do(method, path string, body any) ([]byte, error) {
	f.calls = append(f.calls, method+" "+path)
	f.bodies[method+" "+path] = body
	key := method + " " + strings.SplitN(path, "?", 2)[0]
	a, ok := f.answers[key]
	if !ok {
		return nil, nil
	}
	if err, ok := a.(error); ok {
		return nil, err
	}
	return json.Marshal(a)
}

func (f *fakeWorld) Get(path string, v any) error {
	b, err := f.Do("GET", path, nil)
	if err != nil || len(b) == 0 {
		return err
	}
	return json.Unmarshal(b, v)
}

func (f *fakeWorld) SetSchedule(s world.ScheduleSettings) error { f.schedule = &s; return nil }
func (f *fakeWorld) SetCorridor(c world.CorridorSettings) error { f.corridor = &c; return nil }
func (f *fakeWorld) SetRealTraffic(bool, string) error          { return nil }

func (f *fakeWorld) called(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func TestWorldTrafficTools(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	fw := &fakeWorld{bodies: map[string]any{}, answers: map[string]any{
		"GET /api/models":  []string{"FSLTL A320 Air France SL", "FSLTL B738 Ryanair"},
		"GET /api/stands":  []map[string]any{{"index": 0, "label": "A1"}, {"index": 7, "label": "C 22"}},
		"GET /api/control": []world.ControlView{{ID: 3, Tail: "CSA7", ICAO: "LKPR", Kind: "departure", Actions: []string{"pushback"}}},
		"POST /api/control": world.ControlView{ID: 4, Tail: "TVS2", ICAO: "LKPR", Kind: "departure",
			Position: airport.LatLon{Lat: 50.1, Lon: 14.26}},
		"GET /api/schedule":                   map[string]any{"enabled": true, "airports": []string{"LKPR"}, "now": now, "flights": []any{1, 2}},
		"GET /api/boards":                     map[string]any{"departures": []any{}, "arrivals": []any{}},
		"GET /api/sequence":                   []map[string]any{{"runway": "24", "sequence": []any{}}},
		"POST /api/approach/LKPR/CSA7/holdat": fmt.Errorf("POST /api/approach: 409 not one of our arrivals"),
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterWorldTrafficTools(mcp, fw)
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

	// Not started: the read tools say so without starting the engine.
	if got, isErr := call("list_our_traffic", nil); isErr || got["engine"] != "not started" || fw.running {
		t.Fatalf("before start: %v", got)
	}

	// A departure on stand "C22" by type, held for clearances.
	got, isErr := call("spawn_departure", map[string]any{"icao": "lkpr", "callsign": "tvs2", "stand": "c22", "aircraft_type": "B738"})
	if isErr || got["callsign"] != "TVS2" || got["manual"] != true {
		t.Fatalf("spawn: %v", got)
	}
	req, _ := fw.bodies["POST /api/control"].(world.SpawnRequest)
	if req.Stand != 7 || req.Model != "FSLTL B738 Ryanair" || !req.Tug || !req.Procedure || req.Tail != "TVS2" {
		t.Errorf("spawn request %+v", req)
	}
	if !fw.called("GET /api/airport?icao=LKPR") || !fw.called("POST /api/control/4/manual?on=1") {
		t.Errorf("calls %v", fw.calls)
	}
	if _, isErr := call("spawn_departure", map[string]any{"icao": "LKPR", "callsign": "CSA7"}); !isErr {
		t.Error("a call sign already ours accepted")
	}

	if _, isErr := call("atc_clearance", map[string]any{"callsign": "csa7", "action": "pushback", "facing": "east"}); isErr ||
		!fw.called("POST /api/control/3/pushback?facing=east") {
		t.Errorf("clearance: %v", fw.calls)
	}
	if _, isErr := call("atc_clearance", map[string]any{"callsign": "CSA7", "action": "follow", "behind": "tvs2"}); isErr ||
		!fw.called("POST /api/control/3/follow?tail=TVS2") {
		t.Errorf("follow: %v", fw.calls)
	}
	if _, isErr := call("atc_clearance", map[string]any{"callsign": "CSA7", "action": "follow"}); !isErr {
		t.Error("follow without behind accepted")
	}
	if got, isErr := call("atc_clearance", map[string]any{"callsign": "XYZ9", "action": "taxi"}); !isErr {
		t.Errorf("unknown call sign: %v", got)
	}

	if got, isErr := call("approach_instruction", map[string]any{"callsign": "CSA7", "instruction": "holdat", "lat": 50.0, "lon": 14.0}); !isErr ||
		!strings.HasPrefix(got["text"].(string), "NOT_APPLICABLE") {
		t.Errorf("409: %v", got)
	}
	if _, isErr := call("approach_instruction", map[string]any{"callsign": "CSA7", "instruction": "joinfinal"}); !isErr {
		t.Error("joinfinal without a point accepted")
	}

	got, _ = call("get_landing_sequence", nil)
	if ap, _ := got["airports"].([]any); len(ap) != 1 {
		t.Errorf("sequence airports: %v", got)
	}

	got, isErr = call("start_schedule", map[string]any{"airports": "lkpr, lktb", "density": 0.5, "vfr": false})
	if isErr || fw.schedule == nil || fw.schedule.ICAO != "LKPR" || len(fw.schedule.Airports) != 2 || fw.schedule.VFR == nil || *fw.schedule.VFR {
		t.Fatalf("schedule %+v %v", fw.schedule, got)
	}
	if got["flight_count"] != float64(2) || got["flights"] != nil {
		t.Errorf("schedule status: %v", got)
	}

	if _, isErr := call("add_flights", map[string]any{"flights": []any{map[string]any{"callsign": "csa9", "origin": "eddm",
		"destination": "lkpr", "sta_in_min": 30.0}}}); isErr {
		t.Fatal("add_flights")
	}
	if fl, _ := fw.bodies["POST /api/flights"].([]traffic.Flight); len(fl) != 1 || fl[0].Callsign != "CSA9" || !fl[0].STA.Equal(now.Add(30*time.Minute)) {
		t.Errorf("flights %+v", fl)
	}

	if _, isErr := call("set_traffic_corridor", map[string]any{"route": "50,14; 51,15", "level_ft": 36000.0, "crossing": 0.0}); isErr ||
		fw.corridor == nil || len(fw.corridor.Route) != 2 || fw.corridor.Crossing == nil || *fw.corridor.Crossing != 0 {
		t.Errorf("corridor %+v", fw.corridor)
	}
	if _, isErr := call("set_traffic_corridor", map[string]any{"route": "50,14", "level_ft": 36000.0}); !isErr {
		t.Error("one-point route accepted")
	}

	if _, isErr := call("set_player_clearance", map[string]any{"icao": "LKPR", "runway": "24", "phase": "landing"}); isErr {
		t.Error("player clearance")
	}
	if _, isErr := call("set_player_clearance", map[string]any{"icao": "LKPR", "runway": "24", "phase": "flying"}); !isErr {
		t.Error("bad phase accepted")
	}

	fw.answers["GET /api/tcas"] = map[string]any{"ta": 1, "ra": 0, "events": []map[string]any{
		{"callsign": "CSA7", "advisory": "TA"}, {"callsign": "CSA7", "advisory": "clear"}}}
	if got, isErr := call("get_tcas", map[string]any{"limit": 1.0}); isErr || got["ta"] != float64(1) || len(got["events"].([]any)) != 1 {
		t.Errorf("tcas: %v", got)
	}

	if got, isErr := call("stop_schedule", map[string]any{"remove": true}); isErr || got["removed"] != float64(1) {
		t.Errorf("stop: %v", got)
	}
}
