//go:build windows

package simconnect

import "github.com/mrlm-net/simconnect-mcp/internal/bridge"

// HealthInfo implements modes.Mode. It returns mode-specific status fields
// for the /health endpoint, including the current simulator connection state.
func (m *simconnectMode) HealthInfo() map[string]any {
	connected := false
	connectionState := "disconnected"

	if m.bridge != nil {
		state := m.bridge.State()
		connectionState = state.String()
		connected = state == bridge.StateConnected
	}

	return map[string]any{
		"status":           "ok",
		"mode":             "simconnect",
		"sim_connected":    connected,
		"connection_state": connectionState,
		"app_name":         m.cfg.AppName,
	}
}
