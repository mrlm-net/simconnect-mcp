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
}

// SimEvent represents a simulator lifecycle notification.
type SimEvent struct {
	Name  string `json:"name"`
	Value uint32 `json:"value,omitempty"`
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

	// GetSimState returns a snapshot of top-level simulator state.
	GetSimState(ctx context.Context) (SimState, error)

	// SimEvents returns a read-only channel that receives lifecycle events.
	SimEvents() <-chan SimEvent
}

// Sentinel errors returned by Bridge implementations.
var (
	ErrNotConnected    = errors.New("bridge: not connected to simulator")
	ErrUnknownVariable = errors.New("bridge: unknown simulation variable")
	ErrTimeout         = errors.New("bridge: request timed out")
)
