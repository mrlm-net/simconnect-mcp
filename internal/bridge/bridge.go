// Package bridge defines the Bridge interface and the value types it operates on.
// The interface is platform-agnostic; the Windows implementation lives in
// simconnect_bridge.go (//go:build windows).
package bridge

import (
	"context"
	"errors"
)

// ConnectionState represents the current state of the SimConnect connection.
type ConnectionState int

const (
	// StateDisconnected means no active handle; reconnect attempts may be pending.
	StateDisconnected ConnectionState = iota
	// StateConnecting means SimConnect_Open has been called; waiting for confirmation.
	StateConnecting
	// StateConnected means the handle is open and dispatch is running.
	StateConnected
)

// String returns a human-readable connection state label.
func (s ConnectionState) String() string {
	switch s {
	case StateConnected:
		return "connected"
	case StateConnecting:
		return "connecting"
	default:
		return "disconnected"
	}
}

// SimVar is a named simulation variable with its current value.
type SimVar struct {
	Name    string  `json:"name"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	SimTime float64 `json:"sim_time,omitempty"`
}

// SimVarRequest identifies a variable to read.
type SimVarRequest struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
}

// SimVarResult is one element of a GetSimVars batch response.
// Exactly one of Value or Error will be meaningful.
type SimVarResult struct {
	Name    string  `json:"name"`
	Unit    string  `json:"unit"`
	Value   float64 `json:"value,omitempty"`
	SimTime float64 `json:"sim_time,omitempty"`
	Error   string  `json:"error,omitempty"` // non-empty signals per-item failure
}

// SimState is a snapshot of top-level simulator state.
type SimState struct {
	Connected        bool    `json:"connected"`
	Paused           bool    `json:"paused"`
	CurrentFlight    string  `json:"current_flight"`
	SimTime          float64 `json:"sim_time"`
	SimulatorVersion string  `json:"simulator_version"`

	// Aircraft position
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude_ft,omitempty"`

	// Aircraft speed
	GroundSpeed       float64 `json:"ground_speed_kts,omitempty"`
	IndicatedAirspeed float64 `json:"indicated_airspeed_kts,omitempty"`
	VerticalSpeed     float64 `json:"vertical_speed_fpm,omitempty"` // feet per minute

	// Aircraft orientation
	TrueHeading float64 `json:"true_heading_deg,omitempty"`

	// Status
	OnGround bool `json:"on_ground"`
}

// SimEvent represents a simulator lifecycle notification.
type SimEvent struct {
	Name  string `json:"name"`
	Value uint32 `json:"value,omitempty"`
}

// TrafficEntry is one aircraft returned by a nearby-traffic scan.
type TrafficEntry struct {
	ObjectID    uint32  `json:"object_id"`
	Title       string  `json:"title"`
	ATCID       string  `json:"atc_id"`
	ATCAirline  string  `json:"atc_airline"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	AltitudeFt  float64 `json:"altitude_ft"`
	TrueHeading float64 `json:"true_heading_deg"`
	GroundSpeed float64 `json:"ground_speed_kts"`
	OnGround    bool    `json:"on_ground"`
}

// EnrichedTrafficEntry extends TrafficEntry with velocity-derived fields:
// vertical speed, actual ground track, and inferred flight phase.
// It also includes the parking-state and runway-occupancy flags.
type EnrichedTrafficEntry struct {
	ObjectID         uint32  `json:"object_id"`
	Title            string  `json:"title"`
	ATCID            string  `json:"atc_id"`
	ATCAirline       string  `json:"atc_airline"`
	Category         string  `json:"category"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	AltitudeFt       float64 `json:"altitude_ft"`
	TrueHeading      float64 `json:"true_heading_deg"`
	TrackDeg         float64 `json:"track_deg"`          // actual ground track (from velocity vectors)
	GroundSpeed      float64 `json:"ground_speed_kts"`
	VerticalSpeedFPM float64 `json:"vertical_speed_fpm"` // VELOCITY WORLD Y × 60
	OnGround         bool    `json:"on_ground"`
	InParkingState   bool    `json:"in_parking_state"`
	OnAnyRunway      bool    `json:"on_any_runway"`
	FlightPhase      string  `json:"flight_phase"` // PARKED/TAXI/CLIMB/LEVEL/DESCENT/APPROACH/FINAL
}

