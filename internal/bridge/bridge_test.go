package bridge_test

import (
	"context"
	"testing"

	. "github.com/mrlm-net/simconnect-mcp/internal/bridge"
)

func TestMockBridge_InterfaceAssertion(t *testing.T) {
	// Compile-time assertion is in mock_bridge.go; this verifies runtime usability.
	var b Bridge = &MockBridge{}
	if b == nil {
		t.Fatal("MockBridge should not be nil")
	}
}

func TestMockBridge_OpenClose(t *testing.T) {
	m := &MockBridge{}
	if m.State() != StateDisconnected {
		t.Fatalf("initial state: got %v, want %v", m.State(), StateDisconnected)
	}
	_ = m.Open(context.Background(), "test")
	if m.State() != StateConnected {
		t.Fatalf("after Open: got %v, want %v", m.State(), StateConnected)
	}
	_ = m.Close()
	if m.State() != StateDisconnected {
		t.Fatalf("after Close: got %v, want %v", m.State(), StateDisconnected)
	}
}

func TestMockBridge_GetSimVar(t *testing.T) {
	m := &MockBridge{
		MockState:  StateConnected,
		MockSimVar: SimVar{Value: 35000, Unit: "feet"},
	}
	sv, err := m.GetSimVar(context.Background(), "PLANE ALTITUDE", "feet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv.Name != "PLANE ALTITUDE" {
		t.Errorf("Name: got %q, want %q", sv.Name, "PLANE ALTITUDE")
	}
	if sv.Value != 35000 {
		t.Errorf("Value: got %v, want 35000", sv.Value)
	}
}

func TestMockBridge_GetSimVar_Error(t *testing.T) {
	m := &MockBridge{
		MockState: StateConnected,
		MockError: ErrNotConnected,
	}
	_, err := m.GetSimVar(context.Background(), "PLANE ALTITUDE", "feet")
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestMockBridge_GetSimVars(t *testing.T) {
	m := &MockBridge{
		MockState:  StateConnected,
		MockSimVar: SimVar{Value: 250},
	}
	results, err := m.GetSimVars(context.Background(), []SimVarRequest{
		{Name: "AIRSPEED INDICATED", Unit: "knots"},
		{Name: "PLANE ALTITUDE", Unit: "feet"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "AIRSPEED INDICATED" {
		t.Errorf("results[0].Name: got %q", results[0].Name)
	}
}

func TestMockBridge_TransmitEvent(t *testing.T) {
	m := &MockBridge{MockState: StateConnected}
	err := m.TransmitEvent(context.Background(), "LANDING_LIGHTS_TOGGLE", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.TransmitEventCalls) != 1 {
		t.Fatalf("expected 1 call recorded, got %d", len(m.TransmitEventCalls))
	}
	if m.TransmitEventCalls[0].Name != "LANDING_LIGHTS_TOGGLE" {
		t.Errorf("call name: got %q", m.TransmitEventCalls[0].Name)
	}
}

func TestMockBridge_GetSimState(t *testing.T) {
	m := &MockBridge{
		MockState: StateConnected,
		MockSimState: SimState{
			Paused:           false,
			CurrentFlight:    "flight.flt",
			SimulatorVersion: "MSFS 2024",
		},
	}
	state, err := m.GetSimState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Connected {
		t.Error("expected Connected=true when MockState=StateConnected")
	}
	if state.CurrentFlight != "flight.flt" {
		t.Errorf("CurrentFlight: got %q", state.CurrentFlight)
	}
}

func TestMockBridge_SimEvents(t *testing.T) {
	m := &MockBridge{}
	ch := m.SimEvents()
	if ch == nil {
		t.Error("SimEvents should never return nil channel")
	}
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnectionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
