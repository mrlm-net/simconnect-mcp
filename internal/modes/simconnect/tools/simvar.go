//go:build windows

// Package tools registers the live-data MCP tools for simconnect mode.
package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterSimVarTools registers the get_simvar_value and get_simvar_values MCP tools.
// Both tools require an active bridge connection (bridge.StateConnected).
func RegisterSimVarTools(mcp *mcpadapter.Server, b bridge.Bridge) {
	registerGetSimVarValue(mcp, b)
	registerGetSimVarValues(mcp, b)
}

// registerGetSimVarValue registers the get_simvar_value tool, which reads a
// single live simulation variable from the running simulator by name and unit.
func registerGetSimVarValue(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_simvar_value").
		Description("Read a single live simulation variable from the running simulator.").
		StringParam("name", "SimVar name, e.g. 'PLANE ALTITUDE'").
		StringParam("unit", "Unit string, e.g. 'feet'").
		Required("name", "unit").
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
		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: no active simulator connection"), nil
		}

		sv, err := b.GetSimVar(ctx, name, unit)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
		}

		return mcpadapter.JSONResult(map[string]any{
			"name":     sv.Name,
			"value":    sv.Value,
			"unit":     sv.Unit,
			"sim_time": sv.SimTime,
		})
	})
}

// registerGetSimVarValues registers the get_simvar_values tool, which reads up
// to 20 live simulation variables in a single call. Per-item failures are
// embedded in the response and do not abort the batch.
func registerGetSimVarValues(mcp *mcpadapter.Server, b bridge.Bridge) {
	// The vars parameter is an array of objects — SchemaProperty only models
	// scalar types, so we register it as "array" and enforce item shape and
	// the max-20 limit in the handler.
	tool := mcpadapter.NewTool("get_simvar_values").
		Description("Read up to 20 live simulation variables in one call. " +
			"'vars' must be an array of objects each with 'name' (string) and 'unit' (string).").
		Required("vars").
		Build()

	// Manually add the array param — NewTool only exposes scalar builders, so we
	// inject the SchemaProperty directly after Build() would lose the reference.
	// Use a fresh builder approach: re-build from scratch with the array property
	// embedded via the exported InputSchema field on the returned Tool.
	tool.InputSchema.Properties["vars"] = mcpadapter.SchemaProperty{
		Type:        "array",
		Description: "List of SimVar name/unit pairs to read (max 20). Each item must have 'name' and 'unit' string fields.",
	}

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		rawVars, ok := args["vars"]
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'vars' is required"), nil
		}

		varList, ok := rawVars.([]any)
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'vars' must be an array"), nil
		}
		if len(varList) == 0 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: 'vars' must not be empty"), nil
		}
		if len(varList) > 20 {
			return mcpadapter.ErrorResult(
				fmt.Sprintf("INVALID_ARGUMENT: 'vars' exceeds maximum of 20 items (got %d)", len(varList)),
			), nil
		}

		requests := make([]bridge.SimVarRequest, 0, len(varList))
		for i, v := range varList {
			m, ok := v.(map[string]any)
			if !ok {
				return mcpadapter.ErrorResult(
					fmt.Sprintf("INVALID_ARGUMENT: vars[%d] must be an object", i),
				), nil
			}
			name, _ := m["name"].(string)
			unit, _ := m["unit"].(string)
			if name == "" {
				return mcpadapter.ErrorResult(
					fmt.Sprintf("INVALID_ARGUMENT: vars[%d].name is required", i),
				), nil
			}
			if unit == "" {
				return mcpadapter.ErrorResult(
					fmt.Sprintf("INVALID_ARGUMENT: vars[%d].unit is required", i),
				), nil
			}
			requests = append(requests, bridge.SimVarRequest{Name: name, Unit: unit})
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: no active simulator connection"), nil
		}

		results, err := b.GetSimVars(ctx, requests)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
		}

		out := make([]map[string]any, len(results))
		for i, r := range results {
			if r.Error != "" {
				// Per-item failure: include name and unit for context but omit value/sim_time.
				out[i] = map[string]any{
					"name":  r.Name,
					"unit":  r.Unit,
					"error": r.Error,
				}
			} else {
				out[i] = map[string]any{
					"name":     r.Name,
					"value":    r.Value,
					"unit":     r.Unit,
					"sim_time": r.SimTime,
				}
			}
		}

		return mcpadapter.JSONResult(out)
	})
}
