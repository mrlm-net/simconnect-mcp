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

	// MockSetSimVarError is returned by SetSimVar when non-nil.
	MockSetSimVarError error

	// SetSimVarCalls records arguments passed to SetSimVar.
	SetSimVarCalls []struct {
		Name  string
		Unit  string
		Value float64
	}

	// MockTrafficResults is returned by GetTraffic when MockTrafficError is nil.
	MockTrafficResults []TrafficEntry

	// MockTrafficError is returned by GetTraffic when non-nil.
	MockTrafficError error

	// MockEnrichedTrafficResults is returned by GetEnrichedTraffic when MockEnrichedTrafficError is nil.
	MockEnrichedTrafficResults []EnrichedTrafficEntry

	// MockEnrichedTrafficError is returned by GetEnrichedTraffic when non-nil.
	MockEnrichedTrafficError error

	// MockAirports is returned by GetAirports and GetNearestAirport when MockAirportsError is nil.
	MockAirports []AirportEntry

	// MockAirportsError is returned by GetAirports and GetNearestAirport when non-nil.
	MockAirportsError error

	// MockAirportDetails is returned by GetAirportDetails when MockAirportDetailsError is nil.
	MockAirportDetails *AirportDetails

	// MockAirportDetailsError is returned by GetAirportDetails when non-nil.
	MockAirportDetailsError error

	// MockVORs is returned by GetVORs when MockVORsError is nil.
	MockVORs []VOREntry

	// MockVORsError is returned by GetVORs when non-nil.
	MockVORsError error

	// MockVORDetails is returned by GetVORDetails when MockVORDetailsError is nil.
	MockVORDetails *VORDetails

	// MockVORDetailsError is returned by GetVORDetails when non-nil.
	MockVORDetailsError error

	// MockNDBs is returned by GetNDBs when MockNDBsError is nil.
	MockNDBs []NDBEntry

	// MockNDBsError is returned by GetNDBs when non-nil.
	MockNDBsError error

	// MockNDBDetails is returned by GetNDBDetails when MockNDBDetailsError is nil.
	MockNDBDetails *NDBDetails

	// MockNDBDetailsError is returned by GetNDBDetails when non-nil.
	MockNDBDetailsError error

	// MockWaypoints is returned by GetWaypoints when MockWaypointsError is nil.
	MockWaypoints []WaypointEntry

	// MockWaypointsError is returned by GetWaypoints when non-nil.
	MockWaypointsError error

	// MockWaypointDetails is returned by GetWaypointDetails when MockWaypointDetailsError is nil.
	MockWaypointDetails *WaypointDetails

	// MockWaypointDetailsError is returned by GetWaypointDetails when non-nil.
	MockWaypointDetailsError error

	// MockAirportTaxiways is returned by GetAirportTaxiways when MockAirportTaxiwaysError is nil.
	MockAirportTaxiways *AirportTaxiways

	// MockAirportTaxiwaysError is returned by GetAirportTaxiways when non-nil.
	MockAirportTaxiwaysError error

	// MockAirportParkings is returned by GetAirportParkings when MockAirportParkingsError is nil.
	MockAirportParkings *AirportParkings

	// MockAirportParkingsError is returned by GetAirportParkings when non-nil.
	MockAirportParkingsError error
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

func (m *MockBridge) SetSimVar(_ context.Context, name, unit string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SetSimVarCalls = append(m.SetSimVarCalls, struct {
		Name  string
		Unit  string
		Value float64
	}{Name: name, Unit: unit, Value: value})
	return m.MockSetSimVarError
}

func (m *MockBridge) GetSimState(_ context.Context) (SimState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state := m.MockSimState
	state.Connected = m.MockState == StateConnected
	return state, nil
}

func (m *MockBridge) GetTraffic(_ context.Context, _ uint32) ([]TrafficEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockTrafficError != nil {
		return nil, m.MockTrafficError
	}
	return m.MockTrafficResults, nil
}

func (m *MockBridge) GetEnrichedTraffic(_ context.Context, _ uint32) ([]EnrichedTrafficEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockEnrichedTrafficError != nil {
		return nil, m.MockEnrichedTrafficError
	}
	return m.MockEnrichedTrafficResults, nil
}

func (m *MockBridge) GetAirports(_ context.Context) ([]AirportEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockAirportsError != nil {
		return nil, m.MockAirportsError
	}
	return m.MockAirports, nil
}

func (m *MockBridge) GetNearestAirport(_ context.Context) (*AirportEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockAirportsError != nil {
		return nil, m.MockAirportsError
	}
	if len(m.MockAirports) == 0 {
		return nil, nil
	}
	entry := m.MockAirports[0]
	return &entry, nil
}

func (m *MockBridge) GetAirportDetails(_ context.Context, _, _ string, _ bool) (*AirportDetails, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockAirportDetailsError != nil {
		return nil, m.MockAirportDetailsError
	}
	return m.MockAirportDetails, nil
}

func (m *MockBridge) GetVORs(_ context.Context) ([]VOREntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockVORsError != nil {
		return nil, m.MockVORsError
	}
	return m.MockVORs, nil
}

func (m *MockBridge) GetVORDetails(_ context.Context, _, _ string) (*VORDetails, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockVORDetailsError != nil {
		return nil, m.MockVORDetailsError
	}
	return m.MockVORDetails, nil
}

func (m *MockBridge) GetNDBs(_ context.Context) ([]NDBEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockNDBsError != nil {
		return nil, m.MockNDBsError
	}
	return m.MockNDBs, nil
}

func (m *MockBridge) GetNDBDetails(_ context.Context, _, _ string) (*NDBDetails, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockNDBDetailsError != nil {
		return nil, m.MockNDBDetailsError
	}
	return m.MockNDBDetails, nil
}

func (m *MockBridge) GetWaypoints(_ context.Context) ([]WaypointEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockWaypointsError != nil {
		return nil, m.MockWaypointsError
	}
	return m.MockWaypoints, nil
}

func (m *MockBridge) GetWaypointDetails(_ context.Context, _, _ string) (*WaypointDetails, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockWaypointDetailsError != nil {
		return nil, m.MockWaypointDetailsError
	}
	return m.MockWaypointDetails, nil
}

func (m *MockBridge) GetAirportTaxiways(_ context.Context, _, _ string) (*AirportTaxiways, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockAirportTaxiwaysError != nil {
		return nil, m.MockAirportTaxiwaysError
	}
	return m.MockAirportTaxiways, nil
}

func (m *MockBridge) GetAirportParkings(_ context.Context, _, _ string) (*AirportParkings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MockAirportParkingsError != nil {
		return nil, m.MockAirportParkingsError
	}
	return m.MockAirportParkings, nil
}

func (m *MockBridge) SimEvents() <-chan SimEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.EventsCh != nil {
		return m.EventsCh
	}
	return make(chan SimEvent)
}
