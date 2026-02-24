// Package integration contains end-to-end tests for the MCP docs mode.
// Tests start the full Gin router in-process via net/http/httptest and
// exercise all 11 registered MCP tools plus the health and routing layer.
//
// Environment:
//   - DOCS_MSFS_VERSION=both  (set via t.Setenv in TestMain)
//   - No network calls — embedded fixture corpus only
//
// Run with: go test -race -v ./tests/integration/
package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/docs"
	"github.com/mrlm-net/simconnect-mcp/internal/server"
)

// ── server bootstrap ─────────────────────────────────────────────────────────

// newTestServer builds and mounts a docs-mode Gin engine backed by the
// embedded corpus. It sets DOCS_MSFS_VERSION=both for the duration of t.
// The returned *httptest.Server is closed automatically when t completes.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("DOCS_MSFS_VERSION", "both")

	gin.SetMode(gin.TestMode)
	r := server.New()

	cfg := docs.Config{
		MSFSVersion: "both",
		// OverridePath intentionally empty — use embedded assets.
	}
	mode := docs.New(cfg)
	if err := mode.Mount(r); err != nil {
		t.Fatalf("docs mode mount failed: %v", err)
	}

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// ── JSON-RPC 2.0 helpers ─────────────────────────────────────────────────────

// callTool POSTs a JSON-RPC 2.0 tools/call request to /mcp and returns the
// parsed response body. It fails the test immediately on any transport error
// or non-200 HTTP status.
func callTool(t *testing.T, serverURL, toolName string, args map[string]any) map[string]any {
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
	return postMCP(t, serverURL, payload)
}

// listTools POSTs a tools/list request to /mcp.
func listTools(t *testing.T, serverURL string) map[string]any {
	t.Helper()
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	return postMCP(t, serverURL, payload)
}

// postMCP sends a JSON-RPC 2.0 request to /mcp and returns the parsed body.
func postMCP(t *testing.T, serverURL string, payload map[string]any) map[string]any {
	t.Helper()

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
		t.Fatalf("POST /mcp: unexpected status %d", resp.StatusCode)
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

// toolResult extracts the result map from a JSON-RPC 2.0 response.
// Fails the test if the field is absent or the wrong type.
func toolResult(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'result': %v", resp)
	}
	return r
}

