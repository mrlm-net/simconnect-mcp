//go:build windows

package tools

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

func TestGetFuelState(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterFuelTool(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	call := func() (map[string]any, bool) {
		resp := callToolEvent(t, srv.URL, "get_fuel_state", nil)
		isErr := false
		if r, ok := resp["result"].(map[string]any); ok {
			isErr, _ = r["isError"].(bool)
		}
		var got map[string]any
		_ = json.Unmarshal([]byte(contentTextEvent(t, resp)), &got)
		return got, isErr
	}

	// An A320: no center fuel in this load, both mains half full.
	vals := []float64{3200, 6400, 21440, 0, 2200, 1600, 1800, 1600, 1800}
	res := make([]bridge.SimVarResult, len(fuelVars))
	for i, v := range fuelVars {
		res[i] = bridge.SimVarResult{Name: v.Name, Unit: v.Unit, Value: vals[i]}
	}
	mb.MockSimVarResults = res
	got, isErr := call()
	if isErr || got["total_percent_full"] != 50.0 || got["total_weight_kg"] != 9725.0 {
		t.Fatalf("fuel: %v", got)
	}
	tanks, _ := got["tanks"].([]any)
	if len(tanks) != 3 || tanks[0].(map[string]any)["percent_full"] != 0.0 || tanks[1].(map[string]any)["percent_full"] != 88.9 {
		t.Errorf("tanks: %v", tanks)
	}

	// A single-engine piston: no center tank fitted.
	res[4].Value = 0
	mb.MockSimVarResults = res
	got, _ = call()
	if tanks, _ := got["tanks"].([]any); len(tanks) != 2 {
		t.Errorf("no center tank fitted: %v", got["tanks"])
	}

	mb.MockState = bridge.StateDisconnected
	if _, isErr = call(); !isErr {
		t.Error("disconnected: no error")
	}
}
