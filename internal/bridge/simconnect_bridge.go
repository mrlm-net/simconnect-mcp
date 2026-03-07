//go:build windows

// Package bridge — simconnect_bridge.go
// Windows-only implementation of Bridge backed by github.com/mrlm-net/simconnect.
package bridge

import (
	"context"
	"fmt"
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
//	LATITUDE → float64, LONGITUDE → float64, ALTITUDE → float64,
//	ICAO → [8]byte, NAME → [32]byte, NAME64 → [64]byte
type airportFacilityData struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	ICAO      [8]byte
	Name      [32]byte
	Name64    [64]byte
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
		mgrErrCh:   make(chan error, 1),
		simEventCh: make(chan SimEvent, 64),
		pending:     make(map[uint32]chan float64),
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

	mgr := manager.New(
		appName,
		manager.WithContext(mgrCtx),
		manager.WithAutoReconnect(true),
		manager.WithAutoDetect(),
		manager.WithSimStatePeriod(types.SIMCONNECT_PERIOD_SECOND),
	)
	b.mgr = mgr

	// Wire up the message handler so we can receive SimObject data responses
	// for GetSimVar / GetSimVars.
	mgr.OnMessage(b.handleMessage)

	// Capture simulator application version on connection open.
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

	sub := mgr.SubscribeWithType("", 256, []types.SIMCONNECT_RECV_ID{
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

// GetAirportDetails returns detailed facility data for the given ICAO airport.
// It issues AddToFacilityDefinition + RequestFacilityData calls and collects
// the FACILITY_DATA response, returning when FACILITY_DATA_END is received.
func (b *simconnectBridge) GetAirportDetails(ctx context.Context, icao, region string) (*AirportDetails, error) {
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

	// Register airport facility definition: OPEN/CLOSE markers bracket the fields.
	for _, field := range []string{
		"OPEN AIRPORT",
		"LATITUDE", "LONGITUDE", "ALTITUDE", "ICAO", "NAME", "NAME64",
		"CLOSE AIRPORT",
	} {
		if err := mgr.AddToFacilityDefinition(defID, field); err != nil {
			return nil, fmt.Errorf("bridge: AddToFacilityDefinition %q: %w", field, err)
		}
	}

	sub := mgr.SubscribeWithType("", 32, []types.SIMCONNECT_RECV_ID{
		types.SIMCONNECT_RECV_ID_FACILITY_DATA,
		types.SIMCONNECT_RECV_ID_FACILITY_DATA_END,
	})
	defer sub.Unsubscribe()

	if err := mgr.RequestFacilityData(defID, reqID, icao, region); err != nil {
		return nil, fmt.Errorf("bridge: RequestFacilityData %s: %w", icao, err)
	}

	var details *AirportDetails

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return details, nil
		case <-deadline.C:
			if details == nil {
				return nil, fmt.Errorf("bridge: airport %q not found (timeout)", icao)
			}
			return details, nil
		case msg, ok := <-sub.Messages():
			if !ok {
				return details, nil
			}
			switch types.SIMCONNECT_RECV_ID(msg.DwID) {
			case types.SIMCONNECT_RECV_ID_FACILITY_DATA:
				fd := msg.AsFacilityData()
				if fd == nil || uint32(fd.UserRequestId) != reqID {
					continue
				}
				if fd.Type != types.SIMCONNECT_FACILITY_DATA_AIRPORT {
					continue
				}
				data := engine.CastDataAs[airportFacilityData](&fd.Data)
				details = &AirportDetails{
					ICAO:      engine.BytesToString(data.ICAO[:]),
					Region:    region,
					Name:      engine.BytesToString(data.Name[:]),
					Name64:    engine.BytesToString(data.Name64[:]),
					Latitude:  data.Latitude,
					Longitude: data.Longitude,
					AltitudeM: data.Altitude,
				}
			case types.SIMCONNECT_RECV_ID_FACILITY_DATA_END:
				fd := msg.AsFacilityDataEnd()
				if fd == nil || uint32(fd.RequestId) != reqID {
					continue
				}
				return details, nil
			}
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

// handleMessage is the Manager OnMessage callback.  It inspects incoming
// SIMCONNECT_RECV_SIMOBJECT_DATA packets and delivers the float64 value to
// the waiting GetSimVar request, if any.
func (b *simconnectBridge) handleMessage(msg engine.Message) {
	if msg.Err != nil {
		return
	}
	if types.SIMCONNECT_RECV_ID(msg.DwID) != types.SIMCONNECT_RECV_ID_SIMOBJECT_DATA {
		return
	}

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
		// This message belongs to the manager's own internal requests (camera/SimState).
		return
	}

	// The first 8 bytes of DwData are the float64 we requested.
	data := engine.CastDataAs[simVarDataStruct](&simObjData.DwData)
	select {
	case ch <- data.Value:
	default:
		// Channel already has a value or was closed; discard.
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