// contentText returns the text of content[0] in a CallToolResult.
// Parses the outer JSON-RPC result, then drills into content[0].text.
func contentText(t *testing.T, resp map[string]any) string {
	t.Helper()
	result := toolResult(t, resp)
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("result missing content array: %v", result)
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

// isToolError returns true when the result carries IsError=true.
func isToolError(t *testing.T, resp map[string]any) bool {
	t.Helper()
	result := toolResult(t, resp)
	v, _ := result["isError"].(bool)
	return v
}

// parseContentJSON parses the JSON string in content[0].text into dst.
func parseContentJSON(t *testing.T, resp map[string]any, dst any) {
	t.Helper()
	text := contentText(t, resp)
	if err := json.Unmarshal([]byte(text), dst); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestToolsList verifies that tools/list returns exactly 11 named tools.
func TestToolsList(t *testing.T) {
	srv := newTestServer(t)

	resp := listTools(t, srv.URL)
	result := toolResult(t, resp)

	toolsRaw, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list result missing 'tools' array: %v", result)
	}

	want := []string{
		"list_simvars",
		"get_simvar",
		"list_events",
		"get_event",
		"list_functions",
		"get_function",
		"list_structures",
		"get_structure",
		"list_error_codes",
		"get_error_code",
		"search_docs",
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

// TestListSimVars verifies list_simvars returns items and correct page structure.
func TestListSimVars(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "list_simvars", map[string]any{})

	var page struct {
		Items      []any `json:"items"`
		Page       int   `json:"page"`
		PageSize   int   `json:"page_size"`
		TotalItems int   `json:"total_items"`
		TotalPages int   `json:"total_pages"`
	}
	parseContentJSON(t, resp, &page)

	if len(page.Items) == 0 {
		t.Fatal("list_simvars returned no items")
	}
	if page.Page != 1 {
		t.Errorf("expected page=1, got %d", page.Page)
	}
	if page.TotalItems <= 0 {
		t.Errorf("expected total_items > 0, got %d", page.TotalItems)
	}
}

// TestGetSimVar_Found verifies that get_simvar returns a SimVar with a non-empty name.
func TestGetSimVar_Found(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "get_simvar", map[string]any{
		"name": "PLANE ALTITUDE",
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var sv struct {
		Name string `json:"name"`
	}
	parseContentJSON(t, resp, &sv)

	if sv.Name == "" {
		t.Fatal("get_simvar returned SimVar with empty name")
	}
}

// TestGetSimVar_NotFound verifies that an unknown SimVar name returns an IsError response
// with a NOT_FOUND code in the content text.
func TestGetSimVar_NotFound(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "get_simvar", map[string]any{
		"name": "NONEXISTENT_VAR_XYZ",
	})

	if !isToolError(t, resp) {
		t.Fatal("expected isError=true for unknown simvar")
	}

	text := contentText(t, resp)
	if !strings.Contains(text, "NOT_FOUND") {
		t.Errorf("expected content to contain NOT_FOUND, got: %s", text)
	}
}

// TestGetSimVar_CaseInsensitive verifies that lookup with lowercase name succeeds.
func TestGetSimVar_CaseInsensitive(t *testing.T) {
	srv := newTestServer(t)

	upper := callTool(t, srv.URL, "get_simvar", map[string]any{"name": "PLANE ALTITUDE"})
	lower := callTool(t, srv.URL, "get_simvar", map[string]any{"name": "plane altitude"})

	if isToolError(t, upper) {
		t.Fatalf("uppercase lookup failed: %s", contentText(t, upper))
	}
	if isToolError(t, lower) {
		t.Fatalf("lowercase lookup failed: %s", contentText(t, lower))
	}

	var svUpper, svLower struct {
		Name string `json:"name"`
	}
	parseContentJSON(t, upper, &svUpper)
	parseContentJSON(t, lower, &svLower)

	if svUpper.Name == "" || svLower.Name == "" {
		t.Fatal("one or both SimVar names are empty")
	}
	// Both lookups should return the same item.
	if strings.ToUpper(svUpper.Name) != strings.ToUpper(svLower.Name) {
		t.Errorf("case-insensitive mismatch: %q vs %q", svUpper.Name, svLower.Name)
	}
}

// TestListEvents verifies that list_events returns a non-empty list.
func TestListEvents(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "list_events", map[string]any{})

	var page struct {
		Items []any `json:"items"`
	}
	parseContentJSON(t, resp, &page)

	if len(page.Items) == 0 {
		t.Fatal("list_events returned no items")
	}
}

// TestGetEvent_NotFound verifies that an unknown event name returns NOT_FOUND.
func TestGetEvent_NotFound(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "get_event", map[string]any{
		"name": "NONEXISTENT_EVENT_XYZ",
	})

	if !isToolError(t, resp) {
		t.Fatal("expected isError=true for unknown event")
	}

	text := contentText(t, resp)
	if !strings.Contains(text, "NOT_FOUND") {
		t.Errorf("expected content to contain NOT_FOUND, got: %s", text)
	}
}

// TestListFunctions verifies that list_functions returns a non-empty list.
func TestListFunctions(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "list_functions", map[string]any{})

	var page struct {
		Items []any `json:"items"`
	}
	parseContentJSON(t, resp, &page)

	if len(page.Items) == 0 {
		t.Fatal("list_functions returned no items")
	}
}

// TestListStructures verifies that list_structures returns a non-empty list.
func TestListStructures(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "list_structures", map[string]any{})

	var page struct {
		Items []any `json:"items"`
	}
	parseContentJSON(t, resp, &page)

	if len(page.Items) == 0 {
		t.Fatal("list_structures returned no items")
	}
}

