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

func newEnrichedTrafficServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterEnrichedTrafficTool(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func callEnrichedTool(t *testing.T, serverURL string, args map[string]any) map[string]any {
	t.Helper()
	return callToolEvent(t, serverURL, "get_traffic_with_phase", args)
}

func TestGetTrafficWithPhase_NoTraffic(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:                  bridge.StateConnected,
		MockEnrichedTrafficResults: []bridge.EnrichedTrafficEntry{},
	}
	srv := newEnrichedTrafficServer(t, mb)

	resp := callEnrichedTool(t, srv.URL, map[string]any{})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	if got["count"].(float64) != 0 {
		t.Errorf("expected count=0, got %v", got["count"])
	}
	if got["radius_meters"].(float64) != 25000 {
		t.Errorf("expected default radius=25000, got %v", got["radius_meters"])
	}
}

func TestGetTrafficWithPhase_WithEnrichedData(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockEnrichedTrafficResults: []bridge.EnrichedTrafficEntry{
			{
				ObjectID:         1,
				Title:            "FenixA319",
				ATCID:            "NJU100",
				ATCAirline:       "easyJet",
				Category:         "Airplane",
				Latitude:         32.697,
				Longitude:        -16.778,
				AltitudeFt:       3500,
				TrueHeading:      270,
				TrackDeg:         268.5,
				GroundSpeed:      180,
				VerticalSpeedFPM: -800,
				OnGround:         false,
				InParkingState:   false,
				OnAnyRunway:      false,
				FlightPhase:      "APPROACH",
			},
			{
				ObjectID:       2,
				Title:          "Airbus A320",
				ATCID:          "TAP123",
				FlightPhase:    "PARKED",
				OnGround:       true,
				InParkingState: true,
				GroundSpeed:    0,
			},
		},
	}
	srv := newEnrichedTrafficServer(t, mb)

	resp := callEnrichedTool(t, srv.URL, map[string]any{"radius_meters": float64(50000)})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	if got["count"].(float64) != 2 {
		t.Errorf("expected count=2, got %v", got["count"])
	}

	traffic := got["traffic"].([]any)
	if len(traffic) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(traffic))
	}

	first := traffic[0].(map[string]any)
	if first["flight_phase"] != "APPROACH" {
		t.Errorf("flight_phase: got %v, want APPROACH", first["flight_phase"])
	}
	if first["track_deg"].(float64) != 268.5 {
		t.Errorf("track_deg: got %v, want 268.5", first["track_deg"])
	}
	if first["vertical_speed_fpm"].(float64) != -800 {
		t.Errorf("vertical_speed_fpm: got %v, want -800", first["vertical_speed_fpm"])
	}
	if first["category"] != "Airplane" {
		t.Errorf("category: got %v, want Airplane", first["category"])
	}

	second := traffic[1].(map[string]any)
	if second["flight_phase"] != "PARKED" {
		t.Errorf("second flight_phase: got %v, want PARKED", second["flight_phase"])
	}
	if second["in_parking_state"] != true {
		t.Errorf("in_parking_state: got %v, want true", second["in_parking_state"])
	}
}

func TestGetTrafficWithPhase_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newEnrichedTrafficServer(t, mb)

	resp := callEnrichedTool(t, srv.URL, map[string]any{})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key when disconnected")
	}
}

func TestGetTrafficWithPhase_SimTime(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:                  bridge.StateConnected,
		MockEnrichedTrafficResults: []bridge.EnrichedTrafficEntry{},
		MockSimTime:                7200.0,
	}
	srv := newEnrichedTrafficServer(t, mb)

	resp := callEnrichedTool(t, srv.URL, map[string]any{})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}
	if got["sim_time"] != float64(7200.0) {
		t.Errorf("expected sim_time=7200.0, got %v", got["sim_time"])
	}
}

func TestGetTrafficWithPhase_RadiusCapped(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:                  bridge.StateConnected,
		MockEnrichedTrafficResults: []bridge.EnrichedTrafficEntry{},
	}
	srv := newEnrichedTrafficServer(t, mb)

	resp := callEnrichedTool(t, srv.URL, map[string]any{"radius_meters": float64(999999)})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	if got["radius_meters"].(float64) != 200000 {
		t.Errorf("expected capped radius=200000, got %v", got["radius_meters"])
	}
}
