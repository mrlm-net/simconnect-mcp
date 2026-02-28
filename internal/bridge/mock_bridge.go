// Package bridge — mock_bridge.go
// NO build tag — must compile on all platforms.
package bridge

import (
	"context"
	"sync"
)

// MockBridge is a thread-safe Bridge implementation for use in tests.
// Configure its fields before use; all methods return the configured values.
type MockBridge struct {
	mu sync.RWMutex

	// MockState is the state returned by State().
	// Open() sets it to StateConnected; Close() sets it to StateDisconnected.
	MockState ConnectionState

	// MockSimVar is returned by GetSimVar when MockError is nil.
	MockSimVar SimVar

	// MockSimVarResults is returned by GetSimVars when MockError is nil.
	// If nil, GetSimVars returns one SimVarResult per request using MockSimVar.Value.
	MockSimVarResults []SimVarResult

	// MockSimState is returned by GetSimState.
	MockSimState SimState

	// MockError is returned by GetSimVar, GetSimVars, TransmitEvent, GetSimState
	// when non-nil. GetSimState ignores MockError (always succeeds).
	MockError error

	// EventsCh is the channel returned by SimEvents.
	// If nil, SimEvents returns a non-nil but never-written channel.
	EventsCh chan SimEvent

	// TransmitEventCalls records the (name, value) pairs passed to TransmitEvent.
	TransmitEventCalls []struct {
		Name  string
		Value uint32
	}
}

// Compile-time assertion: MockBridge must implement Bridge.
var _ Bridge = (*MockBridge)(nil)

func (m *MockBridge) Open(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MockState = StateConnected
	return nil
}

func (m *MockBridge) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MockState = StateDisconnected
	return nil
}

func (m *MockBridge) State() ConnectionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MockState
}

func (m *MockBridge) GetSimVar(_ context.Context, name, unit string) (SimVar, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockError != nil {
		return SimVar{}, m.MockError
	}
	sv := m.MockSimVar
	sv.Name = name
	sv.Unit = unit
	return sv, nil
}

func (m *MockBridge) GetSimVars(_ context.Context, vars []SimVarRequest) ([]SimVarResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockError != nil {
		return nil, m.MockError
	}
	if m.MockSimVarResults != nil {
		return m.MockSimVarResults, nil
	}
	results := make([]SimVarResult, len(vars))
	for i, v := range vars {
		results[i] = SimVarResult{
			Name:  v.Name,
			Unit:  v.Unit,
			Value: m.MockSimVar.Value,
		}
	}
	return results, nil
}

func (m *MockBridge) TransmitEvent(_ context.Context, name string, value uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TransmitEventCalls = append(m.TransmitEventCalls, struct {
		Name  string
		Value uint32
	}{Name: name, Value: value})
	return m.MockError
}

func (m *MockBridge) GetSimState(_ context.Context) (SimState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state := m.MockSimState
	state.Connected = m.MockState == StateConnected
	return state, nil
}

func (m *MockBridge) SimEvents() <-chan SimEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.EventsCh != nil {
		return m.EventsCh
	}
	return make(chan SimEvent)
}
