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

func newNavaidServer(t *testing.T, mb *bridge.MockBridge) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterNavaidTools(mcp, mb)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func parseNavaidJSON(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	text := contentTextEvent(t, resp)
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("parse navaid JSON: %v\ntext: %s", err, text)
	}
	return got
}

// ── get_vors_in_range ─────────────────────────────────────────────────────────

func TestGetVORsInRange_Connected(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockVORs: []bridge.VOREntry{
			{ICAO: "OPO", Region: "LP", Latitude: 41.237, Longitude: -8.670, FrequencyHz: 113600000, DistanceKM: 5.0},
			{ICAO: "LIS", Region: "LP", Latitude: 38.774, Longitude: -9.134, FrequencyHz: 114100000, DistanceKM: 120.0},
			{ICAO: "FAR", Region: "LP", Latitude: 37.014, Longitude: -7.966, FrequencyHz: 115700000, DistanceKM: 350.0},
		},
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vors_in_range", map[string]any{"radius_km": float64(200)})
	got := parseNavaidJSON(t, resp)

	// FAR is at 350km — outside 200km radius.
	if got["count"].(float64) != 2 {
		t.Errorf("expected count=2 (VORs within 200km), got %v", got["count"])
	}
	if got["radius_km"].(float64) != 200 {
		t.Errorf("expected radius_km=200, got %v", got["radius_km"])
	}
	vors := got["vors"].([]any)
	if len(vors) != 2 {
		t.Fatalf("expected 2 VOR entries, got %d", len(vors))
	}
	first := vors[0].(map[string]any)
	if first["icao"] != "OPO" {
		t.Errorf("expected first VOR OPO, got %v", first["icao"])
	}
}

func TestGetVORsInRange_DefaultRadius(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockVORs: []bridge.VOREntry{
			{ICAO: "OPO", Region: "LP", DistanceKM: 10.0},
			{ICAO: "LIS", Region: "LP", DistanceKM: 300.0},
		},
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vors_in_range", map[string]any{})
	got := parseNavaidJSON(t, resp)

	// Default radius is 200km — only OPO qualifies.
	if got["count"].(float64) != 1 {
		t.Errorf("expected count=1 with default 200km radius, got %v", got["count"])
	}
	if got["radius_km"].(float64) != 200 {
		t.Errorf("expected radius_km=200 (default), got %v", got["radius_km"])
	}
}

func TestGetVORsInRange_RadiusCap(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockVORs:  []bridge.VOREntry{{ICAO: "OPO", DistanceKM: 400.0}},
	}
	srv := newNavaidServer(t, mb)

	// radius_km > 500 should be capped to 500.
	resp := callToolEvent(t, srv.URL, "get_vors_in_range", map[string]any{"radius_km": float64(9999)})
	got := parseNavaidJSON(t, resp)

	if got["radius_km"].(float64) != 500 {
		t.Errorf("expected radius_km capped to 500, got %v", got["radius_km"])
	}
	// OPO at 400km is within the capped 500km.
	if got["count"].(float64) != 1 {
		t.Errorf("expected count=1 (within capped 500km), got %v", got["count"])
	}
}

func TestGetVORsInRange_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vors_in_range", map[string]any{})
	got := parseNavaidJSON(t, resp)

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key when disconnected")
	}
}

// ── get_vor_details ───────────────────────────────────────────────────────────

func TestGetVORDetails_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockVORDetails: &bridge.VORDetails{
			ICAO:         "OPO",
			Region:       "LP",
			Name:         "Porto",
			Latitude:     41.237,
			Longitude:    -8.670,
			FrequencyHz:  113600000,
			FrequencyMHz: 113.6,
			MagVar:       -3.2,
			NavRangeNM:   130,
			IsNAV:        true,
			IsDME:        true,
		},
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vor_details", map[string]any{"icao": "OPO"})
	got := parseNavaidJSON(t, resp)

	if got["icao"] != "OPO" {
		t.Errorf("expected icao=OPO, got %v", got["icao"])
	}
	if got["name"] != "Porto" {
		t.Errorf("expected name=Porto, got %v", got["name"])
	}
	if got["frequency_mhz"].(float64) != 113.6 {
		t.Errorf("expected frequency_mhz=113.6, got %v", got["frequency_mhz"])
	}
	if got["is_nav"] != true {
		t.Errorf("expected is_nav=true, got %v", got["is_nav"])
	}
	if got["is_dme"] != true {
		t.Errorf("expected is_dme=true, got %v", got["is_dme"])
	}
}

