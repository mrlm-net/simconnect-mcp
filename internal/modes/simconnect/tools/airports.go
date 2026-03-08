//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/convert"
)

// RegisterAirportTools registers the get_airports_in_range, get_nearest_airport,
// and get_airport_details MCP tools onto the provided server.
func RegisterAirportTools(mcp *mcpadapter.Server, b bridge.Bridge) {
	registerGetAirportsInRange(mcp, b)
	registerGetNearestAirport(mcp, b)
	registerGetAirportDetails(mcp, b)
}

func registerGetAirportsInRange(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_airports_in_range").
		Description("Return a list of airports in the simulator's reality bubble (loaded scenery area), " +
			"sorted by distance from the player aircraft. Each entry includes ICAO code, region, " +
			"lat/lon, altitude (metres MSL), and distance (km). " +
			"By default only standard ICAO airports are returned (4 uppercase letters, e.g. EDDM). " +
			"Set expanded=true to include all entries (private fields, military strips, simulator-only identifiers). " +
			"Use radius_km to limit results; defaults to 50 km, maximum 500 km.").
		NumberParam("radius_km", "Maximum distance from player aircraft in kilometres (default 50, max 500).").
		BoolParam("expanded", "When true, include non-standard identifiers (private fields, simulator-only codes). Default false.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		maxDistKM := 50.0
		if v, ok := args["radius_km"]; ok {
			if n, ok := v.(float64); ok && n > 0 {
				if n > 500 {
					n = 500
				}
				maxDistKM = n
			}
		}

		expanded, _ := args["expanded"].(bool)

		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"error":    "not connected to simulator",
				"airports": []any{},
			})
		}

		all, err := b.GetAirports(ctx)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error":    fmt.Sprintf("airport list failed: %v", err),
				"airports": []any{},
			})
		}

		// Filter by radius and, unless expanded, by valid ICAO format.
		filtered := make([]bridge.AirportEntry, 0, len(all))
		for _, a := range all {
			if a.DistanceKM > maxDistKM {
				continue
			}
			if !expanded && !convert.IsICAOCode(a.ICAO) {
				continue
			}
			filtered = append(filtered, a)
		}

		return mcpadapter.JSONResult(map[string]any{
			"radius_km": maxDistKM,
			"expanded":  expanded,
			"count":     len(filtered),
			"airports":  filtered,
		})
	})
}

func registerGetNearestAirport(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_nearest_airport").
		Description("Return the single closest airport to the player aircraft. " +
			"Includes ICAO code, region, lat/lon, altitude (metres MSL), and distance (km).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"error": "not connected to simulator",
			})
		}

		entry, err := b.GetNearestAirport(ctx)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error": fmt.Sprintf("nearest airport lookup failed: %v", err),
			})
		}
		if entry == nil {
			return mcpadapter.JSONResult(map[string]any{
				"error": "no airports found in reality bubble",
			})
		}

		return mcpadapter.JSONResult(entry)
	})
}

func registerGetAirportDetails(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_airport_details").
		Description("Return detailed facility data for a specific airport by ICAO code. " +
			"Default response includes name, lat/lon, altitude (metres MSL), magnetic variation, closed status, " +
			"runways (heading, length_m, width_m, surface), and ATC frequencies. " +
			"Set expanded=true to also include parking stands, helipads, instrument approaches, " +
			"departure procedures (SIDs), and arrival procedures (STARs). " +
			"Leave region empty (default) for best results — SimConnect's region filter is strict and " +
			"will silently fail if the region code does not match the simulator's internal value exactly.").
		StringParam("icao", "ICAO airport code (e.g. \"LPMA\", \"EDDM\").").
		StringParam("region", "Optional ICAO region code (e.g. \"LP\", \"ED\"). Leave empty to match any region.").
		BoolParam("expanded", "When true, also include stands, helipads, approaches, SIDs, and STARs. Default false.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if icao == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao is required"), nil
		}
		region, _ := args["region"].(string)
		expanded, _ := args["expanded"].(bool)

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		details, err := b.GetAirportDetails(ctx, icao, region, expanded)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("AIRPORT_DETAILS_ERROR: %v", err)), nil
		}
		if details == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("AIRPORT_NOT_FOUND: airport %q not found", icao)), nil
		}

		return mcpadapter.JSONResult(details)
	})
}
