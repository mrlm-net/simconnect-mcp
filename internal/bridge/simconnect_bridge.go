//go:build windows

// Package bridge — simconnect_bridge.go
// Windows-only implementation of Bridge backed by github.com/mrlm-net/simconnect.
package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/engine"
	"github.com/mrlm-net/simconnect/pkg/manager"
	"github.com/mrlm-net/simconnect/pkg/types"
)

// Compile-time assertion: simconnectBridge must implement Bridge.
var _ Bridge = (*simconnectBridge)(nil)

// simVarDataStruct is the single-field data definition used for each SimVar request.
// SimConnect returns a FLOAT64 value for every numeric variable.
type simVarDataStruct struct {
	Value float64
}

// trafficDataStruct defines the wire-format layout for a traffic scan entry.
// All numeric fields use SIMCONNECT_DATATYPE_FLOAT64 and are placed before the
// string fields to avoid Go struct alignment padding between numeric types.
// Field order must match the AddToDataDefinition calls in GetTraffic exactly.
//
// Layout (240 bytes total, multiple of 8):
//
//	[0–47]   6×float64 — Lat, Lon, Alt, Head, GndSpd, OnGround
//	[48–79]  [32]byte  — ATC ID
//	[80–111] [32]byte  — ATC Airline
//	[112–239][128]byte — Title
type trafficDataStruct struct {
	Lat        float64
	Lon        float64
	Alt        float64
	Head       float64
	GndSpd     float64
	OnGround   float64   // requested as FLOAT64 to avoid int32 alignment gap
	AtcID      [32]byte  // SIMCONNECT_DATATYPE_STRING32
	AtcAirline [32]byte  // SIMCONNECT_DATATYPE_STRING32
	Title      [128]byte // SIMCONNECT_DATATYPE_STRING128
}

// enrichedTrafficDataStruct defines the wire-format layout for an enriched
// traffic scan entry.  All float64 fields are placed before string fields to
// avoid Go struct alignment padding.
//
// Field order must match the AddToDataDefinition calls in GetEnrichedTraffic exactly.
//
// Layout (320 bytes total):
//
//	[0–95]    12×float64 — Lat, Lon, Alt, Head, GndSpd, OnGround,
//	                       VelY, VelX, VelZ, TotalVel, InParking, OnRunway
//	[96–127]  [32]byte   — ATC ID (STRING32)
//	[128–159] [32]byte   — ATC Airline (STRING32)
//	[160–287] [128]byte  — Title (STRING128)
//	[288–319] [32]byte   — Category (STRING32)
type enrichedTrafficDataStruct struct {
	Lat       float64    // PLANE LATITUDE
	Lon       float64    // PLANE LONGITUDE
	Alt       float64    // PLANE ALTITUDE
	Head      float64    // PLANE HEADING DEGREES TRUE
	GndSpd    float64    // GROUND VELOCITY
	OnGround  float64    // SIM ON GROUND (as FLOAT64 to avoid int32 padding)
	VelY      float64    // VELOCITY WORLD Y (fps, vertical — up positive)
	VelX      float64    // VELOCITY WORLD X (fps, east positive)
	VelZ      float64    // VELOCITY WORLD Z (fps, north positive)
	TotalVel  float64    // TOTAL WORLD VELOCITY
	InParking float64    // PLANE IN PARKING STATE
	OnRunway  float64    // ON ANY RUNWAY
	AtcID      [32]byte  // ATC ID        (STRING32)
	AtcAirline [32]byte  // ATC AIRLINE   (STRING32)
	Title      [128]byte // TITLE         (STRING128)
	Category   [32]byte  // CATEGORY      (STRING32)
}

// airportFacilityData mirrors the AddToFacilityDefinition sequence used in
// GetAirportDetails. Field order must match exactly:
//
//	LATITUDE(f64), LONGITUDE(f64), ALTITUDE(f64),
//	ICAO([8]byte), NAME([32]byte), NAME64([64]byte),
//	REGION([8]byte), MAGVAR(f32), IS_CLOSED(int8)
//
// Total wire size: 3×8 + 8 + 32 + 64 + 8 + 4 + 1 = 141 bytes.
type airportFacilityData struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	ICAO      [8]byte
	Name      [32]byte
	Name64    [64]byte
	Region    [8]byte
	MagVar    float32
	IsClosed  int8
}

// runwayFacilityData mirrors the OPEN RUNWAY field sequence:
//
//	HEADING(f32), LENGTH(f32, metres), WIDTH(f32, metres), SURFACE(int32),
//	PRIMARY_NUMBER(int32), PRIMARY_DESIGNATOR(int32),
//	SECONDARY_NUMBER(int32), SECONDARY_DESIGNATOR(int32)
//
// Total wire size: 8×4 = 32 bytes.
type runwayFacilityData struct {
	Heading             float32
	Length              float32 // metres
	Width               float32 // metres
	Surface             int32
	PrimaryNumber       int32
	PrimaryDesignator   int32
	SecondaryNumber     int32
	SecondaryDesignator int32
}

// parkingFacilityData mirrors the OPEN TAXI_PARKING field sequence used in
// GetAirportDetails. Fields are a confirmed subset of the SDK parking dataset.
//
//	NAME(uint32), NUMBER(uint32), HEADING(float32), TYPE(uint32),
//	BIAS_X(float32), BIAS_Z(float32)
//
// Total wire size: 6×4 = 24 bytes.
type parkingFacilityData struct {
	Name    uint32
	Number  uint32
	Heading float32
	Type    uint32
	BiasX   float32
	BiasZ   float32
}

// frequencyFacilityData mirrors the OPEN FREQUENCY field sequence:
//
//	TYPE(int32), FREQUENCY(int32, Hz), NAME([64]byte) = 72 bytes
type frequencyFacilityData struct {
	Type      int32
	Frequency int32
	Name      [64]byte
}

// helipadFacilityData mirrors the OPEN HELIPAD field sequence:
//
//	LATITUDE(f64), LONGITUDE(f64), ALTITUDE(f64),
//	HEADING(f32), LENGTH(f32, metres), WIDTH(f32, metres),
//	SURFACE(int32), TYPE(int32)
//
// Total wire size: 3×8 + 3×4 + 2×4 = 44 bytes.
type helipadFacilityData struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	Heading   float32
	Length    float32 // metres
	Width     float32 // metres
	Surface   int32
	Type      int32
}

// approachFacilityData mirrors the OPEN APPROACH field sequence:
//
//	TYPE(i32), SUFFIX(i32), RUNWAY_NUMBER(i32), RUNWAY_DESIGNATOR(i32),
//	FAF_ICAO([8]byte), FAF_REGION([8]byte),
//	FAF_HEADING(f32), FAF_ALTITUDE(f32), FAF_TYPE(i32), MISSED_ALTITUDE(f32),
//	HAS_LNAV(i32), HAS_LNAVVNAV(i32), HAS_LP(i32), HAS_LPV(i32),
//	N_TRANSITIONS(i32), N_FINAL_APPROACH_LEGS(i32), N_MISSED_APPROACH_LEGS(i32)
//
// Total wire size: 4×4 + 8×2 + 11×4 = 76 bytes.
type approachFacilityData struct {
	Type                int32
	Suffix              int32
	RunwayNumber        int32
	RunwayDesignator    int32
	FAFICAO             [8]byte
	FAFRegion           [8]byte
	FAFHeading          float32
	FAFAltitude         float32
	FAFType             int32
	MissedAltitude      float32
	HasLNAV             int32
	HasLNAVVNAV         int32
	HasLP               int32
	HasLPV              int32
	NTransitions        int32
	NFinalApproachLegs  int32
	NMissedApproachLegs int32
}