func TestGetVORDetails_NotFound(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:      bridge.StateConnected,
		MockVORDetails: nil,
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vor_details", map[string]any{"icao": "ZZZ"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "VOR_NOT_FOUND") {
		t.Errorf("expected VOR_NOT_FOUND, got: %s", text)
	}
}

func TestGetVORDetails_MissingICAO(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vor_details", map[string]any{})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT, got: %s", text)
	}
}

func TestGetVORDetails_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_vor_details", map[string]any{"icao": "OPO"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED, got: %s", text)
	}
}

// ── get_ndbs_in_range ─────────────────────────────────────────────────────────

func TestGetNDBsInRange_Connected(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockNDBs: []bridge.NDBEntry{
			{ICAO: "LIS", Region: "LP", FrequencyHz: 391000, DistanceKM: 50.0},
			{ICAO: "FAR", Region: "LP", FrequencyHz: 356000, DistanceKM: 250.0},
		},
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_ndbs_in_range", map[string]any{"radius_km": float64(200)})
	got := parseNavaidJSON(t, resp)

	if got["count"].(float64) != 1 {
		t.Errorf("expected count=1 (NDBs within 200km), got %v", got["count"])
	}
	ndbs := got["ndbs"].([]any)
	first := ndbs[0].(map[string]any)
	if first["icao"] != "LIS" {
		t.Errorf("expected first NDB LIS, got %v", first["icao"])
	}
}

func TestGetNDBsInRange_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_ndbs_in_range", map[string]any{})
	got := parseNavaidJSON(t, resp)

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key when disconnected")
	}
}

// ── get_ndb_details ───────────────────────────────────────────────────────────

func TestGetNDBDetails_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockNDBDetails: &bridge.NDBDetails{
			ICAO:        "LIS",
			Region:      "LP",
			Name:        "Lisboa",
			Latitude:    38.774,
			Longitude:   -9.134,
			FrequencyHz: 391000,
			FrequencyKHz: 391.0,
			IsTerminal:  false,
		},
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_ndb_details", map[string]any{"icao": "LIS"})
	got := parseNavaidJSON(t, resp)

	if got["icao"] != "LIS" {
		t.Errorf("expected icao=LIS, got %v", got["icao"])
	}
	if got["name"] != "Lisboa" {
		t.Errorf("expected name=Lisboa, got %v", got["name"])
	}
	if got["frequency_khz"].(float64) != 391.0 {
		t.Errorf("expected frequency_khz=391.0, got %v", got["frequency_khz"])
	}
}

func TestGetNDBDetails_NotFound(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:      bridge.StateConnected,
		MockNDBDetails: nil,
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_ndb_details", map[string]any{"icao": "ZZZ"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "NDB_NOT_FOUND") {
		t.Errorf("expected NDB_NOT_FOUND, got: %s", text)
	}
}

func TestGetNDBDetails_MissingICAO(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_ndb_details", map[string]any{})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT, got: %s", text)
	}
}

func TestGetNDBDetails_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_ndb_details", map[string]any{"icao": "LIS"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED, got: %s", text)
	}
}

// ── get_waypoints_in_range ────────────────────────────────────────────────────

