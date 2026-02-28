//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterStateTools registers the get_sim_state MCP tool with the given server.
// The tool returns a snapshot of the current simulator connection state and
// flight status. When the bridge is not in StateConnected it returns
// {connected: false} rather than an error, so callers can safely poll it
// at any connection state.
func RegisterStateTools(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_sim_state").
		Description("Return a snapshot of the current simulator connection state and flight status.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		// Return {connected: false} rather than an error when disconnected.
		// This allows LLM clients to safely poll without triggering error-recovery flows.
		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"connected": false,
			})
		}

		state, err := b.GetSimState(ctx)
		if err != nil {
			return nil, fmt.Errorf("INTERNAL_ERROR: %w", err)
		}

		return mcpadapter.JSONResult(map[string]any{
			"connected":         state.Connected,
			"paused":            state.Paused,
			"current_flight":    state.CurrentFlight,
			"sim_time":          state.SimTime,
			"simulator_version": state.SimulatorVersion,
		})
	})
}