// departureFacilityData mirrors the OPEN DEPARTURE / OPEN ARRIVAL field sequence:
//
//	NAME([8]byte), N_RUNWAY_TRANSITIONS(i32), N_ENROUTE_TRANSITIONS(i32), N_APPROACH_LEGS(i32)
//
// Total wire size: 8 + 3×4 = 20 bytes.
type departureFacilityData struct {
	Name                [8]byte
	NRunwayTransitions  int32
	NEnrouteTransitions int32
	NApproachLegs       int32
}

// facilityReqSet groups all request IDs used in a single GetAirportDetails call.
type facilityReqSet struct {
	base, rw, fr, pk, hp, ap, dp, ar uint32
}

// facilityDefSet holds the pre-registered facility definition IDs for the current
// SimConnect connection. A zero base field indicates the definitions have not yet
// been registered (or were reset after a reconnect).
type facilityDefSet struct {
	base, rw, fr, pk, hp, ap, dp, ar uint32
}

// facilityCallState holds per-call state for a GetAirportDetails request.
// handleMessage writes decoded facility data directly into this struct under mu,
// keeping all buffer reads inside the dispatch goroutine (before Release()).
type facilityCallState struct {
	mu        sync.Mutex
	details   *AirportDetails
	ids       facilityReqSet
	foundBase bool
	endIDs    map[uint32]bool
	endCount  int
	expected  int
	done      chan struct{}
	doneOnce  sync.Once
}

// noReq is a sentinel request ID used for slots that are not active in a given call.
// It is 0xFFFFFFFF, well above the valid user ID range (1–999_999_899).
const noReq = ^uint32(0)

// runwayDesignatorLetter maps SimConnect designator int to its letter suffix.
func runwayDesignatorLetter(d int32) string {
	switch d {
	case 1:
		return "L"
	case 2:
		return "R"
	case 3:
		return "C"
	case 4:
		return "W"
	case 5:
		return "A"
	case 6:
		return "B"
	default:
		return ""
	}
}


// helipadTypeName maps SimConnect helipad TYPE integer to a human-readable label.
func helipadTypeName(t int32) string {
	switch t {
	case 1:
		return "H"
	case 2:
		return "Square"
	case 3:
		return "Circle"
	case 4:
		return "Medical"
	default:
		return "None"
	}
}

// runwayName formats a runway designator string like "08L/26R" from SimConnect
// PRIMARY_NUMBER / PRIMARY_DESIGNATOR / SECONDARY_NUMBER / SECONDARY_DESIGNATOR.
func runwayName(priNum, priDes, secNum, secDes int32) string {
	pri := fmt.Sprintf("%02d%s", priNum, runwayDesignatorLetter(priDes))
	sec := fmt.Sprintf("%02d%s", secNum, runwayDesignatorLetter(secDes))
	return pri + "/" + sec
}

// runwaySurfaceName maps SimConnect SURFACE integer to a human-readable label.
func runwaySurfaceName(s int32) string {
	switch s {
	case 0:
		return "Concrete"
	case 1:
		return "Grass"
	case 2:
		return "Water"
	case 3:
		return "Grass Bumpy"
	case 4:
		return "Asphalt"
	case 7:
		return "Clay"
	case 8:
		return "Snow"
	case 9:
		return "Ice"
	case 10:
		return "Dirt"
	case 11:
		return "Coral"
	case 12:
		return "Gravel"
	case 13:
		return "Oil Treated"
	case 14:
		return "Steel Mats"
	case 15:
		return "Bituminous"
	case 16:
		return "Brick"
	case 17:
		return "Macadam"
	case 18:
		return "Planks"
	case 19:
		return "Sand"
	case 20:
		return "Shale"
	case 21:
		return "Tarmac"
	case 22:
		return "Wright Flyer Track"
	default:
		return fmt.Sprintf("Surface(%d)", s)
	}
}

// parkingTypeName maps SimConnect parking TYPE integer to a human-readable label.
func parkingTypeName(t int32) string {
	switch t {
	case 0:
		return "None"
	case 1:
		return "Ramp GA"
	case 2:
		return "Ramp GA Small"
	case 3:
		return "Ramp GA Medium"
	case 4:
		return "Ramp GA Large"
	case 5:
		return "Ramp Cargo"
	case 6:
		return "Ramp Mil Cargo"
	case 7:
		return "Ramp Mil Combat"
	case 8:
		return "Gate Small"
	case 9:
		return "Gate Medium"
	case 10:
		return "Gate Heavy"
	case 11:
		return "Dock GA"
	case 12:
		return "Fuel"
	case 13:
		return "Vehicle"
	case 14:
		return "Ramp GA Extra"
	case 15:
		return "Gate Extra"
	default:
		return fmt.Sprintf("Type(%d)", t)
	}
}

// frequencyTypeName maps SimConnect frequency TYPE integer to a human-readable label.
func frequencyTypeName(t int32) string {
	switch t {
	case 0:
		return "None"
	case 1:
		return "ATIS"
	case 2:
		return "Multicom"
	case 3:
		return "UNICOM"
	case 4:
		return "CTAF"
	case 5:
		return "Ground"
	case 6:
		return "Tower"
	case 7:
		return "Clearance"
	case 8:
		return "Approach"
	case 9:
		return "Departure"
	case 10:
		return "Center"
	case 11:
		return "FSS"
	case 12:
		return "AWOS"
	case 13:
		return "ASOS"
	case 14:
		return "Clearance Pre-Taxi"
	case 15:
		return "Remote Clearance Delivery"
	default:
		return fmt.Sprintf("Freq(%d)", t)
	}
}

// simconnectBridge wraps the mrlm-net/simconnect Manager to satisfy the Bridge
// interface.  Each instance manages one Manager lifecycle.
//
// ID allocation strategy (must not overlap manager's reserved 999_999_900–999_999_999):
//
//	Definition IDs : 1_000_000 + slot*2     (one per in-flight request slot)
//	Request IDs    : 1_000_001 + slot*2
//	Notification group ID : 1
//	Event ID base  : 100_000 (incremented per TransmitEvent call)
//
// GetSimVar / GetSimVars use one-shot PERIOD_ONCE requests correlated via
// request ID.  A pending map protected by a mutex maps each outstanding request
// ID to the channel waiting for the result.  The Manager's OnMessage callback
// delivers the SIMCONNECT_RECV_ID_SIMOBJECT_DATA payload into the appropriate
// channel.
type simconnectBridge struct {
	mu sync.RWMutex

	mgr        manager.Manager
	cancelMgr  context.CancelFunc
	mgrErrCh   chan error // receives error from mgr.Start() goroutine
	simEventCh chan SimEvent

	// pending maps requestID -> channel expecting exactly one float64 response
	pendingMu sync.Mutex
	pending   map[uint32]chan float64

	// facilityDefs holds pre-registered facility definition IDs for the current connection.
	// Reset on reconnect (OnOpen); re-registered lazily on the first GetAirportDetails call.
	// Protected by facilityMu.
	facilityMu   sync.Mutex
	facilityDefs facilityDefSet

	// facilityPending maps requestID -> per-call state for facility data messages.
	// Populated by GetAirportDetails; handleMessage decodes data inline and writes
	// directly into the state struct under state.mu.
	facilityPending map[uint32]*facilityCallState

	// idCounter provides unique definition / request IDs.
	// Each request consumes two IDs: (counter*2) for definition, (counter*2+1) for request.
	// We start from 1 to stay well within IsValidUserID range.
	idCounter atomic.Uint32

	// eventIDCounter allocates unique event IDs for TransmitClientEvent calls.
	// Starts from 100_000; valid range 1–999_999_899.
	eventIDCounter atomic.Uint32

	// flightFile caches the last known flight file path delivered by the FlightLoaded
	// event.  Populated asynchronously; protected by mu.
	flightFile string

	// simVersion caches the simulator application version string captured from
	// the SIMCONNECT_RECV_OPEN packet. Protected by mu.
	simVersion string
}