func TestGetWaypointsInRange_Connected(t *testing.T) {
	wpts := make([]bridge.WaypointEntry, 5)
	for i := range wpts {
		wpts[i] = bridge.WaypointEntry{
			ICAO:       "ABRI" + string(rune('A'+i)),
			Region:     "LP",
			DistanceKM: float64(i+1) * 10,
		}
	}
	mb := &bridge.MockBridge{
		MockState:     bridge.StateConnected,
		MockWaypoints: wpts,
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_waypoints_in_range", map[string]any{"radius_km": float64(35)})
	got := parseNavaidJSON(t, resp)

	// waypoints at 10, 20, 30 are within 35km; 40 and 50 are not.
	if got["count"].(float64) != 3 {
		t.Errorf("expected count=3, got %v", got["count"])
	}
	if got["radius_km"].(float64) != 35 {
		t.Errorf("expected radius_km=35, got %v", got["radius_km"])
	}
}

func TestGetWaypointsInRange_LimitCap(t *testing.T) {
	wpts := make([]bridge.WaypointEntry, 10)
	for i := range wpts {
		wpts[i] = bridge.WaypointEntry{ICAO: "WPT", DistanceKM: float64(i + 1)}
	}
	mb := &bridge.MockBridge{
		MockState:     bridge.StateConnected,
		MockWaypoints: wpts,
	}
	srv := newNavaidServer(t, mb)

	// Limit to 3 entries.
	resp := callToolEvent(t, srv.URL, "get_waypoints_in_range", map[string]any{
		"radius_km": float64(500),
		"limit":     float64(3),
	})
	got := parseNavaidJSON(t, resp)

	if got["count"].(float64) != 3 {
		t.Errorf("expected count=3 (limited), got %v", got["count"])
	}
	if got["limit"].(float64) != 3 {
		t.Errorf("expected limit=3, got %v", got["limit"])
	}
}

func TestGetWaypointsInRange_LimitCapMax(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:     bridge.StateConnected,
		MockWaypoints: []bridge.WaypointEntry{{ICAO: "WPT", DistanceKM: 1.0}},
	}
	srv := newNavaidServer(t, mb)

	// limit > 1000 should be capped to 1000.
	resp := callToolEvent(t, srv.URL, "get_waypoints_in_range", map[string]any{"limit": float64(9999)})
	got := parseNavaidJSON(t, resp)

	if got["limit"].(float64) != 1000 {
		t.Errorf("expected limit capped to 1000, got %v", got["limit"])
	}
}

func TestGetWaypointsInRange_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_waypoints_in_range", map[string]any{})
	got := parseNavaidJSON(t, resp)

	if _, ok := got["error"]; !ok {
		t.Error("expected 'error' key when disconnected")
	}
}

// ── get_waypoint_details ──────────────────────────────────────────────────────

func TestGetWaypointDetails_Found(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState: bridge.StateConnected,
		MockWaypointDetails: &bridge.WaypointDetails{
			ICAO:       "ABRIX",
			Region:     "LP",
			Latitude:   38.5,
			Longitude:  -9.0,
			MagVar:     -3.5,
			NRoutes:    4,
			IsTerminal: false,
		},
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_waypoint_details", map[string]any{"icao": "ABRIX"})
	got := parseNavaidJSON(t, resp)

	if got["icao"] != "ABRIX" {
		t.Errorf("expected icao=ABRIX, got %v", got["icao"])
	}
	if got["n_routes"].(float64) != 4 {
		t.Errorf("expected n_routes=4, got %v", got["n_routes"])
	}
	if got["is_terminal"] != false {
		t.Errorf("expected is_terminal=false, got %v", got["is_terminal"])
	}
}

func TestGetWaypointDetails_NotFound(t *testing.T) {
	mb := &bridge.MockBridge{
		MockState:           bridge.StateConnected,
		MockWaypointDetails: nil,
	}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_waypoint_details", map[string]any{"icao": "ZZZXX"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "WAYPOINT_NOT_FOUND") {
		t.Errorf("expected WAYPOINT_NOT_FOUND, got: %s", text)
	}
}

func TestGetWaypointDetails_MissingICAO(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateConnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_waypoint_details", map[string]any{})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT, got: %s", text)
	}
}

func TestGetWaypointDetails_Disconnected(t *testing.T) {
	mb := &bridge.MockBridge{MockState: bridge.StateDisconnected}
	srv := newNavaidServer(t, mb)

	resp := callToolEvent(t, srv.URL, "get_waypoint_details", map[string]any{"icao": "ABRIX"})
	text := contentTextEvent(t, resp)

	if !containsStr(text, "BRIDGE_DISCONNECTED") {
		t.Errorf("expected BRIDGE_DISCONNECTED, got: %s", text)
	}
}
