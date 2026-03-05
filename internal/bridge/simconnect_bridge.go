//go:build windows

// Package bridge — simconnect_bridge.go
// Windows-only implementation of Bridge backed by github.com/mrlm-net/simconnect.
package bridge

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

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
	eventID := b.eventIDCounter.Add(1)
	const groupID uint32 = 1

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
		types.SIMCONNECT_EVENT_FLAG_GROUPID_IS_PRIORITY,
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
//   - SimulatorVersion: not exposed by the Manager API; populated from
//     ConnectionOpen data when available (stored as empty string otherwise).
//     The Manager does not surface the application version string after Open
//     without a custom OnOpen handler; we return an empty string here and
//     document the gap in the GitHub issue comment.
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
		Connected:        connected,
		Paused:           mgrState.Paused,
		CurrentFlight:    flightFile,
		SimTime:          mgrState.ZuluTime,
		SimulatorVersion: simVersion,
		Latitude:         mgrState.Latitude,
		Longitude:        mgrState.Longitude,
		Altitude:         mgrState.Altitude,
		GroundSpeed:      mgrState.GroundSpeed,
		IndicatedAirspeed: mgrState.IndicatedAirspeed,
		VerticalSpeed:    mgrState.VerticalSpeed * 60, // fps → fpm
		TrueHeading:      mgrState.TrueHeading,
		OnGround:         mgrState.SimOnGround,
	}, nil
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
