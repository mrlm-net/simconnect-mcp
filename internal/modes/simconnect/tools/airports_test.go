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
			{ICAO: "IABX", Region: "ED", DistanceKM: 2.0},   // starts with 'I' (excluded) — filtered
			{ICAO: "EDF8V", Region: "ED", DistanceKM: 3.0},  // 5 chars — filtered (len != 4)
			{ICAO: "ETSE", Region: "ET", DistanceKM: 12.0},  // valid — kept
		},
	}
	srv := newAirportServer(t, mb)

	// Default (expanded=false): filter by IsICAOCode (4 chars, valid first letter).
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
				{Heading: 54.0, LengthM: 2781, WidthM: 45, Surface: "Asphalt"},
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

// ── get_airport_taxiways tests ────────────────────────────────────────────────

func TestGetAirportTaxiways_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportTaxiways: &bridge.AirportTaxiways{
			ICAO:       "EDDM",
			NameCount:  2,
			PointCount: 3,
			PathCount:  1,
			Names:      []string{"A", "B"},
			Points: []bridge.TaxiwayPoint{
				{Type: "NORMAL", Orientation: "FORWARD", BiasXM: 10.0, BiasZM: -5.0},
				{Type: "HOLD_SHORT", Orientation: "FORWARD", BiasXM: 20.0, BiasZM: 0.0},
				{Type: "NORMAL", Orientation: "FORWARD", BiasXM: 30.0, BiasZM: 5.0},
			},
			Paths: []bridge.TaxiwayPath{
				{Type: "TAXI", WidthM: 22.9, StartNode: 0, EndNode: 1, NameIndex: 0},
			},
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{"icao": "EDDM"})
	got := parseAirportJSON(t, resp)

	if got["icao"] != "EDDM" {
		t.Errorf("expected icao=EDDM, got %v", got["icao"])
	}
	if got["name_count"].(float64) != 2 {
		t.Errorf("expected name_count=2, got %v", got["name_count"])
	}
	if got["path_count"].(float64) != 1 {
		t.Errorf("expected path_count=1, got %v", got["path_count"])
	}
	names := got["names"].([]any)
	if names[0] != "A" {
		t.Errorf("expected first name=A, got %v", names[0])
	}
}

func TestGetAirportTaxiways_MissingICAO(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in error, got: %s", text)
	}
}

func TestGetAirportTaxiways_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{"icao": "EDDM"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in error, got: %s", text)
	}
}

func TestGetAirportTaxiways_NotFound(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:           bridge.StateConnected,
		MockAirportTaxiways: nil,
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{"icao": "ZZZZ"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "TAXIWAY_NOT_FOUND") {
		t.Errorf("expected TAXIWAY_NOT_FOUND in error, got: %s", text)
	}
}

// ── get_airport_parkings tests ────────────────────────────────────────────────

func TestGetAirportParkings_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportParkings: &bridge.AirportParkings{
			ICAO:         "EDDM",
			ParkingCount: 2,
			Parkings: []bridge.AirportParking{
				{
					Type:        "Gate Heavy",
					Name:        "GATE",
					Suffix:      "GATE_A",
					Number:      1,
					Orientation: "FORWARD",
					HeadingDeg:  185.0,
					RadiusM:     36.0,
					BiasXM:      423.2,
					BiasZM:      -81.5,
				},
				{
					Type:        "Ramp GA",
					Name:        "PARKING",
					Suffix:      "NONE",
					Number:      2,
					Orientation: "FORWARD",
					HeadingDeg:  90.0,
					RadiusM:     15.0,
					BiasXM:      100.0,
					BiasZM:      50.0,
				},
			},
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_parkings", map[string]any{"icao": "EDDM"})
	got := parseAirportJSON(t, resp)

	if got["icao"] != "EDDM" {
		t.Errorf("expected icao=EDDM, got %v", got["icao"])
	}
	if got["parking_count"].(float64) != 2 {
		t.Errorf("expected parking_count=2, got %v", got["parking_count"])
	}
	parkings := got["parkings"].([]any)
	first := parkings[0].(map[string]any)
	if first["type"] != "Gate Heavy" {
		t.Errorf("expected type=Gate Heavy, got %v", first["type"])
	}
	if first["heading_deg"].(float64) != 185.0 {
		t.Errorf("expected heading_deg=185.0, got %v", first["heading_deg"])
	}
}

