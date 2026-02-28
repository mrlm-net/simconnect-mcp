//go:build windows

package tools

import (
	"context"
	"fmt"
	"math"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterEventTools registers the transmit_event MCP tool with the given server.
// The tool transmits a named simulator input event to the running simulator instance.
func RegisterEventTools(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("transmit_event").
		Description("Transmit a named simulator input event to the running simulator.").
		StringParam("name", "SimConnect event name, e.g. 'LANDING_LIGHTS_TOGGLE'").
		NumberParam("value", "Optional DWORD value for parameterized events (default 0, range 0-4294967295)").
		Required("name").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'name' is required"), nil
		}

		// Parse optional value — JSON numbers arrive as float64 from encoding/json.
		var value uint32
		if rawVal, ok := args["value"]; ok && rawVal != nil {
			f, ok := rawVal.(float64)
			if !ok {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'value' must be a number"), nil
			}
			// Reject negative, non-integer, and out-of-range values.
			if f < 0 || f > math.MaxUint32 || f != math.Trunc(f) {
				return mcpadapter.ErrorResult(
					fmt.Sprintf("INVALID_ARGUMENT: 'value' must be an integer in range [0, 4294967295], got %v", f),
				), nil
			}
			value = uint32(f)
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: no active simulator connection"), nil
		}

		if err := b.TransmitEvent(ctx, name, value); err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
		}

		return mcpadapter.JSONResult(map[string]any{
			"success": true,
			"event":   name,
			"value":   value,
		})
	})
}
