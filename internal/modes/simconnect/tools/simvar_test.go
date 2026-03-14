//go:build windows

package tools

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// newSimVarServer returns a test HTTP server with RegisterSimVarTools mounted
// using the provided MockBridge. The server is closed when t completes.
func newSimVarServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterSimVarTools(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// callTool sends a JSON-RPC 2.0 tools/call request and returns the parsed response.
func callToolSimVar(t *testing.T, serverURL, toolName string, args map[string]any) map[string]any {
	t.Helper()
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(serverURL+"/mcp", "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal response: %v\nbody: %s", err, raw)
	}
	return result
}

// contentText drills into result.content[0].text of a JSON-RPC response.
func contentTextSimVar(t *testing.T, resp map[string]any) string {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'result': %v", resp)
	}
	content, ok := r["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("result missing content array: %v", r)
	}
	item, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] not an object: %v", content[0])
	}
	text, ok := item["text"].(string)
	if !ok {
		t.Fatalf("content[0].text not a string: %v", item)
	}
	return text
}

// isError returns true when the result carries isError=true.
func isErrorSimVar(t *testing.T, resp map[string]any) bool {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		return false
	}
	v, _ := r["isError"].(bool)
	return v
}

// ── get_simvar_value tests ────────────────────────────────────────────────────

// TestGetSimVarValue_Success verifies that a connected bridge returns name, value, and unit.
func TestGetSimVarValue_Success(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:  bridge.StateConnected,
		MockSimVar: bridge.SimVar{Value: 35000},
	}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_value", map[string]any{
		"name": "PLANE ALTITUDE",
		"unit": "feet",
	})

	if isErrorSimVar(t, resp) {
		t.Fatalf("expected success but got error: %s", contentTextSimVar(t, resp))
	}

	text := contentTextSimVar(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["name"] != "PLANE ALTITUDE" {
		t.Errorf("name: got %v, want PLANE ALTITUDE", got["name"])
	}
	if got["value"] != float64(35000) {
		t.Errorf("value: got %v, want 35000", got["value"])
	}
	if got["unit"] != "feet" {
		t.Errorf("unit: got %v, want feet", got["unit"])
	}
}

// TestGetSimVarValue_Disconnected verifies that a disconnected bridge returns BRIDGE_DISCONNECTED.
func TestGetSimVarValue_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_value", map[string]any{
		"name": "PLANE ALTITUDE",
		"unit": "feet",
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for disconnected bridge")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in content, got: %s", text)
	}
}

// TestGetSimVarValue_MissingName verifies that an absent 'name' arg returns INVALID_ARGUMENT.
func TestGetSimVarValue_MissingName(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_value", map[string]any{
		"unit": "feet",
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for missing name")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestGetSimVarValue_MissingUnit verifies that an absent 'unit' arg returns INVALID_ARGUMENT.
func TestGetSimVarValue_MissingUnit(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_value", map[string]any{
		"name": "PLANE ALTITUDE",
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for missing unit")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// ── get_simvar_values tests ───────────────────────────────────────────────────

// TestGetSimVarValues_Success verifies that 2 vars returns an array of 2 results.
func TestGetSimVarValues_Success(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:  bridge.StateConnected,
		MockSimVar: bridge.SimVar{Value: 1000},
	}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": []any{
			map[string]any{"name": "PLANE ALTITUDE", "unit": "feet"},
			map[string]any{"name": "AIRSPEED INDICATED", "unit": "knots"},
		},
	})

	if isErrorSimVar(t, resp) {
		t.Fatalf("expected success but got error: %s", contentTextSimVar(t, resp))
	}

	text := contentTextSimVar(t, resp)
	var results []map[string]any
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0]["name"] != "PLANE ALTITUDE" {
		t.Errorf("results[0].name: got %v, want PLANE ALTITUDE", results[0]["name"])
	}
	if results[0]["unit"] != "feet" {
		t.Errorf("results[0].unit: got %v, want feet", results[0]["unit"])
	}
	if results[1]["name"] != "AIRSPEED INDICATED" {
		t.Errorf("results[1].name: got %v, want AIRSPEED INDICATED", results[1]["name"])
	}
	if results[1]["unit"] != "knots" {
		t.Errorf("results[1].unit: got %v, want knots", results[1]["unit"])
	}
}

