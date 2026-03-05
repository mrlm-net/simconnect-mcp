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

// ── test helpers ──────────────────────────────────────────────────────────────

func newTrafficServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterTrafficTool(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func callToolTraffic(t *testing.T, serverURL string, args map[string]any) map[string]any {
	t.Helper()
	return callToolEvent(t, serverURL, "get_nearby_traffic", args)
}

func contentTextTraffic(t *testing.T, resp map[string]any) string {
	t.Helper()
	return contentTextEvent(t, resp)
}

// ── get_nearby_traffic tests ──────────────────────────────────────────────────

func TestGetNearbyTraffic_NoTraffic(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:          bridge.StateConnected,
		MockTrafficResults: []bridge.TrafficEntry{},
	}
	srv := newTrafficServer(t, mb)

	resp := callToolTraffic(t, srv.URL, map[string]any{})
	text := contentTextTraffic(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["count"].(float64) != 0 {
		t.Errorf("expected count=0, got %v", got["count"])
	}
	if got["radius_meters"].(float64) != 25000 {
		t.Errorf("expected default radius=25000, got %v", got["radius_meters"])
	}
}

func TestGetNearbyTraffic_WithTraffic(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockTrafficResults: []bridge.TrafficEntry{
			{ObjectID: 1, Title: "Boeing 737", ATCID: "CSA100", ATCAirline: "Czech Airlines",
				Latitude: 50.1, Longitude: 14.3, AltitudeFt: 1200, OnGround: true},
			{ObjectID: 2, Title: "Airbus A320", ATCID: "EZY200", ATCAirline: "easyJet",
				Latitude: 50.2, Longitude: 14.4, AltitudeFt: 5000, OnGround: false},
		},
	}
	srv := newTrafficServer(t, mb)

	resp := callToolTraffic(t, srv.URL, map[string]any{"radius_meters": float64(50000)})
	text := contentTextTraffic(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["count"].(float64) != 2 {
		t.Errorf("expected count=2, got %v", got["count"])
	}
	if got["radius_meters"].(float64) != 50000 {
		t.Errorf("expected radius_meters=50000, got %v", got["radius_meters"])
	}
	traffic := got["traffic"].([]any)
	if len(traffic) != 2 {
		t.Fatalf("expected 2 traffic entries, got %d", len(traffic))
	}
	first := traffic[0].(map[string]any)
	if first["atc_id"] != "CSA100" {
		t.Errorf("atc_id: got %v, want CSA100", first["atc_id"])
	}
}

func TestGetNearbyTraffic_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newTrafficServer(t, mb)

	resp := callToolTraffic(t, srv.URL, map[string]any{})
	text := contentTextTraffic(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key in result when disconnected")
	}
}

func TestGetNearbyTraffic_RadiusCapped(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:          bridge.StateConnected,
		MockTrafficResults: []bridge.TrafficEntry{},
	}
	srv := newTrafficServer(t, mb)

	resp := callToolTraffic(t, srv.URL, map[string]any{"radius_meters": float64(999999)})
	text := contentTextTraffic(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["radius_meters"].(float64) != 200000 {
		t.Errorf("expected capped radius=200000, got %v", got["radius_meters"])
	}
}
