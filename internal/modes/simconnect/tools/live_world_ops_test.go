//go:build windows

package tools

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/traffic"
	"github.com/mrlm-net/simconnect/pkg/traffic/world"
)

func TestWorldOperationTools(t *testing.T) {
	fw := &fakeWorld{running: true, bodies: map[string]any{}, answers: map[string]any{
		"GET /api/control":               []world.ControlView{{ID: 3, Tail: "CSA7", ICAO: "LKPR"}},
		"POST /api/emergency":            map[string]any{},
		"POST /api/runway-use":           map[string]any{"use": "06"},
		"GET /api/stations/staffed":      []world.StaffedStation{{ICAO: "LKPR", Position: "tower"}},
		"PUT /api/stations/staffed":      []world.StaffedStation{{ICAO: "LKPR", Position: "ground"}},
		"GET /api/control/3/taxi/phrase": world.ClearancePhrase{Controller: traffic.Transmission{Text: "CSA7, taxi to holding point runway 24 via B"}},
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	registerWorldOperations(mcp, fw)
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

	if got, isErr := call("declare_emergency", map[string]any{"callsign": "csa7", "kind": "medical"}); isErr || got["emergency"] != true {
		t.Fatalf("declare_emergency: %v", got)
	}
	if b, _ := fw.bodies["POST /api/emergency"].(map[string]any); b["tail"] != "CSA7" || b["kind"] != "medical" || b["on"] != true {
		t.Errorf("emergency body: %v", b)
	}
	if _, isErr := call("declare_emergency", map[string]any{}); !isErr {
		t.Error("declare_emergency without a call sign should fail")
	}

	if got, isErr := call("set_runway_in_use", map[string]any{"icao": "lkpr", "runway": "06"}); isErr || got["use"] != "06" {
		t.Fatalf("set_runway_in_use: %v", got)
	}

	if got, isErr := call("get_staffed_stations", nil); isErr || len(got["staffed"].([]any)) != 1 {
		t.Fatalf("get_staffed_stations: %v", got)
	}
	if _, isErr := call("set_staffed_stations", map[string]any{"stations": []any{map[string]any{"icao": "LKPR", "position": "ground"}}}); isErr {
		t.Fatal("set_staffed_stations failed")
	}
	if l, _ := fw.bodies["PUT /api/stations/staffed"].([]world.StaffedStation); len(l) != 1 || l[0].Position != "ground" {
		t.Errorf("staffed body: %v", fw.bodies["PUT /api/stations/staffed"])
	}
	if _, isErr := call("set_staffed_stations", map[string]any{"stations": []any{map[string]any{"icao": "LKPR"}}}); !isErr {
		t.Error("a station with no position or name should fail")
	}

	got, isErr := call("preview_clearance", map[string]any{"callsign": "CSA7", "action": "taxi"})
	if p, _ := got["phrase"].(map[string]any); isErr || p == nil {
		t.Fatalf("preview_clearance: %v", got)
	}
	if fw.called("POST /api/control") {
		t.Error("preview_clearance must not give the clearance")
	}
}