// AirportEntry holds basic airport info from a facilities list scan.
// DistanceKM is the Haversine distance from the player aircraft's current position.
type AirportEntry struct {
	ICAO       string  `json:"icao"`
	Region     string  `json:"region"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	AltitudeM  float64 `json:"altitude_m"`
	DistanceKM float64 `json:"distance_km"`
}

// AirportRunway holds data for a single runway at an airport.
// SimConnect's LENGTH and WIDTH fields are returned in metres.
type AirportRunway struct {
	Name    string  `json:"name"`     // e.g. "08L/26R"
	Heading float64 `json:"heading_deg"`
	LengthM float64 `json:"length_m"`
	WidthM  float64 `json:"width_m"`
	Surface string  `json:"surface"`
}

// AirportHelipad holds data for a single helipad at an airport.
type AirportHelipad struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	AltitudeM float64 `json:"altitude_m"`
	Heading   float64 `json:"heading_deg"`
	LengthM   float64 `json:"length_m"`
	WidthM    float64 `json:"width_m"`
	Surface   string  `json:"surface"`
	Type      string  `json:"type"` // None / H / Square / Circle / Medical
}

// AirportStand holds data for a single parking stand / gate.
type AirportStand struct {
	Number  int     `json:"number"`
	Type    string  `json:"type"`
	Heading float64 `json:"heading_deg"`
}

// AirportFrequency holds a single ATC frequency at an airport.
type AirportFrequency struct {
	Type    string  `json:"type"`
	FreqMHz float64 `json:"freq_mhz"`
	Name    string  `json:"name"`
}

// AirportApproach holds summary data for one instrument approach procedure.
type AirportApproach struct {
	Type        string `json:"type"`         // GPS/ILS/VOR/RNAV/NDB/LOCALIZER/SDF/LDA/VORDME/NDBDME/LOCALIZER_BACK_COURSE
	Runway      string `json:"runway"`        // e.g. "08L"; empty for circling approaches
	HasLNAV     bool   `json:"has_lnav"`
	HasLNAVVNAV bool   `json:"has_lnavvnav"`
	HasLP       bool   `json:"has_lp"`
	HasLPV      bool   `json:"has_lpv"`
}

// AirportProcedure holds summary data for one SID (departure) or STAR (arrival).
type AirportProcedure struct {
	Name               string `json:"name"`
	RunwayTransitions  int    `json:"runway_transitions"`
	EnrouteTransitions int    `json:"enroute_transitions"`
}

// VOREntry holds basic VOR station info from a facilities list scan.
// DistanceKM is the Haversine distance from the player aircraft's current position.
type VOREntry struct {
	ICAO        string  `json:"icao"`
	Region      string  `json:"region"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	AltitudeM   float64 `json:"altitude_m"`
	FrequencyHz uint32  `json:"frequency_hz"`
	MagVar      float64 `json:"magvar_deg"`
	DistanceKM  float64 `json:"distance_km"`
}

// VORDetails holds detailed facility data for a specific VOR station.
type VORDetails struct {
	ICAO          string  `json:"icao"`
	Region        string  `json:"region"`
	Name          string  `json:"name"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	AltitudeM     float64 `json:"altitude_m"`
	FrequencyHz   uint32  `json:"frequency_hz"`
	FrequencyMHz  float64 `json:"frequency_mhz"`
	MagVar        float64 `json:"magvar_deg"`
	NavRangeNM    float64 `json:"nav_range_nm"`
	IsNAV         bool    `json:"is_nav"`
	IsDME         bool    `json:"is_dme"`
	IsTACAN       bool    `json:"is_tacan"`
	HasGlideSlope bool    `json:"has_glide_slope"`
	HasBackCourse bool    `json:"has_back_course"`
}

// NDBEntry holds basic NDB station info from a facilities list scan.
// DistanceKM is the Haversine distance from the player aircraft's current position.
type NDBEntry struct {
	ICAO        string  `json:"icao"`
	Region      string  `json:"region"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	AltitudeM   float64 `json:"altitude_m"`
	FrequencyHz uint32  `json:"frequency_hz"`
	MagVar      float64 `json:"magvar_deg"`
	DistanceKM  float64 `json:"distance_km"`
}

