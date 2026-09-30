//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// fuelVars are read in one GetSimVars batch: the totals, then quantity and
// capacity of each main tank.
var fuelVars = []bridge.SimVarRequest{
	{Name: "FUEL TOTAL QUANTITY", Unit: "gallons"},
	{Name: "FUEL TOTAL CAPACITY", Unit: "gallons"},
	{Name: "FUEL TOTAL QUANTITY WEIGHT", Unit: "pounds"},
	{Name: "FUEL TANK CENTER QUANTITY", Unit: "gallons"},
	{Name: "FUEL TANK CENTER CAPACITY", Unit: "gallons"},
	{Name: "FUEL TANK LEFT MAIN QUANTITY", Unit: "gallons"},
	{Name: "FUEL TANK LEFT MAIN CAPACITY", Unit: "gallons"},
	{Name: "FUEL TANK RIGHT MAIN QUANTITY", Unit: "gallons"},
	{Name: "FUEL TANK RIGHT MAIN CAPACITY", Unit: "gallons"},
}

var fuelTanks = []string{"Center", "Left Main", "Right Main"}

// fuelTank is one tank of the user aircraft.
type fuelTank struct {
	Name        string  `json:"name"`
	QuantityGal float64 `json:"quantity_gal"`
	CapacityGal float64 `json:"capacity_gal"`
	PercentFull float64 `json:"percent_full"`
}

func percent(qty, capacity float64) float64 {
	if capacity <= 0 {
		return 0
	}
	return round(qty/capacity*100, 1)
}

// RegisterFuelTool registers get_fuel_state: the user aircraft's fuel.
func RegisterFuelTool(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_fuel_state").
		Description("The user aircraft's fuel: total quantity and capacity (US gallons), percent full, and weight (lb and " +
			"kg), with the center, left main and right main tanks (tanks the aircraft does not have are left out).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to the simulator"), nil
		}
		res, err := b.GetSimVars(ctx, fuelVars)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: fuel: %v", err)), nil
		}
		if len(res) != len(fuelVars) {
			return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: fuel: %d of %d values", len(res), len(fuelVars))), nil
		}
		for _, r := range res {
			if r.Error != "" {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: fuel: %s: %s", r.Name, r.Error)), nil
			}
		}
		qty, capacity, lbs := res[0].Value, res[1].Value, res[2].Value
		tanks := []fuelTank{}
		for i, name := range fuelTanks {
			q, c := res[3+2*i].Value, res[4+2*i].Value
			if c <= 0 {
				continue // not fitted
			}
			tanks = append(tanks, fuelTank{name, round(q, 1), round(c, 1), percent(q, c)})
		}
		return mcpadapter.JSONResult(map[string]any{
			"total_quantity_gal": round(qty, 1),
			"total_capacity_gal": round(capacity, 1),
			"total_percent_full": percent(qty, capacity),
			"total_weight_lbs":   round(lbs, 0),
			"total_weight_kg":    round(lbs*0.45359237, 0),
			"tanks":              tanks,
		})
	})
}
