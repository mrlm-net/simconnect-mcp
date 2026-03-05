//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterSetSimVarTool registers the set_simvar_value MCP tool with the given server.
// The tool writes a writable numeric simulation variable to the user aircraft.
func RegisterSetSimVarTool(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("set_simvar_value").
		Description("Write a numeric simulation variable to the user aircraft. " +
			"Only writable SimVars (e.g. 'AUTOPILOT ALTITUDE LOCK VAR', 'AUTOPILOT AIRSPEED HOLD VAR') will take effect. " +
			"Use get_simvar (docs tool) to verify whether a variable is settable.").
		StringParam("name", "SimVar name, e.g. 'AUTOPILOT ALTITUDE LOCK VAR'").
		StringParam("unit", "Unit string, e.g. 'feet'").
		NumberParam("value", "Numeric value to write").
		Required("name", "unit", "value").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		name, _ := args["name"].(string)
		unit, _ := args["unit"].(string)

		if name == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'name' is required"), nil
		}
		if unit == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'unit' is required"), nil
		}

		rawVal, hasVal := args["value"]
		if !hasVal || rawVal == nil {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'value' is required"), nil
		}
		value, ok := rawVal.(float64)
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'value' must be a number"), nil
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: no active simulator connection"), nil
		}

		if err := b.SetSimVar(ctx, name, unit, value); err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
		}

		return mcpadapter.JSONResult(map[string]any{
			"success": true,
			"name":    name,
			"unit":    unit,
			"value":   value,
		})
	})
}