// NDBDetails holds detailed facility data for a specific NDB station.
type NDBDetails struct {
	ICAO        string  `json:"icao"`
	Region      string  `json:"region"`
	Name        string  `json:"name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	AltitudeM   float64 `json:"altitude_m"`
	FrequencyHz uint32  `json:"frequency_hz"`
	FrequencyKHz float64 `json:"frequency_khz"`
	Type        int32   `json:"type"`
	RangeNM     float64 `json:"range_nm"`
	MagVar      float64 `json:"magvar_deg"`
	IsTerminal  bool    `json:"is_terminal"`
}

// WaypointEntry holds basic waypoint info from a facilities list scan.
// DistanceKM is the Haversine distance from the player aircraft's current position.
type WaypointEntry struct {
	ICAO       string  `json:"icao"`
	Region     string  `json:"region"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	AltitudeM  float64 `json:"altitude_m"`
	MagVar     float64 `json:"magvar_deg"`
	DistanceKM float64 `json:"distance_km"`
}

// WaypointDetails holds detailed facility data for a specific waypoint.
type WaypointDetails struct {
	ICAO       string  `json:"icao"`
	Region     string  `json:"region"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	AltitudeM  float64 `json:"altitude_m"`
	Type       int32   `json:"type"`
	MagVar     float64 `json:"magvar_deg"`
	NRoutes    int32   `json:"n_routes"`
	IsTerminal bool    `json:"is_terminal"`
}

// AirportDetails holds detailed facility data for a specific airport.
type AirportDetails struct {
	ICAO        string             `json:"icao"`
	Region      string             `json:"region"`
	Name        string             `json:"name"`
	Name64      string             `json:"name64"`
	Latitude    float64            `json:"latitude"`
	Longitude   float64            `json:"longitude"`
	AltitudeM   float64            `json:"altitude_m"`
	MagVar      float64            `json:"magvar_deg"`
	IsClosed    bool               `json:"is_closed"`
	RunwayCount  int                `json:"runway_count"`
	Runways      []AirportRunway    `json:"runways"`
	Frequencies  []AirportFrequency `json:"frequencies"`
	StandCount   int                `json:"stand_count,omitempty"`
	Stands       []AirportStand     `json:"stands,omitempty"`
	HelipadCount int                `json:"helipad_count,omitempty"`
	Helipads     []AirportHelipad   `json:"helipads,omitempty"`
	ApproachCount  int                `json:"approach_count,omitempty"`
	Approaches     []AirportApproach  `json:"approaches,omitempty"`
	DepartureCount int                `json:"departure_count,omitempty"`
	Departures     []AirportProcedure `json:"departures,omitempty"`
	ArrivalCount   int                `json:"arrival_count,omitempty"`
	Arrivals       []AirportProcedure `json:"arrivals,omitempty"`
}

// TaxiwayPoint is a single node in the airport taxiway network graph.
// BiasX and BiasZ are offsets in metres from the airport reference point
// along the longitudinal and latitudinal axes respectively.
type TaxiwayPoint struct {
	Type        string  `json:"type"`        // NONE/NORMAL/HOLD_SHORT/ILS_HOLD_SHORT/HOLD_SHORT_NO_DRAW/ILS_HOLD_SHORT_NO_DRAW
	Orientation string  `json:"orientation"` // FORWARD/REVERSE (meaningful only for hold-short types)
	BiasXM      float32 `json:"bias_x_m"`
	BiasZM      float32 `json:"bias_z_m"`
}

// TaxiwayPath is a directed edge in the airport taxiway network graph.
// Start and End are indices into the TaxiwayPoints slice of the parent AirportTaxiways.
// NameIndex is an index into the Names slice.
type TaxiwayPath struct {
	Type              string  `json:"type"`               // NONE/TAXI/RUNWAY/PARKING/PATH/CLOSED/VEHICLE/ROAD/PAINTED_LINE
	WidthM            float32 `json:"width_m"`
	LeftHalfWidthM    float32 `json:"left_half_width_m"`
	RightHalfWidthM   float32 `json:"right_half_width_m"`
	WeightLbs         uint32  `json:"weight_lbs"`
	RunwayNumber      int32   `json:"runway_number"`
	RunwayDesignator  string  `json:"runway_designator"`  // L/R/C/W/A/B or empty
	LeftEdge          string  `json:"left_edge"`          // NONE/SOLID/DASHED/SOLID_DASHED
	LeftEdgeLighted   bool    `json:"left_edge_lighted"`
	RightEdge         string  `json:"right_edge"`
	RightEdgeLighted  bool    `json:"right_edge_lighted"`
	CenterLine        bool    `json:"center_line"`
	CenterLineLighted bool    `json:"center_line_lighted"`
	StartNode         int32   `json:"start_node"` // index 0–3999 into Points
	EndNode           int32   `json:"end_node"`   // index 0–3999 into Points
	NameIndex         uint32  `json:"name_index"` // index into Names
}

// AirportTaxiways holds the complete taxiway graph for a specific airport.
// The three slices form a directed graph: Paths are edges that reference
// node indices (StartNode/EndNode into Points) and name indices (into Names).
type AirportTaxiways struct {
	ICAO       string         `json:"icao"`
	NameCount  int            `json:"name_count"`
	PointCount int            `json:"point_count"`
	PathCount  int            `json:"path_count"`
	Names      []string       `json:"names"`  // taxiway letter/name strings, indexed by TaxiwayPath.NameIndex
	Points     []TaxiwayPoint `json:"points"` // graph nodes, indexed by TaxiwayPath.StartNode/EndNode
	Paths      []TaxiwayPath  `json:"paths"`  // graph edges
}

// AirportParking holds the full parking record for a single stand, gate, or ramp spot.
type AirportParking struct {
	Type          string  `json:"type"`            // Ramp GA/Gate Small/Gate Heavy/Vehicle/etc.
	TaxiPointType string  `json:"taxi_point_type"` // NONE/NORMAL/HOLD_SHORT/etc.
	Name          string  `json:"name"`            // NONE/PARKING/N/NE/GATE/GATE_A/etc.
	Suffix        string  `json:"suffix"`          // same enum as Name
	Number        uint32  `json:"number"`
	Orientation   string  `json:"orientation"` // FORWARD/REVERSE
	HeadingDeg    float32 `json:"heading_deg"`
	RadiusM       float32 `json:"radius_m"`
	BiasXM        float32 `json:"bias_x_m"`
	BiasZM        float32 `json:"bias_z_m"`
}

// AirportParkings holds all parking stands, gates, and ramps at an airport.
type AirportParkings struct {
	ICAO         string           `json:"icao"`
	ParkingCount int              `json:"parking_count"`
	Parkings     []AirportParking `json:"parkings"`
}

// Bridge abstracts all SimConnect SDK calls needed by the four MCP tools.
// Implementations must be safe to call from multiple goroutines concurrently.
type Bridge interface {
	// Open starts the background connection and dispatch loop.
	// It does not block; connection establishment is asynchronous.
	// Calling Open on an already-open bridge is a no-op.
	Open(ctx context.Context, appName string) error

	// Close shuts down the dispatch loop and releases the SimConnect handle.
	// Calling Close on a closed bridge is a no-op.
	Close() error

	// State returns the current connection state without blocking.
	State() ConnectionState

	// GetSimVar reads a single simulation variable by name and unit.
	// Returns ErrNotConnected if the bridge is not in StateConnected.
	GetSimVar(ctx context.Context, name, unit string) (SimVar, error)

	// GetSimVars reads up to 20 simulation variables.
	// Each entry in the returned slice corresponds positionally to the input slice.
	// Per-variable errors are embedded in SimVarResult.Error rather than aborting the batch.
	GetSimVars(ctx context.Context, vars []SimVarRequest) ([]SimVarResult, error)

	// TransmitEvent sends a named client event to the simulator.
	// value is the DWORD data attached to the event (0 for events that ignore data).
	TransmitEvent(ctx context.Context, name string, value uint32) error

	// SetSimVar writes a numeric simulation variable to the user aircraft.
	// Only writable SimVars (e.g. autopilot settings) will take effect.
	SetSimVar(ctx context.Context, name, unit string, value float64) error

	// GetSimState returns a snapshot of top-level simulator state.
	GetSimState(ctx context.Context) (SimState, error)

	// GetTraffic returns nearby aircraft within the given radius (metres).
	// The player aircraft is included in the results.
	GetTraffic(ctx context.Context, radiusMeters uint32) ([]TrafficEntry, error)

	// GetEnrichedTraffic returns nearby aircraft with additional velocity-derived
	// fields: vertical_speed_fpm, track_deg, flight_phase, in_parking_state,
	// on_any_runway, and category.  Uses a single RequestDataOnSimObjectType call
	// with an expanded data definition — same latency as GetTraffic.
	GetEnrichedTraffic(ctx context.Context, radiusMeters uint32) ([]EnrichedTrafficEntry, error)

	// GetAirports returns airports in the simulator's reality bubble, sorted by
	// Haversine distance from the player aircraft. DistanceKM is populated on
	// each entry. Returns nil if no airports are found.
	GetAirports(ctx context.Context) ([]AirportEntry, error)

	// GetNearestAirport returns the single closest airport to the player aircraft.
	// Returns nil, nil if no airports are in the reality bubble.
	GetNearestAirport(ctx context.Context) (*AirportEntry, error)

	// GetAirportDetails returns detailed facility data for the given ICAO airport.
	// Pass region="" to match any region. Returns nil if the airport is not found.
	// When expanded=false: base info, runways, and frequencies are returned (3 requests).
	// When expanded=true: stands and helipads are also included (5 requests).
	GetAirportDetails(ctx context.Context, icao, region string, expanded bool) (*AirportDetails, error)

	// GetVORs returns VOR stations in the simulator's reality bubble, sorted by
	// Haversine distance from the player aircraft. Returns nil if none found.
	GetVORs(ctx context.Context) ([]VOREntry, error)

	// GetVORDetails returns detailed facility data for the given VOR ICAO.
	// Pass region="" to match any region. Returns nil if the VOR is not found.
	GetVORDetails(ctx context.Context, icao, region string) (*VORDetails, error)

	// GetNDBs returns NDB stations in the simulator's reality bubble, sorted by
	// Haversine distance from the player aircraft. Returns nil if none found.
	GetNDBs(ctx context.Context) ([]NDBEntry, error)

	// GetNDBDetails returns detailed facility data for the given NDB ICAO.
	// Pass region="" to match any region. Returns nil if the NDB is not found.
	GetNDBDetails(ctx context.Context, icao, region string) (*NDBDetails, error)

	// GetWaypoints returns waypoints in the simulator's reality bubble, sorted by
	// Haversine distance from the player aircraft. Returns nil if none found.
	GetWaypoints(ctx context.Context) ([]WaypointEntry, error)

	// GetWaypointDetails returns detailed facility data for the given waypoint ICAO.
	// Pass region="" to match any region. Returns nil if the waypoint is not found.
	GetWaypointDetails(ctx context.Context, icao, region string) (*WaypointDetails, error)

	// GetAirportTaxiways returns the taxiway network graph (names, paths, and nodes)
	// for the given ICAO airport. Pass region="" to match any region.
	// Returns nil, nil if the airport is not found or has no taxiway data.
	GetAirportTaxiways(ctx context.Context, icao, region string) (*AirportTaxiways, error)

	// GetAirportParkings returns all parking stands, gates, and ramps for the given
	// ICAO airport. Pass region="" to match any region.
	// Returns nil, nil if the airport is not found or has no parking data.
	GetAirportParkings(ctx context.Context, icao, region string) (*AirportParkings, error)

	// GetSimTime returns the current simulator Zulu time in seconds since midnight.
	// Returns 0 if the bridge is not connected. Uses cached manager SimState — no extra request.
	GetSimTime() float64

	// SimEvents returns a read-only channel that receives lifecycle events.
	SimEvents() <-chan SimEvent
}

// Sentinel errors returned by Bridge implementations.
var (
	ErrNotConnected    = errors.New("bridge: not connected to simulator")
	ErrUnknownVariable = errors.New("bridge: unknown simulation variable")
	ErrTimeout         = errors.New("bridge: request timed out")
)