// NewSimConnectBridge constructs a simconnectBridge.  The bridge is idle until
// Open is called.
func NewSimConnectBridge() Bridge {
	b := &simconnectBridge{
		mgrErrCh:        make(chan error, 1),
		simEventCh:      make(chan SimEvent, 64),
		pending:         make(map[uint32]chan float64),
		facilityPending: make(map[uint32]*facilityCallState),
	}
	b.eventIDCounter.Store(100_000)
	return b
}

// Open starts the background Manager connection and dispatch loop.
// It is non-blocking; connection establishment happens asynchronously.
// Calling Open on an already-open bridge is a no-op.
func (b *simconnectBridge) Open(ctx context.Context, appName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.mgr != nil {
		// Already open — no-op per interface contract.
		return nil
	}

	// Use context.Background() so the manager's lifetime is independent of the
	// caller's context. The caller's context (e.g. a 10-second startup timeout)
	// must not cancel the manager's background goroutines when it expires.
	mgrCtx, cancel := context.WithCancel(context.Background())
	b.cancelMgr = cancel

	// Redirect SDK logs to stderr so they don't corrupt the stdio MCP transport.
	stderrLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	mgr := manager.New(
		appName,
		manager.WithContext(mgrCtx),
		manager.WithAutoReconnect(true),
		manager.WithAutoDetect(),
		manager.WithSimStatePeriod(types.SIMCONNECT_PERIOD_SECOND),
		manager.WithLogger(stderrLogger),
	)
	b.mgr = mgr

	// Wire up the message handler so we can receive SimObject data responses
	// for GetSimVar / GetSimVars.
	mgr.OnMessage(b.handleMessage)

	// Capture simulator application version on connection open.
	// Also reset facility definitions so they are re-registered on the next call
	// (the SimConnect session is new after a reconnect, losing all prior defs).
	mgr.OnOpen(func(data types.ConnectionOpenData) {
		v := fmt.Sprintf("%d.%d.%d.%d",
			data.ApplicationVersionMajor,
			data.ApplicationVersionMinor,
			data.ApplicationBuildMajor,
			data.ApplicationBuildMinor,
		)
		b.mu.Lock()
		b.simVersion = v
		b.mu.Unlock()
		b.facilityMu.Lock()
		b.facilityDefs = facilityDefSet{}
		b.facilityMu.Unlock()
	})

	// Wire up lifecycle events that feed SimEvents().
	mgr.OnFlightLoaded(func(filename string) {
		b.mu.Lock()
		b.flightFile = filename
		b.mu.Unlock()
		b.pushSimEvent(SimEvent{Name: "FlightLoaded", Value: 0})
	})

	mgr.OnCrashed(func() {
		b.pushSimEvent(SimEvent{Name: "Crashed", Value: 0})
	})

	// Pause handler: mrlm delivers paused=true/false
	mgr.OnPause(func(paused bool) {
		name := "Unpaused"
		var val uint32
		if paused {
			name = "Paused"
			val = 1
		}
		b.pushSimEvent(SimEvent{Name: name, Value: val})
	})

	// Start the Manager in the background; it blocks until stopped.
	go func() {
		err := mgr.Start()
		select {
		case b.mgrErrCh <- err:
		default:
		}
	}()

	return nil
}

// Close shuts down the dispatch loop and releases the SimConnect handle.
// Calling Close on a closed bridge is a no-op.
func (b *simconnectBridge) Close() error {
	b.mu.Lock()
	mgr := b.mgr
	cancel := b.cancelMgr
	b.mgr = nil
	b.cancelMgr = nil
	b.mu.Unlock()

	if mgr == nil {
		return nil
	}

	// Drain any waiting requests with a disconnection error so callers unblock.
	b.pendingMu.Lock()
	for id, ch := range b.pending {
		close(ch)
		delete(b.pending, id)
	}
	b.pendingMu.Unlock()

	b.facilityMu.Lock()
	for id, state := range b.facilityPending {
		state.doneOnce.Do(func() { close(state.done) })
		delete(b.facilityPending, id)
	}
	b.facilityMu.Unlock()

	if err := mgr.Stop(); err != nil {
		cancel()
		return fmt.Errorf("bridge: %w", err)
	}
	cancel()
	return nil
}

// State returns the current connection state without blocking.
func (b *simconnectBridge) State() ConnectionState {
	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()

	if mgr == nil {
		return StateDisconnected
	}

	switch mgr.ConnectionState() {
	case manager.StateAvailable:
		return StateConnected
	case manager.StateConnected:
		// Manager's StateConnected means TCP/pipe is open but RECV_OPEN not yet
		// received.  Map to our StateConnecting.
		return StateConnecting
	case manager.StateConnecting, manager.StateReconnecting:
		return StateConnecting
	default:
		return StateDisconnected
	}
}

// approachTypeName converts a SimConnect approach type int to a string label.
func approachTypeName(t int32) string {
	switch t {
	case 1:
		return "GPS"
	case 2:
		return "VOR"
	case 3:
		return "NDB"
	case 4:
		return "ILS"
	case 5:
		return "LOCALIZER"
	case 6:
		return "SDF"
	case 7:
		return "LDA"
	case 8:
		return "VORDME"
	case 9:
		return "NDBDME"
	case 10:
		return "RNAV"
	case 11:
		return "LOCALIZER_BACK_COURSE"
	default:
		return "UNKNOWN"
	}
}

// approachRunwayName builds a runway string (e.g. "08L") from SimConnect approach fields.
// Returns an empty string for circling approaches (runway number 0).
func approachRunwayName(number, designator int32) string {
	if number == 0 {
		return ""
	}
	return fmt.Sprintf("%02d%s", number, runwayDesignatorLetter(designator))
}

// GetSimVar reads a single simulation variable by name and unit.
// It registers a one-shot data definition, requests a single frame of data,
// and blocks until the response arrives or the context is cancelled.
func (b *simconnectBridge) GetSimVar(ctx context.Context, name, unit string) (SimVar, error) {
	if b.State() != StateConnected {
		return SimVar{}, ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return SimVar{}, ErrNotConnected
	}

	defID, reqID := b.allocIDs()

	// Register one float64 datum on this definition.
	if err := mgr.AddToDataDefinition(defID, name, unit, types.SIMCONNECT_DATATYPE_FLOAT64, 0, 0); err != nil {
		return SimVar{}, fmt.Errorf("bridge: AddToDataDefinition %s: %w", name, err)
	}

	// Create a buffered channel and register it before making the request so we
	// never miss the response.
	resultCh := make(chan float64, 1)
	b.pendingMu.Lock()
	b.pending[reqID] = resultCh
	b.pendingMu.Unlock()

	// Request a single snapshot (PERIOD_ONCE delivers exactly one packet).
	if err := mgr.RequestDataOnSimObject(
		reqID, defID,
		types.SIMCONNECT_OBJECT_ID_USER,
		types.SIMCONNECT_PERIOD_ONCE,
		types.SIMCONNECT_DATA_REQUEST_FLAG_DEFAULT,
		0, 0, 0,
	); err != nil {
		b.pendingMu.Lock()
		delete(b.pending, reqID)
		b.pendingMu.Unlock()
		_ = mgr.ClearDataDefinition(defID)
		return SimVar{}, fmt.Errorf("bridge: RequestDataOnSimObject %s: %w", name, err)
	}

	// Wait for the response or context cancellation.
	var value float64
	var ok bool
	select {
	case <-ctx.Done():
		b.pendingMu.Lock()
		delete(b.pending, reqID)
		b.pendingMu.Unlock()
		_ = mgr.ClearDataDefinition(defID)
		return SimVar{}, fmt.Errorf("bridge: %w", ErrTimeout)
	case value, ok = <-resultCh:
		if !ok {
			// Channel was closed by Close() — bridge shutting down.
			_ = mgr.ClearDataDefinition(defID)
			return SimVar{}, ErrNotConnected
		}
	}

	// Clean up the data definition.
	_ = mgr.ClearDataDefinition(defID)

	return SimVar{Name: name, Value: value, Unit: unit}, nil
}

