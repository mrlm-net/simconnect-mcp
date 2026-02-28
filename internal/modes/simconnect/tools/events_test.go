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

// newEventServer returns a test HTTP server with RegisterEventTools mounted
// using the provided MockBridge. The server is closed when t completes.
func newEventServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterEventTools(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// callToolEvent sends a JSON-RPC 2.0 tools/call request and returns the parsed response.
func callToolEvent(t *testing.T, serverURL, toolName string, args map[string]any) map[string]any {
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

// contentTextEvent drills into result.content[0].text of a JSON-RPC response.
func contentTextEvent(t *testing.T, resp map[string]any) string {
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

// isErrorEvent returns true when the result carries isError=true.
func isErrorEvent(t *testing.T, resp map[string]any) bool {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		return false
	}
	v, _ := r["isError"].(bool)
	return v
}

// ── transmit_event tests ──────────────────────────────────────────────────────

// TestTransmitEvent_Success verifies that a connected bridge with a valid name
// returns {success:true, event:name} and records the call on MockBridge.
func TestTransmitEvent_Success(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{
		"name": "LANDING_LIGHTS_TOGGLE",
	})

	if isErrorEvent(t, resp) {
		t.Fatalf("expected success but got error: %s", contentTextEvent(t, resp))
	}

	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["success"] != true {
		t.Errorf("success: got %v, want true", got["success"])
	}
	if got["event"] != "LANDING_LIGHTS_TOGGLE" {
		t.Errorf("event: got %v, want LANDING_LIGHTS_TOGGLE", got["event"])
	}

	// Verify the call was recorded on the mock.
	if len(mb.TransmitEventCalls) != 1 {
		t.Fatalf("expected 1 TransmitEvent call, got %d", len(mb.TransmitEventCalls))
	}
	if mb.TransmitEventCalls[0].Name != "LANDING_LIGHTS_TOGGLE" {
		t.Errorf("recorded call name: got %q, want LANDING_LIGHTS_TOGGLE", mb.TransmitEventCalls[0].Name)
	}
}

// TestTransmitEvent_Disconnected verifies that a disconnected bridge returns BRIDGE_DISCONNECTED.
func TestTransmitEvent_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{
		"name": "LANDING_LIGHTS_TOGGLE",
	})

	if !isErrorEvent(t, resp) {
		t.Fatal("expected isError=true for disconnected bridge")
	}
	text := contentTextEvent(t, resp)
	if !strings.Contains(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in content, got: %s", text)
	}
}

// TestTransmitEvent_MissingName verifies that an absent 'name' arg returns INVALID_ARGUMENT.
func TestTransmitEvent_MissingName(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{})

	if !isErrorEvent(t, resp) {
		t.Fatal("expected isError=true for missing name")
	}
	text := contentTextEvent(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestTransmitEvent_NegativeValue verifies that value=-1 returns INVALID_ARGUMENT.
func TestTransmitEvent_NegativeValue(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{
		"name":  "SOME_EVENT",
		"value": float64(-1),
	})

	if !isErrorEvent(t, resp) {
		t.Fatal("expected isError=true for value=-1")
	}
	text := contentTextEvent(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestTransmitEvent_ValueExceedsMaxUint32 verifies that value=4294967296 (> MaxUint32)
// returns INVALID_ARGUMENT.
func TestTransmitEvent_ValueExceedsMaxUint32(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{
		"name":  "SOME_EVENT",
		"value": float64(4294967296), // MaxUint32 + 1
	})

	if !isErrorEvent(t, resp) {
		t.Fatal("expected isError=true for value=4294967296")
	}
	text := contentTextEvent(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in content, got: %s", text)
	}
}

// TestTransmitEvent_MaxUint32 verifies that value=4294967295 (MaxUint32) succeeds.
func TestTransmitEvent_MaxUint32(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{
		"name":  "SOME_EVENT",
		"value": float64(4294967295), // MaxUint32
	})

	if isErrorEvent(t, resp) {
		t.Fatalf("expected success for value=MaxUint32 but got error: %s", contentTextEvent(t, resp))
	}

	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}
	if got["success"] != true {
		t.Errorf("success: got %v, want true", got["success"])
	}

	// Verify the value was recorded correctly.
	if len(mb.TransmitEventCalls) != 1 {
		t.Fatalf("expected 1 TransmitEvent call, got %d", len(mb.TransmitEventCalls))
	}
	if mb.TransmitEventCalls[0].Value != 4294967295 {
		t.Errorf("recorded value: got %d, want 4294967295", mb.TransmitEventCalls[0].Value)
	}
}

// TestTransmitEvent_ZeroValue verifies that value=0 (default) succeeds.
func TestTransmitEvent_ZeroValue(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newEventServer(t, mb)

	resp := callToolEvent(t, srv.URL, "transmit_event", map[string]any{
		"name":  "SOME_EVENT",
		"value": float64(0),
	})

	if isErrorEvent(t, resp) {
		t.Fatalf("expected success for value=0 but got error: %s", contentTextEvent(t, resp))
	}

	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}
	if got["success"] != true {
		t.Errorf("success: got %v, want true", got["success"])
	}

	if len(mb.TransmitEventCalls) != 1 {
		t.Fatalf("expected 1 TransmitEvent call, got %d", len(mb.TransmitEventCalls))
	}
	if mb.TransmitEventCalls[0].Value != 0 {
		t.Errorf("recorded value: got %d, want 0", mb.TransmitEventCalls[0].Value)
	}
}
