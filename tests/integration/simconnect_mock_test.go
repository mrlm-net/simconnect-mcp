//go:build windows

// Package integration — simconnect_mock_test.go
// End-to-end tests for the MCP simconnect mode using MockBridge.
// These tests start the full Gin router in-process via net/http/httptest and
// exercise all 4 registered MCP tools plus the health endpoint.
//
// No live simulator is required — all bridge interactions are satisfied by
// bridge.MockBridge. The //go:build windows tag is required because
// internal/modes/simconnect is gated with //go:build windows.
//
// Run with: go test -race -v -tags windows ./tests/integration/
package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/simconnect"
	"github.com/mrlm-net/simconnect-mcp/internal/server"
)

// bridgeMounter is a local interface used to access MountWithBridge on the
// concrete simconnect mode without relying on the unexported type.
// *simconnectMode satisfies this interface on Windows builds.
type bridgeMounter interface {
	MountWithBridge(r *gin.Engine, b bridge.Bridge) error
}

// newSimconnectMockServer builds a simconnect-mode Gin engine backed by the
// provided MockBridge. The returned *httptest.Server is closed automatically
// when t completes.
func newSimconnectMockServer(t *testing.T, mock *bridge.MockBridge) *httptest.Server {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := server.New()

	cfg := simconnect.ConfigFromEnv()
	mode := simconnect.New(cfg)

	bm, ok := mode.(bridgeMounter)
	if !ok {
		t.Fatal("simconnect.New() does not implement bridgeMounter; MountWithBridge unavailable")
	}

	if err := bm.MountWithBridge(r, mock); err != nil {
		t.Fatalf("simconnect mode MountWithBridge failed: %v", err)
	}

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// connectedMock returns a MockBridge configured for a connected simulator with
// sensible default values for SimVar and SimState queries.
//
// Note: MountWithBridge calls b.Open() which sets MockState = StateConnected,
// so the initial MockState value here is overwritten. The mock's state remains
// connected after mounting, which is the desired behaviour for these tests.
func connectedMock() *bridge.MockBridge {
	return &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockSimVar: bridge.SimVar{
			Value:   35000,
			SimTime: 1.0,
		},
		MockSimState: bridge.SimState{
			Connected:        true,
			Paused:           false,
			CurrentFlight:    "test.flt",
			SimulatorVersion: "MSFS 2024",
		},
	}
}