// TestListErrorCodes verifies that list_error_codes returns a non-empty list.
func TestListErrorCodes(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "list_error_codes", map[string]any{})

	var page struct {
		Items []any `json:"items"`
	}
	parseContentJSON(t, resp, &page)

	if len(page.Items) == 0 {
		t.Fatal("list_error_codes returned no items")
	}
}

// TestGetErrorCode_ByValue verifies that get_error_code with value=0 returns
// SIMCONNECT_EXCEPTION_NONE.
func TestGetErrorCode_ByValue(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "get_error_code", map[string]any{
		"value": float64(0),
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var ec struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	parseContentJSON(t, resp, &ec)

	if ec.Name != "SIMCONNECT_EXCEPTION_NONE" {
		t.Errorf("expected SIMCONNECT_EXCEPTION_NONE, got %q", ec.Name)
	}
	if ec.Value != 0 {
		t.Errorf("expected value=0, got %d", ec.Value)
	}
}

// TestSearchDocs_All verifies that search_docs with type=all and a known query
// returns at least one result.
func TestSearchDocs_All(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "search_docs", map[string]any{
		"query": "altitude",
		"type":  "all",
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var results struct {
		Total   int   `json:"total"`
		Results []any `json:"results"`
	}
	parseContentJSON(t, resp, &results)

	if results.Total == 0 || len(results.Results) == 0 {
		t.Fatal("search_docs returned no results for query 'altitude'")
	}
}

// TestSearchDocs_TypeFilter verifies that search_docs with type=simvar returns
// only simvar-typed results.
func TestSearchDocs_TypeFilter(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "search_docs", map[string]any{
		"query": "altitude",
		"type":  "simvar",
	})

	if isToolError(t, resp) {
		t.Fatalf("expected success but got error: %s", contentText(t, resp))
	}

	var results struct {
		Total   int `json:"total"`
		Results []struct {
			Type string `json:"type"`
		} `json:"results"`
	}
	parseContentJSON(t, resp, &results)

	if results.Total == 0 || len(results.Results) == 0 {
		t.Fatal("search_docs type=simvar returned no results for query 'altitude'")
	}

	for i, r := range results.Results {
		if r.Type != "simvar" {
			t.Errorf("result[%d] type=%q, want simvar", i, r.Type)
		}
	}
}

// TestSearchDocs_EmptyQuery verifies that an empty query returns an
// INVALID_ARGUMENT tool error.
func TestSearchDocs_EmptyQuery(t *testing.T) {
	srv := newTestServer(t)

	resp := callTool(t, srv.URL, "search_docs", map[string]any{
		"query": "",
	})

	if !isToolError(t, resp) {
		t.Fatal("expected isError=true for empty query")
	}

	text := contentText(t, resp)
	if !strings.Contains(text, "INVALID_ARGUMENT") {
		t.Errorf("expected content to contain INVALID_ARGUMENT, got: %s", text)
	}
}

// TestHealth verifies that GET /health returns 200 with docs_loaded=true,
// simvar_count > 0, and event_count > 0.
func TestHealth(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/health") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		DocsLoaded  bool    `json:"docs_loaded"`
		SimVarCount float64 `json:"simvar_count"` // JSON numbers decode as float64
		EventCount  float64 `json:"event_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode /health body: %v", err)
	}

	if !body.DocsLoaded {
		t.Error("health: docs_loaded is false")
	}
	if body.SimVarCount <= 0 {
		t.Errorf("health: simvar_count=%v, want > 0", body.SimVarCount)
	}
	if body.EventCount <= 0 {
		t.Errorf("health: event_count=%v, want > 0", body.EventCount)
	}
}

// TestUnknownRoute verifies that GET /nonexistent returns HTTP 404 with a
// structured JSON error body (not plain-text HTML).
func TestUnknownRoute(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/nonexistent") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode 404 body: %v", err)
	}
	if body.Error.Code == "" {
		t.Error("404 body missing error.code")
	}
}
