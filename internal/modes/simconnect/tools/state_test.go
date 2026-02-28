//go:build windows

package tools

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// newStateServer returns a test HTTP server with RegisterStateTools mounted
// using the provided MockBridge. The server is closed when t completes.
func newStateServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterStateTools(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// callToolState sends a JSON-RPC 2.0 tools/call request and returns the parsed response.
func callToolState(t *testing.T, serverURL, toolName string, args map[string]any) map[string]any {
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

// contentTextState drills into result.content[0].text of a JSON-RPC response.
func contentTextState(t *testing.T, resp map[string]any) string {
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

// isErrorState returns true when the result carries isError=true.
func isErrorState(t *testing.T, resp map[string]any) bool {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		return false
	}
	v, _ := r["isError"].(bool)
	return v
}

// ── get_sim_state tests ───────────────────────────────────────────────────────

// TestGetSimState_Connected verifies that a connected bridge returns the full
// state snapshot with connected=true, paused, current_flight, and simulator_version.
func TestGetSimState_Connected(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockSimState: bridge.SimState{
			Paused:           false,
			CurrentFlight:    "test.flt",
			SimulatorVersion: "MSFS 2024",
		},
	}
	srv := newStateServer(t, mb)

	resp := callToolState(t, srv.URL, "get_sim_state", map[string]any{})

	if isErrorState(t, resp) {
		t.Fatalf("expected success but got error: %s", contentTextState(t, resp))
	}

	text := contentTextState(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["connected"] != true {
		t.Errorf("connected: got %v, want true", got["connected"])
	}
	if got["paused"] != false {
		t.Errorf("paused: got %v, want false", got["paused"])
	}
	if got["current_flight"] != "test.flt" {
		t.Errorf("current_flight: got %v, want test.flt", got["current_flight"])
	}
	if got["simulator_version"] != "MSFS 2024" {
		t.Errorf("simulator_version: got %v, want MSFS 2024", got["simulator_version"])
	}
}

// TestGetSimState_Disconnected verifies that a disconnected bridge returns
// {connected: false} as a successful (non-error) result — not an error response.
// LLM clients must be able to safely poll get_sim_state without triggering error-recovery flows.
func TestGetSimState_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newStateServer(t, mb)

	resp := callToolState(t, srv.URL, "get_sim_state", map[string]any{})

	// Disconnected must NOT produce an error result — it's a normal polling response.
	if isErrorState(t, resp) {
		t.Fatalf("expected success (not error) for disconnected state, got: %s", contentTextState(t, resp))
	}

	text := contentTextState(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}

	if got["connected"] != false {
		t.Errorf("connected: got %v, want false", got["connected"])
	}

	// The disconnected response must only contain connected=false — no other fields.
	if len(got) != 1 {
		t.Errorf("disconnected response should contain only 'connected' field, got keys: %v", got)
	}
}
