package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// Keep Gin quiet during tests.
	gin.SetMode(gin.TestMode)
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// newRouter builds a minimal Gin engine suitable for testing a single
// middleware without the noise of gin.Default().
func newRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(middlewares...)
	return r
}

var uuidV4Re = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// -----------------------------------------------------------------------
// RequestID
// -----------------------------------------------------------------------

func TestRequestID_PreservesIncomingHeader(t *testing.T) {
	r := newRouter(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-fixed-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "my-fixed-id" {
		t.Errorf("expected preserved request ID %q, got %q", "my-fixed-id", got)
	}
}

func TestRequestID_GeneratesUUIDV4WhenAbsent(t *testing.T) {
	r := newRouter(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := w.Header().Get("X-Request-ID")
	if got == "" {
		t.Fatal("expected X-Request-ID to be set, got empty string")
	}
	if !uuidV4Re.MatchString(got) {
		t.Errorf("generated ID %q does not match UUID v4 pattern", got)
	}
}

func TestRequestID_GetRequestIDMatchesHeader(t *testing.T) {
	var captured string
	r := newRouter(RequestID())
	r.GET("/", func(c *gin.Context) {
		captured = GetRequestID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	headerID := w.Header().Get("X-Request-ID")
	if captured != headerID {
		t.Errorf("GetRequestID returned %q, want %q (header value)", captured, headerID)
	}
}

func TestRequestID_GetRequestIDWithFixedID(t *testing.T) {
	var captured string
	r := newRouter(RequestID())
	r.GET("/", func(c *gin.Context) {
		captured = GetRequestID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "fixed-abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if captured != "fixed-abc" {
		t.Errorf("GetRequestID returned %q, want %q", captured, "fixed-abc")
	}
}

// -----------------------------------------------------------------------
// CORS — debug mode
// -----------------------------------------------------------------------

func TestCORS_DebugMode_LocalhostGetsAllowHeaders(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(gin.TestMode)

	r := newRouter(CORS())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected Allow-Origin %q, got %q", "http://localhost:3000", got)
	}
}

func TestCORS_DebugMode_McpSessionIdInAllowHeaders(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(gin.TestMode)

	r := newRouter(CORS())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Fatal("Access-Control-Allow-Headers header is missing")
	}
	// Check Mcp-Session-Id is present (case-insensitive substring search is
	// acceptable here because the header value is a comma-separated list).
	found := false
	for _, h := range splitHeader(allowHeaders) {
		if http.CanonicalHeaderKey(h) == "Mcp-Session-Id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Mcp-Session-Id not found in Access-Control-Allow-Headers: %q", allowHeaders)
	}
}

// -----------------------------------------------------------------------
// CORS — release mode
// -----------------------------------------------------------------------

func TestCORS_ReleaseMode_NonLocalhostReturns403(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	r := newRouter(CORS())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-localhost in release mode, got %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' object in response body, got: %v", body)
	}
	if errObj["code"] != "FORBIDDEN" {
		t.Errorf("expected code FORBIDDEN, got %v", errObj["code"])
	}
}

func TestCORS_ReleaseMode_LocalhostAllowed(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	r := newRouter(CORS())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	origins := []string{
		"http://localhost:8080",
		"http://127.0.0.1:3000",
	}
	for _, origin := range origins {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("origin %q: expected 200 in release mode, got %d", origin, w.Code)
		}
	}
}

// -----------------------------------------------------------------------
// CORS — OPTIONS preflight
// -----------------------------------------------------------------------

func TestCORS_OptionsPreflight_Returns204(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(gin.TestMode)

	r := newRouter(CORS())
	r.OPTIONS("/api", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/api", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS preflight, got %d", w.Code)
	}
}

// -----------------------------------------------------------------------
// Error handlers
// -----------------------------------------------------------------------

func TestNoRouteHandler_Returns404JSON(t *testing.T) {
	r := gin.New()
	r.NoRoute(NoRouteHandler())

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	assertErrorBody(t, w, "NOT_FOUND")
}

func TestNoMethodHandler_Returns405JSON(t *testing.T) {
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.NoMethod(NoMethodHandler())
	r.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
	assertErrorBody(t, w, "METHOD_NOT_ALLOWED")
}

func TestRecoveryHandler_PanicReturns500JSON(t *testing.T) {
	r := gin.New()
	r.Use(RecoveryHandler())
	r.GET("/boom", func(c *gin.Context) {
		panic("something went very wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	assertErrorBody(t, w, "INTERNAL_ERROR")
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// assertErrorBody decodes the JSON response and asserts that
// body.error.code equals wantCode.
func assertErrorBody(t *testing.T, w *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var env errorEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if env.Error.Code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, env.Error.Code)
	}
	if env.Error.Message == "" {
		t.Error("expected non-empty error message")
	}
}

// splitHeader splits a comma-separated header value and trims whitespace.
func splitHeader(s string) []string {
	var out []string
	for _, part := range splitComma(s) {
		out = append(out, trimSpace(part))
	}
	return out
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