// GetSimVars reads up to 20 simulation variables in a sequential batch.
// Per-variable errors are embedded in SimVarResult.Error rather than aborting
// the batch.
func (b *simconnectBridge) GetSimVars(ctx context.Context, vars []SimVarRequest) ([]SimVarResult, error) {
	results := make([]SimVarResult, len(vars))
	for i, v := range vars {
		sv, err := b.GetSimVar(ctx, v.Name, v.Unit)
		results[i] = SimVarResult{
			Name:  v.Name,
			Unit:  v.Unit,
			Value: sv.Value,
		}
		if err != nil {
			results[i].Error = err.Error()
		}
		// Bail out if the caller's context was cancelled.
		if ctx.Err() != nil {
			break
		}
	}
	return results, nil
}

// TransmitEvent sends a named client event to the simulator.
// It maps the event name to a fresh event ID, adds it to a notification group,
// and transmits it to the user object.
func (b *simconnectBridge) TransmitEvent(ctx context.Context, name string, value uint32) error {
	if b.State() != StateConnected {
		return ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return ErrNotConnected
	}

	// Allocate a fresh event ID for this named event.
	// Group ID 3000 with priority 1000 and DEFAULT flag matches the SDK examples.
	eventID := b.eventIDCounter.Add(1)
	const groupID uint32 = 3000

	if err := mgr.MapClientEventToSimEvent(eventID, name); err != nil {
		return fmt.Errorf("bridge: MapClientEventToSimEvent %s: %w", name, err)
	}
	if err := mgr.AddClientEventToNotificationGroup(groupID, eventID, false); err != nil {
		return fmt.Errorf("bridge: AddClientEventToNotificationGroup %s: %w", name, err)
	}
	if err := mgr.SetNotificationGroupPriority(groupID, 1000); err != nil {
		return fmt.Errorf("bridge: SetNotificationGroupPriority: %w", err)
	}

	if err := mgr.TransmitClientEvent(
		types.SIMCONNECT_OBJECT_ID_USER,
		eventID,
		value,
		groupID,
		types.SIMCONNECT_EVENT_FLAG_DEFAULT,
	); err != nil {
		return fmt.Errorf("bridge: TransmitClientEvent %s: %w", name, err)
	}

	// Clean up — remove the event from the notification group so its ID can be
	// logically reused without accumulating stale group entries.
	_ = mgr.RemoveClientEvent(groupID, eventID)
	return nil
}

// SetSimVar writes a numeric simulation variable to the user aircraft.
// It registers a one-shot data definition, pushes the value via SetDataOnSimObject,
// and cleans up the definition. Only writable SimVars will take effect in the sim.
func (b *simconnectBridge) SetSimVar(ctx context.Context, name, unit string, value float64) error {
	if b.State() != StateConnected {
		return ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return ErrNotConnected
	}

	defID, _ := b.allocIDs()

	if err := mgr.AddToDataDefinition(defID, name, unit, types.SIMCONNECT_DATATYPE_FLOAT64, 0, 0); err != nil {
		return fmt.Errorf("bridge: AddToDataDefinition %s: %w", name, err)
	}
	defer func() { _ = mgr.ClearDataDefinition(defID) }()

	if err := mgr.SetDataOnSimObject(
		defID,
		types.SIMCONNECT_OBJECT_ID_USER,
		types.SIMCONNECT_DATA_SET_FLAG_DEFAULT,
		0,
		uint32(unsafe.Sizeof(value)),
		unsafe.Pointer(&value),
	); err != nil {
		return fmt.Errorf("bridge: SetDataOnSimObject %s: %w", name, err)
	}
	return nil
}

// GetSimState returns a snapshot of top-level simulator state.
// It reads from the Manager's continuously-updated SimState and populates the
// Bridge's SimState fields.
//
// Mapping notes:
//   - Connected: derived from bridge.State()
//   - Paused:    manager.SimState().Paused
//   - CurrentFlight: last FlightLoaded filename cached in b.flightFile
//   - SimTime:   manager.SimState().ZuluTime (seconds since midnight Zulu)
//   - SimulatorVersion: captured from ConnectionOpen data via OnOpen handler.
//   - Position/speed fields: read from manager.SimState() (SDK v0.4.2 fixes
//     the alignment bug that produced garbage values in earlier versions).
//     Note: position fields may be zero for ~1 second after connect until the
//     first 1Hz SimState update arrives from the simulator.
func (b *simconnectBridge) GetSimState(ctx context.Context) (SimState, error) {
	b.mu.RLock()
	mgr := b.mgr
	flightFile := b.flightFile
	simVersion := b.simVersion
	b.mu.RUnlock()

	connected := b.State() == StateConnected

	if mgr == nil || !connected {
		return SimState{Connected: false}, nil
	}

	mgrState := mgr.SimState()

	return SimState{
		Connected:         connected,
		Paused:            mgrState.Paused,
		CurrentFlight:     flightFile,
		SimTime:           mgrState.ZuluTime,
		SimulatorVersion:  simVersion,
		Latitude:          mgrState.Latitude,
		Longitude:         mgrState.Longitude,
		Altitude:          mgrState.Altitude,
		GroundSpeed:       mgrState.GroundSpeed,
		IndicatedAirspeed: mgrState.IndicatedAirspeed,
		VerticalSpeed:     mgrState.VerticalSpeed * 60, // SDK returns fps; convert to fpm
		TrueHeading:       mgrState.TrueHeading,
		OnGround:          mgrState.SimOnGround,
	}, nil
}

// GetTraffic returns nearby aircraft within radiusMeters of the user aircraft.
// It issues a one-shot RequestDataOnSimObjectType call and collects all
// SIMOBJECT_DATA_BYTYPE responses until the batch is complete or the context
// is cancelled.  The player aircraft is included in the results.
func (b *simconnectBridge) GetTraffic(ctx context.Context, radiusMeters uint32) ([]TrafficEntry, error) {
	if b.State() != StateConnected {
		return nil, ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return nil, ErrNotConnected
	}

	defID, reqID := b.allocIDs()

	// Register the data definition.  Field order mirrors trafficDataStruct.
	defs := []struct {
		name, unit string
		typ        types.SIMCONNECT_DATATYPE
	}{
		{"PLANE LATITUDE", "degrees", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE LONGITUDE", "degrees", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE ALTITUDE", "feet", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE HEADING DEGREES TRUE", "degrees", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"GROUND VELOCITY", "knots", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"SIM ON GROUND", "bool", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"ATC ID", "", types.SIMCONNECT_DATATYPE_STRING32},
		{"ATC AIRLINE", "", types.SIMCONNECT_DATATYPE_STRING32},
		{"TITLE", "", types.SIMCONNECT_DATATYPE_STRING128},
	}
	for i, d := range defs {
		if err := mgr.AddToDataDefinition(defID, d.name, d.unit, d.typ, 0, uint32(i)); err != nil {
			_ = mgr.ClearDataDefinition(defID)
			return nil, fmt.Errorf("bridge: AddToDataDefinition %s: %w", d.name, err)
		}
	}
	defer func() { _ = mgr.ClearDataDefinition(defID) }()

	// Subscribe to BYTYPE messages only, so we don't interfere with the main
	// OnMessage handler that serves GetSimVar requests.
	sub := mgr.SubscribeWithType("", 128, []types.SIMCONNECT_RECV_ID{
		types.SIMCONNECT_RECV_ID_SIMOBJECT_DATA_BYTYPE,
	})
	defer sub.Unsubscribe()

	if err := mgr.RequestDataOnSimObjectType(reqID, defID, radiusMeters, types.SIMCONNECT_SIMOBJECT_TYPE_AIRCRAFT); err != nil {
		return nil, fmt.Errorf("bridge: RequestDataOnSimObjectType: %w", err)
	}

	var results []TrafficEntry
	var expected uint32

	// 3-second timeout — if no response arrives, assume no traffic.
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return results, nil
		case <-deadline.C:
			return results, nil
		case msg, ok := <-sub.Messages():
			if !ok {
				return results, nil
			}
			simObjData := msg.AsSimObjectDataBType()
			if simObjData == nil || uint32(simObjData.DwRequestID) != reqID {
				continue
			}
			if simObjData.DwOutOf == 0 {
				// No aircraft in range.
				return nil, nil
			}
			if expected == 0 {
				expected = uint32(simObjData.DwOutOf)
				results = make([]TrafficEntry, 0, expected)
			}
			data := engine.CastDataAs[trafficDataStruct](&simObjData.DwData)
			results = append(results, TrafficEntry{
				ObjectID:    uint32(simObjData.DwObjectID),
				Title:       engine.BytesToString(data.Title[:]),
				ATCID:       engine.BytesToString(data.AtcID[:]),
				ATCAirline:  engine.BytesToString(data.AtcAirline[:]),
				Latitude:    data.Lat,
				Longitude:   data.Lon,
				AltitudeFt:  data.Alt,
				TrueHeading: data.Head,
				GroundSpeed: data.GndSpd,
				OnGround:    data.OnGround != 0,
			})
			if uint32(len(results)) >= expected {
				return results, nil
			}
		}
	}
}

