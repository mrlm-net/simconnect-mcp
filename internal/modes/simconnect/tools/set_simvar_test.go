//go:build windows

package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"net/http/httptest"
)

// ── test helpers ─────────────────────────────────────────────────────────────

func newSetSimVarServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterSetSimVarTool(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func callToolSetSimVar(t *testing.T, serverURL string, args map[string]any) map[string]any {
	t.Helper()
	return callToolEvent(t, serverURL, "set_simvar_value", args)
}

func contentTextSetSimVar(t *testing.T, resp map[string]any) string {
	t.Helper()
	return contentTextEvent(t, resp)
}

func isErrorSetSimVar(t *testing.T, resp map[string]any) bool {
	t.Helper()
	return isErrorEvent(t, resp)
}

// ── set_simvar_value tests ────────────────────────────────────────────────────

// TestSetSimVarValue_Success verifies that a connected bridge returns {success:true}
// and records the call on MockBridge.
func TestSetSimVarValue_Success(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"name":  "AUTOPILOT ALTITUDE LOCK VAR",
		"unit":  "feet",
		"value": float64(10000),
	})

	if isErrorSetSimVar(t, resp) {
		t.Fatalf("expected success but got error: %s", contentTextSetSimVar(t, resp))
	}

	text := contentTextSetSimVar(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["success"] != true {
		t.Errorf("success: got %v, want true", got["success"])
	}
	if got["name"] != "AUTOPILOT ALTITUDE LOCK VAR" {
		t.Errorf("name: got %v, want 'AUTOPILOT ALTITUDE LOCK VAR'", got["name"])
	}
	if got["unit"] != "feet" {
		t.Errorf("unit: got %v, want 'feet'", got["unit"])
	}
	if got["value"] != float64(10000) {
		t.Errorf("value: got %v, want 10000", got["value"])
	}

	// Verify the call was recorded on the mock.
	if len(mb.SetSimVarCalls) != 1 {
		t.Fatalf("expected 1 SetSimVar call, got %d", len(mb.SetSimVarCalls))
	}
	call := mb.SetSimVarCalls[0]
	if call.Name != "AUTOPILOT ALTITUDE LOCK VAR" {
		t.Errorf("recorded name: got %q, want 'AUTOPILOT ALTITUDE LOCK VAR'", call.Name)
	}
	if call.Unit != "feet" {
		t.Errorf("recorded unit: got %q, want 'feet'", call.Unit)
	}
	if call.Value != 10000 {
		t.Errorf("recorded value: got %v, want 10000", call.Value)
	}
}

// TestSetSimVarValue_Disconnected verifies that a disconnected bridge returns BRIDGE_DISCONNECTED.
func TestSetSimVarValue_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"name":  "AUTOPILOT ALTITUDE LOCK VAR",
		"unit":  "feet",
		"value": float64(5000),
	})

	if !isErrorSetSimVar(t, resp) {
		t.Fatal("expected isError=true for disconnected bridge")
	}
	text := contentTextSetSimVar(t, resp)
	if !strings.Contains(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in content, got: %s", text)
	}
}

// TestSetSimVarValue_MissingName verifies that an absent 'name' returns INVALID_ARGUMENT.
func TestSetSimVarValue_MissingName(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"unit":  "feet",
		"value": float64(5000),
	})

	if !isErrorSetSimVar(t, resp) {
		t.Fatal("expected isError=true for missing name")
	}
	text := contentTextSetSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestSetSimVarValue_MissingUnit verifies that an absent 'unit' returns INVALID_ARGUMENT.
func TestSetSimVarValue_MissingUnit(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"name":  "AUTOPILOT ALTITUDE LOCK VAR",
		"value": float64(5000),
	})

	if !isErrorSetSimVar(t, resp) {
		t.Fatal("expected isError=true for missing unit")
	}
	text := contentTextSetSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestSetSimVarValue_MissingValue verifies that an absent 'value' returns INVALID_ARGUMENT.
func TestSetSimVarValue_MissingValue(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"name": "AUTOPILOT ALTITUDE LOCK VAR",
		"unit": "feet",
	})

	if !isErrorSetSimVar(t, resp) {
		t.Fatal("expected isError=true for missing value")
	}
	text := contentTextSetSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestSetSimVarValue_BridgeError verifies that a bridge error returns INTERNAL_ERROR.
func TestSetSimVarValue_BridgeError(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:         bridge.StateConnected,
		MockSetSimVarError: errors.New("write failed"),
	}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"name":  "AUTOPILOT ALTITUDE LOCK VAR",
		"unit":  "feet",
		"value": float64(5000),
	})

	if !isErrorSetSimVar(t, resp) {
		t.Fatal("expected isError=true for bridge error")
	}
	text := contentTextSetSimVar(t, resp)
	if !strings.Contains(text, "INTERNAL_ERROR") {
		t.Errorf("expected INTERNAL_ERROR in content, got: %s", text)
	}
}

// TestSetSimVarValue_NegativeValue verifies that negative values are accepted
// (SimConnect supports negative SimVars, e.g. bank angle, vertical speed).
func TestSetSimVarValue_NegativeValue(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSetSimVarServer(t, mb)

	resp := callToolSetSimVar(t, srv.URL, map[string]any{
		"name":  "PLANE BANK DEGREES",
		"unit":  "degrees",
		"value": float64(-15),
	})

	if isErrorSetSimVar(t, resp) {
		t.Fatalf("expected success for negative value but got error: %s", contentTextSetSimVar(t, resp))
	}

	if len(mb.SetSimVarCalls) != 1 || mb.SetSimVarCalls[0].Value != -15 {
		t.Errorf("expected value -15 to be recorded; calls: %v", mb.SetSimVarCalls)
	}
}