// TestGetSimVarValues_Disconnected verifies that a disconnected bridge returns BRIDGE_DISCONNECTED.
func TestGetSimVarValues_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": []any{
			map[string]any{"name": "PLANE ALTITUDE", "unit": "feet"},
		},
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for disconnected bridge")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in content, got: %s", text)
	}
}

// TestGetSimVarValues_EmptyVars verifies that an empty vars array returns INVALID_ARGUMENT.
func TestGetSimVarValues_EmptyVars(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": []any{},
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for empty vars array")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestGetSimVarValues_TooManyVars verifies that 21 vars returns INVALID_ARGUMENT.
func TestGetSimVarValues_TooManyVars(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newSimVarServer(t, mb)

	// Build a slice of 21 var requests.
	vars := make([]any, 21)
	for i := range vars {
		vars[i] = map[string]any{"name": "PLANE ALTITUDE", "unit": "feet"}
	}

	resp := callToolSimVar(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": vars,
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for 21 vars (exceeds max of 20)")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestGetSimVarValue_UnknownVariable verifies that ErrUnknownVariable produces UNKNOWN_VARIABLE error.
func TestGetSimVarValue_UnknownVariable(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:           bridge.StateConnected,
		MockUnknownVarError: true,
	}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_value", map[string]any{
		"name": "NONEXISTENT VARIABLE",
		"unit": "feet",
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for unknown variable")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "UNKNOWN_VARIABLE") {
		t.Errorf("expected UNKNOWN_VARIABLE in content, got: %s", text)
	}
	if !strings.Contains(text, "NONEXISTENT VARIABLE") {
		t.Errorf("expected variable name in content, got: %s", text)
	}
}

// TestGetSimVarValues_UnknownVariable verifies that ErrUnknownVariable from batch returns UNKNOWN_VARIABLE.
func TestGetSimVarValues_UnknownVariable(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:           bridge.StateConnected,
		MockUnknownVarError: true,
	}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": []any{
			map[string]any{"name": "BAD VAR", "unit": "feet"},
		},
	})

	if !isErrorSimVar(t, resp) {
		t.Fatal("expected isError=true for unknown variable in batch")
	}
	text := contentTextSimVar(t, resp)
	if !strings.Contains(text, "UNKNOWN_VARIABLE") {
		t.Errorf("expected UNKNOWN_VARIABLE in content, got: %s", text)
	}
}

// TestGetSimVarValues_PerItemError verifies that when MockSimVarResults contains
// one error entry, that item has an "error" field and the other has a "value" field.
func TestGetSimVarValues_PerItemError(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockSimVarResults: []bridge.SimVarResult{
			{Name: "PLANE ALTITUDE", Unit: "feet", Value: 35000},
			{Name: "BAD VAR", Unit: "units", Error: "bridge: unknown simulation variable"},
		},
	}
	srv := newSimVarServer(t, mb)

	resp := callToolSimVar(t, srv.URL, "get_simvar_values", map[string]any{
		"vars": []any{
			map[string]any{"name": "PLANE ALTITUDE", "unit": "feet"},
			map[string]any{"name": "BAD VAR", "unit": "units"},
		},
	})

	if isErrorSimVar(t, resp) {
		t.Fatalf("expected top-level success but got error: %s", contentTextSimVar(t, resp))
	}

	text := contentTextSimVar(t, resp)
	var results []map[string]any
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First item should have a value field and no error.
	if _, hasValue := results[0]["value"]; !hasValue {
		t.Errorf("results[0] expected 'value' field, got: %v", results[0])
	}
	if errField, hasError := results[0]["error"]; hasError {
		t.Errorf("results[0] unexpected 'error' field: %v", errField)
	}

	// Second item should have an error field and no value.
	if _, hasError := results[1]["error"]; !hasError {
		t.Errorf("results[1] expected 'error' field, got: %v", results[1])
	}
	if valField, hasValue := results[1]["value"]; hasValue {
		t.Errorf("results[1] unexpected 'value' field: %v", valField)
	}
}