// GetEnrichedTraffic returns nearby aircraft with velocity-derived fields.
// It issues a single RequestDataOnSimObjectType call with an expanded data
// definition that includes VELOCITY WORLD Y/X/Z, TOTAL WORLD VELOCITY,
// PLANE IN PARKING STATE, ON ANY RUNWAY, and CATEGORY in addition to the
// fields returned by GetTraffic.  Track bearing and flight phase are derived
// from the velocity vectors before returning.
func (b *simconnectBridge) GetEnrichedTraffic(ctx context.Context, radiusMeters uint32) ([]EnrichedTrafficEntry, error) {
	if b.State() != StateConnected {
		return nil, ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return nil, ErrNotConnected
	}

	defID, reqID := b.allocIDs()

	// Field order must mirror enrichedTrafficDataStruct exactly.
	defs := []struct {
		name, unit string
		typ        types.SIMCONNECT_DATATYPE
	}{
		{"PLANE LATITUDE", "degrees", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE LONGITUDE", "degrees", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE ALTITUDE", "feet", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE HEADING DEGREES TRUE", "degrees", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"GROUND VELOCITY", "knots", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"SIM ON GROUND", "bool", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"VELOCITY WORLD Y", "feet per second", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"VELOCITY WORLD X", "feet per second", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"VELOCITY WORLD Z", "feet per second", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"TOTAL WORLD VELOCITY", "feet per second", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"PLANE IN PARKING STATE", "bool", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"ON ANY RUNWAY", "bool", types.SIMCONNECT_DATATYPE_FLOAT64},
		{"ATC ID", "", types.SIMCONNECT_DATATYPE_STRING32},
		{"ATC AIRLINE", "", types.SIMCONNECT_DATATYPE_STRING32},
		{"TITLE", "", types.SIMCONNECT_DATATYPE_STRING128},
		{"CATEGORY", "", types.SIMCONNECT_DATATYPE_STRING32},
	}
	for i, d := range defs {
		if err := mgr.AddToDataDefinition(defID, d.name, d.unit, d.typ, 0, uint32(i)); err != nil {
			_ = mgr.ClearDataDefinition(defID)
			return nil, fmt.Errorf("bridge: AddToDataDefinition %s: %w", d.name, err)
		}
	}
	defer func() { _ = mgr.ClearDataDefinition(defID) }()

	sub := mgr.SubscribeWithType("", 128, []types.SIMCONNECT_RECV_ID{
		types.SIMCONNECT_RECV_ID_SIMOBJECT_DATA_BYTYPE,
	})
	defer sub.Unsubscribe()

	if err := mgr.RequestDataOnSimObjectType(reqID, defID, radiusMeters, types.SIMCONNECT_SIMOBJECT_TYPE_AIRCRAFT); err != nil {
		return nil, fmt.Errorf("bridge: RequestDataOnSimObjectType: %w", err)
	}

	var results []EnrichedTrafficEntry
	var expected uint32

	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return results, nil
		case <-deadline.C:
			return results, nil
		case msg, ok := <-sub.Messages():
			if !ok {
				return results, nil
			}
			simObjData := msg.AsSimObjectDataBType()
			if simObjData == nil || uint32(simObjData.DwRequestID) != reqID {
				continue
			}
			if simObjData.DwOutOf == 0 {
				return nil, nil
			}
			if expected == 0 {
				expected = uint32(simObjData.DwOutOf)
				results = make([]EnrichedTrafficEntry, 0, expected)
			}
			data := engine.CastDataAs[enrichedTrafficDataStruct](&simObjData.DwData)
			onGround := data.OnGround != 0
			vsFPM := data.VelY * 60.0
			results = append(results, EnrichedTrafficEntry{
				ObjectID:         uint32(simObjData.DwObjectID),
				Title:            engine.BytesToString(data.Title[:]),
				ATCID:            engine.BytesToString(data.AtcID[:]),
				ATCAirline:       engine.BytesToString(data.AtcAirline[:]),
				Category:         engine.BytesToString(data.Category[:]),
				Latitude:         data.Lat,
				Longitude:        data.Lon,
				AltitudeFt:       data.Alt,
				TrueHeading:      data.Head,
				TrackDeg:         computeTrackDeg(data.VelX, data.VelZ),
				GroundSpeed:      data.GndSpd,
				VerticalSpeedFPM: vsFPM,
				OnGround:         onGround,
				InParkingState:   data.InParking != 0,
				OnAnyRunway:      data.OnRunway != 0,
				FlightPhase:      computeFlightPhase(onGround, data.GndSpd, vsFPM),
			})
			if uint32(len(results)) >= expected {
				return results, nil
			}
		}
	}
}

