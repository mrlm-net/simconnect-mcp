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

func newAirportServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterAirportTools(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func parseAirportJSON(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse content JSON: %v\ntext: %s", err, text)
	}
	return got
}

// ── get_airports_in_range tests ───────────────────────────────────────────────

func TestGetAirportsInRange_Connected(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirports: []bridge.AirportEntry{
			{ICAO: "LPMA", Region: "LP", Latitude: 32.697, Longitude: -16.778, AltitudeM: 58, DistanceKM: 0.5},
			{ICAO: "LPPS", Region: "LP", Latitude: 33.073, Longitude: -16.380, AltitudeM: 337, DistanceKM: 45.2},
			{ICAO: "LPFL", Region: "LP", Latitude: 39.456, Longitude: -31.131, AltitudeM: 97, DistanceKM: 820.0},
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airports_in_range", map[string]any{"radius_km": float64(100)})
	got := parseAirportJSON(t, resp)

	if got["count"].(float64) != 2 {
		t.Errorf("expected count=2 (airports within 100km), got %v", got["count"])
	}
	if got["radius_km"].(float64) != 100 {
		t.Errorf("expected radius_km=100, got %v", got["radius_km"])
	}
	airports := got["airports"].([]any)
	if len(airports) != 2 {
		t.Fatalf("expected 2 airport entries, got %d", len(airports))
	}
	first := airports[0].(map[string]any)
	if first["icao"] != "LPMA" {
		t.Errorf("expected first airport LPMA, got %v", first["icao"])
	}
}

func TestGetAirportsInRange_DefaultRadius(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirports: []bridge.AirportEntry{
			{ICAO: "LPMA", Region: "LP", DistanceKM: 1.0},
			{ICAO: "LPPS", Region: "LP", DistanceKM: 100.0},
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airports_in_range", map[string]any{})
	got := parseAirportJSON(t, resp)

	// Default radius is 50km — only LPMA qualifies.
	if got["count"].(float64) != 1 {
		t.Errorf("expected count=1 with default radius 50km, got %v", got["count"])
	}
	if got["radius_km"].(float64) != 50 {
		t.Errorf("expected radius_km=50 (default), got %v", got["radius_km"])
	}
}

func TestGetAirportsInRange_FilterNonICAO(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirports: []bridge.AirportEntry{
			{ICAO: "EDDM", Region: "ED", DistanceKM: 1.0},   // valid — kept
			{ICAO: "EDB1", Region: "ED", DistanceKM: 2.0},   // digit — filtered
			{ICAO: "EDF8V", Region: "ED", DistanceKM: 3.0},  // 5 chars + digit — filtered
			{ICAO: "ETSE", Region: "ET", DistanceKM: 12.0},  // valid — kept
		},
	}
	srv := newAirportServer(t, mb)

	// Default (expanded=false): only 4-letter all-caps codes.
	resp := callToolEvent(t, srv.URL, "get_airports_in_range", map[string]any{"radius_km": float64(50)})
	got := parseAirportJSON(t, resp)
	if got["count"].(float64) != 2 {
		t.Errorf("expected count=2 (only valid ICAO), got %v", got["count"])
	}
}

func TestGetAirportsInRange_Expanded(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirports: []bridge.AirportEntry{
			{ICAO: "EDDM", Region: "ED", DistanceKM: 1.0},
			{ICAO: "EDB1", Region: "ED", DistanceKM: 2.0},
			{ICAO: "EDF8V", Region: "ED", DistanceKM: 3.0},
		},
	}
	srv := newAirportServer(t, mb)

	// expanded=true: all entries within radius are returned.
	resp := callToolEvent(t, srv.URL, "get_airports_in_range", map[string]any{
		"radius_km": float64(50),
		"expanded":  true,
	})
	got := parseAirportJSON(t, resp)
	if got["count"].(float64) != 3 {
		t.Errorf("expected count=3 (all entries with expanded=true), got %v", got["count"])
	}
	if got["expanded"].(bool) != true {
		t.Errorf("expected expanded=true in response, got %v", got["expanded"])
	}
}

func TestGetAirportsInRange_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airports_in_range", map[string]any{})
	got := parseAirportJSON(t, resp)

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key when disconnected")
	}
}

// ── get_nearest_airport tests ─────────────────────────────────────────────────

func TestGetNearestAirport_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirports: []bridge.AirportEntry{
			{ICAO: "LPMA", Region: "LP", Latitude: 32.697, Longitude: -16.778, AltitudeM: 58, DistanceKM: 0.5},
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_nearest_airport", map[string]any{})
	got := parseAirportJSON(t, resp)

	if got["icao"] != "LPMA" {
		t.Errorf("expected icao=LPMA, got %v", got["icao"])
	}
	if got["distance_km"].(float64) != 0.5 {
		t.Errorf("expected distance_km=0.5, got %v", got["distance_km"])
	}
}

func TestGetNearestAirport_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_nearest_airport", map[string]any{})
	got := parseAirportJSON(t, resp)

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key when disconnected")
	}
}

// ── get_airport_details tests ─────────────────────────────────────────────────

func TestGetAirportDetails_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportDetails: &bridge.AirportDetails{
			ICAO:      "LPMA",
			Region:    "LP",
			Name:      "Madeira",
			Name64:    "Aeroporto Internacional da Madeira Cristiano Ronaldo",
			Latitude:  32.697,
			Longitude: -16.778,
			AltitudeM: 58,
			RunwayCount: 1,
			Runways: []bridge.AirportRunway{
				{Heading: 54.0, LengthFt: 9124, WidthFt: 148, Surface: "Asphalt"},
			},
			StandCount: 2,
			Stands: []bridge.AirportStand{
				{Number: 1, Type: "Gate Small", Heading: 180},
				{Number: 2, Type: "Gate Small", Heading: 0},
			},
			Frequencies: []bridge.AirportFrequency{
				{Type: "Tower", FreqMHz: 118.1, Name: "LPMA TWR"},
				{Type: "ATIS", FreqMHz: 127.8, Name: "LPMA ATIS"},
			},
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_details", map[string]any{"icao": "LPMA"})
	got := parseAirportJSON(t, resp)

	if got["icao"] != "LPMA" {
		t.Errorf("expected icao=LPMA, got %v", got["icao"])
	}
	if got["name"] != "Madeira" {
		t.Errorf("expected name=Madeira, got %v", got["name"])
	}
	if got["runway_count"].(float64) != 1 {
		t.Errorf("expected runway_count=1, got %v", got["runway_count"])
	}
	if got["stand_count"].(float64) != 2 {
		t.Errorf("expected stand_count=2, got %v", got["stand_count"])
	}
	freqs, _ := got["frequencies"].([]any)
	if len(freqs) != 2 {
		t.Errorf("expected 2 frequencies, got %v", len(freqs))
	}
}

func TestGetAirportDetails_MissingICAO(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_details", map[string]any{})
	text := contentTextEvent(t, resp)

	if text == "" {
		t.Fatal("expected non-empty error text when icao is missing")
	}
	// Should contain INVALID_ARGUMENT
	if !containsStr(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in error, got: %s", text)
	}
}

func TestGetAirportDetails_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_details", map[string]any{"icao": "LPMA"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in error, got: %s", text)
	}
}

// containsStr is a simple string-contains check for test assertions.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