// newDisconnectedServer builds a simconnect-mode server and then sets the
// bridge state to StateDisconnected so that bridge-disconnected tests can run.
// MountWithBridge calls Open() which transitions the mock to StateConnected, so
// we reset the state after mounting to simulate a lost connection.
func newDisconnectedServer(t *testing.T) (*httptest.Server, *bridge.MockBridge) {
	t.Helper()

	mock := &bridge.MockBridge{
		MockSimState: bridge.SimState{Connected: false},
	}

	gin.SetMode(gin.TestMode)
	r := server.New()

	cfg := simconnect.ConfigFromEnv()
	mode := simconnect.New(cfg)

	bm, ok := mode.(bridgeMounter)
	if !ok {
		t.Fatal("simconnect.New() does not implement bridgeMounter; MountWithBridge unavailable")
	}

	if err := bm.MountWithBridge(r, mock); err != nil {
		t.Fatalf("simconnect mode MountWithBridge failed: %v", err)
	}

	// Open() in MountWithBridge sets MockState = StateConnected.
	// Force it back to StateDisconnected to simulate a lost connection.
	mock.MockState = bridge.StateDisconnected

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, mock
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestSimconnectToolsList verifies that tools/list returns exactly 4 tools.
func TestSimconnectToolsList(t *testing.T) {
	srv := newSimconnectMockServer(t, connectedMock())

	resp := listTools(t, srv.URL)
	result := toolResult(t, resp)

	toolsRaw, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list result missing 'tools' array: %v", result)
	}

	want := []string{
		"get_simvar_value",
		"get_simvar_values",
		"transmit_event",
		"get_sim_state",
	}

	if len(toolsRaw) != len(want) {
		t.Fatalf("expected %d tools, got %d: %v", len(want), len(toolsRaw), toolsRaw)
	}

	got := make(map[string]bool, len(toolsRaw))
	for _, v := range toolsRaw {
		tool, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("tool entry not an object: %v", v)
		}
		name, _ := tool["name"].(string)
		got[name] = true
	}

	for _, name := range want {
		if !got[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

// TestGetSimVarValue_Success verifies that get_simvar_value returns the
// correct JSON shape: name, value, unit, and sim_time all present.
func TestGetSimVarValue_Success(t *testing.T) {
	srv := newSimconnectMockServer(t, connectedMock())

	resp := callTool(t, srv.URL, "get_simvar_value", map[string]any{
		"name": "PLANE ALTITUDE",
		"unit": "feet",
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var sv struct {
		Name    string  `json:"name"`
		Value   float64 `json:"value"`
		Unit    string  `json:"unit"`
		SimTime float64 `json:"sim_time"`
	}
	parseContentJSON(t, resp, &sv)

	if sv.Name != "PLANE ALTITUDE" {
		t.Errorf("name: got %q, want %q", sv.Name, "PLANE ALTITUDE")
	}
	if sv.Value != 35000 {
		t.Errorf("value: got %v, want 35000", sv.Value)
	}
	if sv.Unit != "feet" {
		t.Errorf("unit: got %q, want %q", sv.Unit, "feet")
	}
	if sv.SimTime != 1.0 {
		t.Errorf("sim_time: got %v, want 1.0", sv.SimTime)
	}
}

// TestGetSimVarValue_Disconnected verifies that get_simvar_value returns an
// error containing "BRIDGE_DISCONNECTED" when the bridge is not connected.
func TestGetSimVarValue_Disconnected(t *testing.T) {
	srv, _ := newDisconnectedServer(t)

	resp := callTool(t, srv.URL, "get_simvar_value", map[string]any{
		"name": "PLANE ALTITUDE",
		"unit": "feet",
	})

	if !isToolError(t, resp) {
		t.Fatal("expected isError=true when bridge is disconnected")
	}

	text := contentText(t, resp)
	if !strings.Contains(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected content to contain BRIDGE_DISCONNECTED, got: %s", text)
	}
}

// TestGetSimVarValues_Success verifies that get_simvar_values returns an array
// with 2 items, each containing name, value, and unit fields.
func TestGetSimVarValues_Success(t *testing.T) {
	srv := newSimconnectMockServer(t, connectedMock())

	resp := callTool(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": []any{
			map[string]any{"name": "PLANE ALTITUDE", "unit": "feet"},
			map[string]any{"name": "AIRSPEED INDICATED", "unit": "knots"},
		},
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var items []struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	parseContentJSON(t, resp, &items)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Name != "PLANE ALTITUDE" {
		t.Errorf("items[0].name: got %q, want %q", items[0].Name, "PLANE ALTITUDE")
	}
	if items[0].Unit != "feet" {
		t.Errorf("items[0].unit: got %q, want %q", items[0].Unit, "feet")
	}
	if items[1].Name != "AIRSPEED INDICATED" {
		t.Errorf("items[1].name: got %q, want %q", items[1].Name, "AIRSPEED INDICATED")
	}
	if items[1].Unit != "knots" {
		t.Errorf("items[1].unit: got %q, want %q", items[1].Unit, "knots")
	}
}

// TestGetSimVarValues_TooManyVars verifies that submitting 21 variables returns
// a tool error containing "INVALID_ARGUMENT".
func TestGetSimVarValues_TooManyVars(t *testing.T) {
	srv := newSimconnectMockServer(t, connectedMock())

	// Build a slice of 21 variable requests.
	vars := make([]any, 21)
	for i := range vars {
		vars[i] = map[string]any{"name": "PLANE ALTITUDE", "unit": "feet"}
	}

	resp := callTool(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": vars,
	})

	if !isToolError(t, resp) {
		t.Fatal("expected isError=true for 21 vars (exceeds max of 20)")
	}

	text := contentText(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected content to contain INVALID_ARGUMENT, got: %s", text)
	}
}

// TestTransmitEvent_Success verifies that transmit_event returns
// {success: true, event: <name>} for a connected bridge.
func TestTransmitEvent_Success(t *testing.T) {
	mock := connectedMock()
	srv := newSimconnectMockServer(t, mock)

	resp := callTool(t, srv.URL, "transmit_event", map[string]any{
		"name": "LANDING_LIGHTS_TOGGLE",
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var result struct {
		Success bool   `json:"success"`
		Event   string `json:"event"`
	}
	parseContentJSON(t, resp, &result)

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Event != "LANDING_LIGHTS_TOGGLE" {
		t.Errorf("event: got %q, want %q", result.Event, "LANDING_LIGHTS_TOGGLE")
	}
}

// TestTransmitEvent_Disconnected verifies that transmit_event returns an error
// containing "BRIDGE_DISCONNECTED" when the bridge has no active connection.
func TestTransmitEvent_Disconnected(t *testing.T) {
	srv, _ := newDisconnectedServer(t)

	resp := callTool(t, srv.URL, "transmit_event", map[string]any{
		"name": "LANDING_LIGHTS_TOGGLE",
	})

	if !isToolError(t, resp) {
		t.Fatal("expected isError=true when bridge is disconnected")
	}

	text := contentText(t, resp)
	if !strings.Contains(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected content to contain BRIDGE_DISCONNECTED, got: %s", text)
	}
}

// TestGetSimState_Connected verifies that get_sim_state returns
// {connected: true, current_flight: "test.flt"} for a connected bridge.
func TestGetSimState_Connected(t *testing.T) {
	srv := newSimconnectMockServer(t, connectedMock())

	resp := callTool(t, srv.URL, "get_sim_state", map[string]any{})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var state struct {
		Connected     bool   `json:"connected"`
		CurrentFlight string `json:"current_flight"`
	}
	parseContentJSON(t, resp, &state)

	if !state.Connected {
		t.Error("expected connected=true")
	}
	if state.CurrentFlight != "test.flt" {
		t.Errorf("current_flight: got %q, want %q", state.CurrentFlight, "test.flt")
	}
}

// TestGetSimState_Disconnected verifies that get_sim_state returns
// {connected: false} as a SUCCESS result (isError must be false) when the
// bridge is not connected.
func TestGetSimState_Disconnected(t *testing.T) {
	srv, _ := newDisconnectedServer(t)

	resp := callTool(t, srv.URL, "get_sim_state", map[string]any{})

	// Must be a success result — get_sim_state never returns an error
	// when disconnected; it returns {connected: false} instead.
	if isToolError(t, resp) {
		t.Fatalf("expected isError=false for disconnected state, got error: %s", contentText(t, resp))
	}

	var state struct {
		Connected bool `json:"connected"`
	}
	parseContentJSON(t, resp, &state)

	if state.Connected {
		t.Error("expected connected=false for disconnected bridge")
	}
}

// TestSimconnectHealth verifies that GET /health returns 200 with
// mode="simconnect" and a sim_connected field.
func TestSimconnectHealth(t *testing.T) {
	// Use a disconnected server to verify the endpoint does not depend on a live simulator.
	srv, _ := newDisconnectedServer(t)

	resp, err := http.Get(srv.URL + "/health") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode /health body: %v", err)
	}

	mode, _ := body["mode"].(string)
	if mode != "simconnect" {
		t.Errorf("health: mode=%q, want %q", mode, "simconnect")
	}

	if _, ok := body["sim_connected"]; !ok {
		t.Error("health: missing sim_connected field")
	}
}