// GetAirports returns all airports in the simulator's reality bubble, sorted
// by Haversine distance from the player aircraft.  It issues a one-shot
// RequestFacilitiesListEX1 call and collects AIRPORT_LIST response packets
// until all batches have arrived or the context is cancelled.
//
// Airport entry size differs between MSFS 2020 (33 bytes) and MSFS 2024 (36/40/41
// bytes); the implementation derives field offsets at runtime from the message
// stride, matching the read-facilities SDK example.
func (b *simconnectBridge) GetAirports(ctx context.Context) ([]AirportEntry, error) {
	if b.State() != StateConnected {
		return nil, ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return nil, ErrNotConnected
	}

	// Get player position for distance computation.
	state := mgr.SimState()
	playerLat := state.Latitude
	playerLon := state.Longitude

	listID, _ := b.allocIDs()

	sub := mgr.SubscribeWithType("", 512, []types.SIMCONNECT_RECV_ID{
		types.SIMCONNECT_RECV_ID_AIRPORT_LIST,
	})
	defer sub.Unsubscribe()

	if err := mgr.RequestFacilitiesListEX1(listID, types.SIMCONNECT_FACILITY_LIST_AIRPORT); err != nil {
		return nil, fmt.Errorf("bridge: RequestFacilitiesListEX1: %w", err)
	}

	var airports []AirportEntry

	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return airports, nil
		case <-deadline.C:
			return airports, nil
		case msg, ok := <-sub.Messages():
			if !ok {
				return airports, nil
			}
			list := msg.AsAirportList()
			if list == nil || uint32(list.DwRequestID) != listID {
				continue
			}

			if list.DwArraySize > 0 {
				// Derive field offsets from the actual wire stride.
				// MSFS 2020: ident[6] + region[3] + 3×float64 = 33 bytes
				// MSFS 2024: ident[9] + region[3] + 3×float64 = 36 bytes (±trailing pad)
				headerSize := unsafe.Sizeof(types.SIMCONNECT_RECV_FACILITIES_LIST{})
				actualDataSize := uintptr(msg.DwSize) - headerSize
				actualEntrySize := actualDataSize / uintptr(list.DwArraySize)

				var identLen, latOff, lonOff, altOff uintptr
				switch actualEntrySize {
				case 33:
					identLen, latOff, lonOff, altOff = 6, 9, 17, 25
				case 36, 40, 41:
					identLen, latOff, lonOff, altOff = 9, 12, 20, 28
				}

				if identLen > 0 {
					dataStart := unsafe.Pointer(uintptr(unsafe.Pointer(list)) + headerSize)
					for i := uint32(0); i < uint32(list.DwArraySize); i++ {
						entryPtr := unsafe.Pointer(uintptr(dataStart) + uintptr(i)*actualEntrySize)
						identBytes := (*[9]byte)(entryPtr)
						regionBytes := (*[3]byte)(unsafe.Pointer(uintptr(entryPtr) + identLen))
						lat := *(*float64)(unsafe.Pointer(uintptr(entryPtr) + latOff))
						lon := *(*float64)(unsafe.Pointer(uintptr(entryPtr) + lonOff))
						alt := *(*float64)(unsafe.Pointer(uintptr(entryPtr) + altOff))
						airports = append(airports, AirportEntry{
							ICAO:       engine.BytesToString(identBytes[:identLen]),
							Region:     engine.BytesToString(regionBytes[:]),
							Latitude:   lat,
							Longitude:  lon,
							AltitudeM:  alt,
							DistanceKM: calc.HaversineKM(playerLat, playerLon, lat, lon),
						})
					}
				}
			}

			// Done when this is the last (or only) packet.
			if uint32(list.DwEntryNumber)+1 >= uint32(list.DwOutOf) {
				// Deduplicate by ICAO — the list can contain the same
				// airport multiple times under the same identifier.
				seen := make(map[string]struct{}, len(airports))
				unique := airports[:0]
				for _, a := range airports {
					if _, ok := seen[a.ICAO]; !ok {
						seen[a.ICAO] = struct{}{}
						unique = append(unique, a)
					}
				}
				airports = unique

				sort.Slice(airports, func(i, j int) bool {
					return airports[i].DistanceKM < airports[j].DistanceKM
				})
				return airports, nil
			}
		}
	}
}

// GetNearestAirport returns the closest airport to the player aircraft.
// It calls GetAirports and returns the first element (shortest distance).
func (b *simconnectBridge) GetNearestAirport(ctx context.Context) (*AirportEntry, error) {
	airports, err := b.GetAirports(ctx)
	if err != nil {
		return nil, err
	}
	if len(airports) == 0 {
		return nil, nil
	}
	entry := airports[0]
	return &entry, nil
}