func TestGetAirportParkings_MissingICAO(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_parkings", map[string]any{})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT in error, got: %s", text)
	}
}

func TestGetAirportParkings_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_parkings", map[string]any{"icao": "EDDM"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED in error, got: %s", text)
	}
}

func TestGetAirportParkings_NotFound(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:           bridge.StateConnected,
		MockAirportParkings: nil,
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_parkings", map[string]any{"icao": "ZZZZ"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "PARKING_NOT_FOUND") {
		t.Errorf("expected PARKING_NOT_FOUND in error, got: %s", text)
	}
}

// ── get_airport_taxiways truncation tests ─────────────────────────────────────

// TestGetAirportTaxiways_DefaultCap verifies paths are capped at 500 by default.
func TestGetAirportTaxiways_DefaultCap(t *testing.T) {
	// Build a mock with 600 paths.
	paths := make([]bridge.TaxiwayPath, 600)
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportTaxiways: &bridge.AirportTaxiways{
			ICAO:       "EDDM",
			NameCount:  5,
			PointCount: 700,
			PathCount:  600,
			Names:      []string{"A", "B", "C", "D", "E"},
			Points:     []bridge.TaxiwayPoint{},
			Paths:      paths,
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{"icao": "EDDM"})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	gotPaths := got["paths"].([]any)
	if len(gotPaths) != 500 {
		t.Errorf("expected 500 paths (default cap), got %d", len(gotPaths))
	}
	if got["truncated"] != true {
		t.Errorf("expected truncated=true, got %v", got["truncated"])
	}
	if got["truncated_to"] != float64(500) {
		t.Errorf("expected truncated_to=500, got %v", got["truncated_to"])
	}
	if got["path_count"] != float64(600) {
		t.Errorf("expected path_count=600 (pre-truncation), got %v", got["path_count"])
	}
}

// TestGetAirportTaxiways_ExplicitCapWithTruncation verifies explicit max_paths works.
func TestGetAirportTaxiways_ExplicitCapWithTruncation(t *testing.T) {
	paths := make([]bridge.TaxiwayPath, 200)
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportTaxiways: &bridge.AirportTaxiways{
			ICAO:      "KLAX",
			PathCount: 200,
			Names:     []string{},
			Points:    []bridge.TaxiwayPoint{},
			Paths:     paths,
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{
		"icao":      "KLAX",
		"max_paths": float64(100),
	})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	gotPaths := got["paths"].([]any)
	if len(gotPaths) != 100 {
		t.Errorf("expected 100 paths, got %d", len(gotPaths))
	}
	if got["truncated"] != true {
		t.Errorf("expected truncated=true")
	}
}

// TestGetAirportTaxiways_NoTruncation verifies small airports return all paths without truncated field.
func TestGetAirportTaxiways_NoTruncation(t *testing.T) {
	paths := make([]bridge.TaxiwayPath, 50)
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportTaxiways: &bridge.AirportTaxiways{
			ICAO:      "LKPD",
			PathCount: 50,
			Names:     []string{"A"},
			Points:    []bridge.TaxiwayPoint{},
			Paths:     paths,
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_airport_taxiways", map[string]any{"icao": "LKPD"})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	gotPaths := got["paths"].([]any)
	if len(gotPaths) != 50 {
		t.Errorf("expected all 50 paths, got %d", len(gotPaths))
	}
	if _, hasTruncated := got["truncated"]; hasTruncated {
		t.Errorf("expected no truncated field for small airport")
	}
}

// ── get_taxiway_names tests ───────────────────────────────────────────────────

// TestGetTaxiwayNames_ReturnsNamesOnly verifies the lightweight tool returns only names.
func TestGetTaxiwayNames_ReturnsNamesOnly(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockAirportTaxiways: &bridge.AirportTaxiways{
			ICAO:      "EDDM",
			NameCount: 3,
			Names:     []string{"A", "B", "C"},
			Points:    []bridge.TaxiwayPoint{},
			Paths:     make([]bridge.TaxiwayPath, 1000),
		},
	}
	srv := newAirportServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_taxiway_names", map[string]any{"icao": "EDDM"})
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}

	if _, hasPaths := got["paths"]; hasPaths {
		t.Error("get_taxiway_names must not include paths")
	}
	if _, hasPoints := got["points"]; hasPoints {
		t.Error("get_taxiway_names must not include points")
	}
	names := got["names"].([]any)
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
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