// ensureFacilityDefs registers the eight facility definition schemas once per
// SimConnect connection and stores them in b.facilityDefs for reuse by all
// subsequent GetAirportDetails calls — eliminating the per-call
// AddToFacilityDefinition overhead. The definitions are reset on reconnect
// (see OnOpen callback) and re-registered lazily on the next call.
//
// The facilityMu lock is held for the entire registration to prevent concurrent
// double-registration. AddToFacilityDefinition is a non-blocking command-queue
// call, so holding the lock briefly is safe.
func (b *simconnectBridge) ensureFacilityDefs(mgr manager.Manager) (facilityDefSet, error) {
	b.facilityMu.Lock()
	defer b.facilityMu.Unlock()

	if b.facilityDefs.base != 0 {
		return b.facilityDefs, nil
	}

	baseDefID, _ := b.allocIDs()
	rwDefID, _ := b.allocIDs()
	frDefID, _ := b.allocIDs()
	pkDefID, _ := b.allocIDs()
	hpDefID, _ := b.allocIDs()
	apDefID, _ := b.allocIDs()
	dpDefID, _ := b.allocIDs()
	arDefID, _ := b.allocIDs()

	for _, f := range []string{
		"OPEN AIRPORT", "LATITUDE", "LONGITUDE", "ALTITUDE", "ICAO", "NAME", "NAME64",
		"REGION", "MAGVAR", "IS_CLOSED",
		"CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(baseDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition base %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN RUNWAY",
		"HEADING", "LENGTH", "WIDTH", "SURFACE",
		"PRIMARY_NUMBER", "PRIMARY_DESIGNATOR",
		"SECONDARY_NUMBER", "SECONDARY_DESIGNATOR",
		"CLOSE RUNWAY", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(rwDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition runway %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN FREQUENCY",
		"TYPE", "FREQUENCY", "NAME",
		"CLOSE FREQUENCY", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(frDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition frequency %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN TAXI_PARKING",
		"NAME", "NUMBER", "HEADING", "TYPE", "BIAS_X", "BIAS_Z",
		"CLOSE TAXI_PARKING", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(pkDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition parking %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN HELIPAD",
		"LATITUDE", "LONGITUDE", "ALTITUDE", "HEADING", "LENGTH", "WIDTH", "SURFACE", "TYPE",
		"CLOSE HELIPAD", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(hpDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition helipad %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN APPROACH",
		"TYPE", "SUFFIX", "RUNWAY_NUMBER", "RUNWAY_DESIGNATOR",
		"FAF_ICAO", "FAF_REGION", "FAF_HEADING", "FAF_ALTITUDE", "FAF_TYPE", "MISSED_ALTITUDE",
		"HAS_LNAV", "HAS_LNAVVNAV", "HAS_LP", "HAS_LPV",
		"N_TRANSITIONS", "N_FINAL_APPROACH_LEGS", "N_MISSED_APPROACH_LEGS",
		"CLOSE APPROACH", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(apDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition approach %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN DEPARTURE",
		"NAME", "N_RUNWAY_TRANSITIONS", "N_ENROUTE_TRANSITIONS", "N_APPROACH_LEGS",
		"CLOSE DEPARTURE", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(dpDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition departure %q: %w", f, err)
		}
	}

	for _, f := range []string{
		"OPEN AIRPORT", "OPEN ARRIVAL",
		"NAME", "N_RUNWAY_TRANSITIONS", "N_ENROUTE_TRANSITIONS", "N_APPROACH_LEGS",
		"CLOSE ARRIVAL", "CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(arDefID, f); err != nil {
			return facilityDefSet{}, fmt.Errorf("bridge: AddToFacilityDefinition arrival %q: %w", f, err)
		}
	}

	b.facilityDefs = facilityDefSet{
		base: baseDefID, rw: rwDefID, fr: frDefID,
		pk: pkDefID, hp: hpDefID, ap: apDefID, dp: dpDefID, ar: arDefID,
	}
	return b.facilityDefs, nil
}

// GetAirportDetails returns detailed facility data for the given ICAO airport.
// Issues two RequestFacilityData calls (basic airport info + runways) and waits
// until both FACILITY_DATA_END messages are received.
func (b *simconnectBridge) GetAirportDetails(ctx context.Context, icao, region string, expanded bool) (*AirportDetails, error) {
	if b.State() != StateConnected {
		return nil, ErrNotConnected
	}

	b.mu.RLock()
	mgr := b.mgr
	b.mu.RUnlock()
	if mgr == nil {
		return nil, ErrNotConnected
	}

	// Ensure facility definitions are registered once for this connection.
	// On first call this registers 8 definitions; subsequent calls return cached IDs.
	defs, err := b.ensureFacilityDefs(mgr)
	if err != nil {
		return nil, err
	}

	// Allocate per-call request IDs (definition IDs are shared / pre-registered).
	_, baseReqID := b.allocIDs()
	_, rwReqID := b.allocIDs()
	_, frReqID := b.allocIDs()

	var pkReqID, hpReqID, apReqID, dpReqID, arReqID uint32
	if expanded {
		_, pkReqID = b.allocIDs()
		_, hpReqID = b.allocIDs()
		_, apReqID = b.allocIDs()
		_, dpReqID = b.allocIDs()
		_, arReqID = b.allocIDs()
	}

	ids := facilityReqSet{
		base: baseReqID, rw: rwReqID, fr: frReqID,
		pk: noReq, hp: noReq, ap: noReq, dp: noReq, ar: noReq,
	}
	if expanded {
		ids.pk = pkReqID
		ids.hp = hpReqID
		ids.ap = apReqID
		ids.dp = dpReqID
		ids.ar = arReqID
	}

	details := &AirportDetails{
		Region:      region,
		Runways:     []AirportRunway{},
		Frequencies: []AirportFrequency{},
	}
	if expanded {
		details.Stands = []AirportStand{}
		details.Helipads = []AirportHelipad{}
		details.Approaches = []AirportApproach{}
		details.Departures = []AirportProcedure{}
		details.Arrivals = []AirportProcedure{}
	}

	endIDs := map[uint32]bool{baseReqID: false, rwReqID: false, frReqID: false}
	if expanded {
		endIDs[pkReqID] = false
		endIDs[hpReqID] = false
		endIDs[apReqID] = false
		endIDs[dpReqID] = false
		endIDs[arReqID] = false
	}
	expectedEnds := 3
	if expanded {
		expectedEnds = 8
	}

	// Create per-call state. handleMessage decodes FACILITY_DATA records directly
	// into state.details under state.mu — all buffer reads happen synchronously
	// inside the dispatch goroutine, before the SDK pool buffer is released.
	state := &facilityCallState{
		details:  details,
		ids:      ids,
		endIDs:   endIDs,
		expected: expectedEnds,
		done:     make(chan struct{}),
	}

	// Register all active request IDs so handleMessage routes to this call.
	b.facilityMu.Lock()
	b.facilityPending[baseReqID] = state
	b.facilityPending[rwReqID] = state
	b.facilityPending[frReqID] = state
	if expanded {
		b.facilityPending[pkReqID] = state
		b.facilityPending[hpReqID] = state
		b.facilityPending[apReqID] = state
		b.facilityPending[dpReqID] = state
		b.facilityPending[arReqID] = state
	}
	b.facilityMu.Unlock()

	defer func() {
		b.facilityMu.Lock()
		delete(b.facilityPending, baseReqID)
		delete(b.facilityPending, rwReqID)
		delete(b.facilityPending, frReqID)
		if expanded {
			delete(b.facilityPending, pkReqID)
			delete(b.facilityPending, hpReqID)
			delete(b.facilityPending, apReqID)
			delete(b.facilityPending, dpReqID)
			delete(b.facilityPending, arReqID)
		}
		b.facilityMu.Unlock()
	}()

	// Fire requests using the pre-registered definition IDs.
	for _, pair := range [][2]uint32{{defs.base, baseReqID}, {defs.rw, rwReqID}, {defs.fr, frReqID}} {
		if err := mgr.RequestFacilityData(pair[0], pair[1], icao, region); err != nil {
			return nil, fmt.Errorf("bridge: RequestFacilityData %s: %w", icao, err)
		}
	}
	if expanded {
		for _, pair := range [][2]uint32{
			{defs.pk, pkReqID}, {defs.hp, hpReqID},
			{defs.ap, apReqID}, {defs.dp, dpReqID}, {defs.ar, arReqID},
		} {
			if err := mgr.RequestFacilityData(pair[0], pair[1], icao, region); err != nil {
				return nil, fmt.Errorf("bridge: RequestFacilityData %s: %w", icao, err)
			}
		}
	}

	// Wait for all FACILITY_DATA_END messages. handleMessage closes state.done
	// once all expected ENDs are received.
	deadline := time.NewTimer(45 * time.Second)
	defer deadline.Stop()

	select {
	case <-state.done:
		// All expected ENDs received. Wait a short window for late DATA records —
		// SimConnect dispatches DATA and END via different internal paths, so a few
		// DATA records may arrive after all ENDs have been seen.
		time.Sleep(200 * time.Millisecond)
	case <-ctx.Done():
		// Context cancelled — return whatever has been collected so far.
	case <-deadline.C:
		// 45-second hard deadline.
	}

	// Read results under lock — handleMessage may still be writing during the
	// 200ms drain window and we need a consistent snapshot.
	state.mu.Lock()
	deduplicateDetails(state.details)
	foundBase := state.foundBase
	state.mu.Unlock()

	if !foundBase {
		return nil, nil
	}
	return state.details, nil
}

// deduplicateDetails removes duplicate runway, stand, and frequency entries that
// SimConnect can return when an airport scenery database has repeated records.
func deduplicateDetails(d *AirportDetails) {
	seen := make(map[string]struct{})
	unique := d.Runways[:0]
	for _, r := range d.Runways {
		k := fmt.Sprintf("%s|%.2f|%.2f|%s", r.Name, r.LengthM, r.WidthM, r.Surface)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			unique = append(unique, r)
		}
	}
	d.Runways = unique
	d.RunwayCount = len(d.Runways)

	if d.Stands != nil {
		seen = make(map[string]struct{})
		uniqS := d.Stands[:0]
		for _, s := range d.Stands {
			k := fmt.Sprintf("%d|%s|%.2f", s.Number, s.Type, s.Heading)
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				uniqS = append(uniqS, s)
			}
		}
		d.Stands = uniqS
		d.StandCount = len(d.Stands)
	}

	seen = make(map[string]struct{})
	uniqF := d.Frequencies[:0]
	for _, f := range d.Frequencies {
		k := fmt.Sprintf("%s|%.6f", f.Type, f.FreqMHz)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			uniqF = append(uniqF, f)
		}
	}
	d.Frequencies = uniqF

	if d.Helipads != nil {
		seen = make(map[string]struct{})
		uniqH := d.Helipads[:0]
		for _, h := range d.Helipads {
			k := fmt.Sprintf("%.6f|%.6f", h.Latitude, h.Longitude)
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				uniqH = append(uniqH, h)
			}
		}
		d.Helipads = uniqH
		d.HelipadCount = len(d.Helipads)
	}

	if d.Approaches != nil {
		seen = make(map[string]struct{})
		uniqAp := d.Approaches[:0]
		for _, a := range d.Approaches {
			k := fmt.Sprintf("%s|%s", a.Type, a.Runway)
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				uniqAp = append(uniqAp, a)
			}
		}
		d.Approaches = uniqAp
		d.ApproachCount = len(d.Approaches)
	}

	if d.Departures != nil {
		seen = make(map[string]struct{})
		uniqDp := d.Departures[:0]
		for _, dep := range d.Departures {
			if _, ok := seen[dep.Name]; !ok {
				seen[dep.Name] = struct{}{}
				uniqDp = append(uniqDp, dep)
			}
		}
		d.Departures = uniqDp
		d.DepartureCount = len(d.Departures)
	}

	if d.Arrivals != nil {
		seen = make(map[string]struct{})
		uniqAr := d.Arrivals[:0]
		for _, arr := range d.Arrivals {
			if _, ok := seen[arr.Name]; !ok {
				seen[arr.Name] = struct{}{}
				uniqAr = append(uniqAr, arr)
			}
		}
		d.Arrivals = uniqAr
		d.ArrivalCount = len(d.Arrivals)
	}
}

// applyFacilityData decodes a single SIMCONNECT_RECV_ID_FACILITY_DATA message
// and appends the result to the appropriate slice in details.
func applyFacilityData(
	msg engine.Message,
	details *AirportDetails,
	ids facilityReqSet,
	foundBase *bool,
) {
	fd := msg.AsFacilityData()
	if fd == nil {
		return
	}
	userReqID := uint32(fd.UserRequestId)
	switch userReqID {
	case ids.base:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_AIRPORT {
			data := engine.CastDataAs[airportFacilityData](&fd.Data)
			details.ICAO = engine.BytesToString(data.ICAO[:])
			details.Region = engine.BytesToString(data.Region[:])
			details.Name = engine.BytesToString(data.Name[:])
			details.Name64 = engine.BytesToString(data.Name64[:])
			details.Latitude = data.Latitude
			details.Longitude = data.Longitude
			details.AltitudeM = data.Altitude
			details.MagVar = float64(data.MagVar)
			details.IsClosed = data.IsClosed != 0
			*foundBase = true
		}
	case ids.rw:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_RUNWAY {
			data := engine.CastDataAs[runwayFacilityData](&fd.Data)
			details.Runways = append(details.Runways, AirportRunway{
				Name:    runwayName(data.PrimaryNumber, data.PrimaryDesignator, data.SecondaryNumber, data.SecondaryDesignator),
				Heading: float64(data.Heading),
				LengthM: float64(data.Length),
				WidthM:  float64(data.Width),
				Surface: runwaySurfaceName(data.Surface),
			})
		}
	case ids.pk:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_TAXI_PARKING {
			data := engine.CastDataAs[parkingFacilityData](&fd.Data)
			if data.Number > 0 {
				details.Stands = append(details.Stands, AirportStand{
					Number:  int(data.Number),
					Type:    parkingTypeName(int32(data.Type)),
					Heading: float64(data.Heading),
				})
			}
		}
	case ids.fr:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_FREQUENCY {
			data := engine.CastDataAs[frequencyFacilityData](&fd.Data)
			details.Frequencies = append(details.Frequencies, AirportFrequency{
				Type:    frequencyTypeName(data.Type),
				FreqMHz: float64(data.Frequency) / 1_000_000.0,
				Name:    engine.BytesToString(data.Name[:]),
			})
		}
	case ids.hp:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_HELIPAD {
			data := engine.CastDataAs[helipadFacilityData](&fd.Data)
			details.Helipads = append(details.Helipads, AirportHelipad{
				Latitude:  data.Latitude,
				Longitude: data.Longitude,
				AltitudeM: data.Altitude,
				Heading:   float64(data.Heading),
				LengthM:   float64(data.Length),
				WidthM:    float64(data.Width),
				Surface:   runwaySurfaceName(data.Surface),
				Type:      helipadTypeName(data.Type),
			})
		}
	case ids.ap:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_APPROACH {
			data := engine.CastDataAs[approachFacilityData](&fd.Data)
			details.Approaches = append(details.Approaches, AirportApproach{
				Type:        approachTypeName(data.Type),
				Runway:      approachRunwayName(data.RunwayNumber, data.RunwayDesignator),
				HasLNAV:     data.HasLNAV != 0,
				HasLNAVVNAV: data.HasLNAVVNAV != 0,
				HasLP:       data.HasLP != 0,
				HasLPV:      data.HasLPV != 0,
			})
		}
	case ids.dp:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_DEPARTURE {
			data := engine.CastDataAs[departureFacilityData](&fd.Data)
			details.Departures = append(details.Departures, AirportProcedure{
				Name:               engine.BytesToString(data.Name[:]),
				RunwayTransitions:  int(data.NRunwayTransitions),
				EnrouteTransitions: int(data.NEnrouteTransitions),
			})
		}
	case ids.ar:
		if fd.Type == types.SIMCONNECT_FACILITY_DATA_ARRIVAL {
			data := engine.CastDataAs[departureFacilityData](&fd.Data)
			details.Arrivals = append(details.Arrivals, AirportProcedure{
				Name:               engine.BytesToString(data.Name[:]),
				RunwayTransitions:  int(data.NRunwayTransitions),
				EnrouteTransitions: int(data.NEnrouteTransitions),
			})
		}
	}
}

// SimEvents returns a read-only channel that receives lifecycle events.
// The channel is backed by an internal 64-element buffer; slow consumers may
// miss events if they do not drain promptly.
func (b *simconnectBridge) SimEvents() <-chan SimEvent {
	return b.simEventCh
}

// --- internal helpers ---

// allocIDs returns a unique (definitionID, requestID) pair by atomically
// incrementing the counter.  IDs are in the user-safe range 1–999_999_899.
func (b *simconnectBridge) allocIDs() (defID, reqID uint32) {
	// We multiply by 2 to produce non-overlapping pairs.  Counter starts at 0,
	// giving pairs (2,3), (4,5), (6,7)…  We add 2 to skip zero.
	n := b.idCounter.Add(1)
	defID = n * 2
	reqID = n*2 + 1
	return defID, reqID
}

// handleMessage is the Manager OnMessage callback. It runs synchronously in the
// dispatch goroutine for every message received from SimConnect.
//
// Routes:
//   - SIMCONNECT_RECV_ID_SIMOBJECT_DATA       → GetSimVar pending channel (float64)
//   - SIMCONNECT_RECV_ID_FACILITY_DATA        → GetAirportDetails per-call channel
//   - SIMCONNECT_RECV_ID_FACILITY_DATA_END    → GetAirportDetails per-call channel
func (b *simconnectBridge) handleMessage(msg engine.Message) {
	if msg.Err != nil {
		return
	}

	switch types.SIMCONNECT_RECV_ID(msg.DwID) {
	case types.SIMCONNECT_RECV_ID_SIMOBJECT_DATA:
		simObjData := msg.AsSimObjectData()
		if simObjData == nil {
			return
		}
		reqID := uint32(simObjData.DwRequestID)
		b.pendingMu.Lock()
		ch, ok := b.pending[reqID]
		if ok {
			delete(b.pending, reqID)
		}
		b.pendingMu.Unlock()
		if !ok {
			// Belongs to manager's own internal requests (camera/SimState).
			return
		}
		// The first 8 bytes of DwData are the float64 we requested.
		data := engine.CastDataAs[simVarDataStruct](&simObjData.DwData)
		select {
		case ch <- data.Value:
		default:
		}

	case types.SIMCONNECT_RECV_ID_FACILITY_DATA:
		// Decode inline — the SDK pool buffer is valid for the entire duration of
		// handleMessage but will be released (returned to pool) immediately after
		// this function returns. We must not forward msg to another goroutine.
		fd := msg.AsFacilityData()
		if fd == nil {
			return
		}
		b.facilityMu.Lock()
		state := b.facilityPending[uint32(fd.UserRequestId)]
		b.facilityMu.Unlock()
		if state != nil {
			state.mu.Lock()
			applyFacilityData(msg, state.details, state.ids, &state.foundBase)
			state.mu.Unlock()
		}

	case types.SIMCONNECT_RECV_ID_FACILITY_DATA_END:
		fde := msg.AsFacilityDataEnd()
		if fde == nil {
			return
		}
		rid := uint32(fde.RequestId)
		b.facilityMu.Lock()
		state := b.facilityPending[rid]
		b.facilityMu.Unlock()
		if state != nil {
			allDone := false
			state.mu.Lock()
			if _, tracked := state.endIDs[rid]; tracked && !state.endIDs[rid] {
				state.endIDs[rid] = true
				state.endCount++
				allDone = state.endCount == state.expected
			}
			state.mu.Unlock()
			if allDone {
				state.doneOnce.Do(func() { close(state.done) })
			}
		}
	}
}

// pushSimEvent delivers a SimEvent to the channel without blocking.
// Events are dropped if the internal buffer is full.
func (b *simconnectBridge) pushSimEvent(e SimEvent) {
	select {
	case b.simEventCh <- e:
	default:
		// Drop to avoid blocking the Manager dispatch goroutine.
	}
}
