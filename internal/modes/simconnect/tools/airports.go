//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/convert"
)

// RegisterAirportTools registers the get_airports_in_range, get_nearest_airport,
// get_airport_details, get_airport_taxiways, get_taxiway_names, and get_airport_parkings
// MCP tools onto the provided server.
func RegisterAirportTools(mcp *mcpadapter.Server, b bridge.Bridge) {
	registerGetAirportsInRange(mcp, b)
	registerGetNearestAirport(mcp, b)
	registerGetAirportDetails(mcp, b)
	registerGetAirportTaxiways(mcp, b)
	registerGetTaxiwayNames(mcp, b)
	registerGetAirportParkings(mcp, b)
}

func registerGetAirportsInRange(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_airports_in_range").
		Description("Airports in the reality bubble, nearest the user aircraft first.").
		NumberParam("radius_km", "km (default 50, max 500)").
		BoolParam("expanded", "Include non-ICAO identifiers (default false)").
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
		Description("The airport nearest the user aircraft.").
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
		Description("An airport's facility data: position, runways, frequencies; expanded adds stands, helipads, "+
			"approaches, SIDs and STARs.").
		StringParam("icao", "Airport ICAO").
		StringParam("region", "ICAO region; best left empty").
		BoolParam("expanded", "Add stands, helipads and procedures (default false)").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if !icaoRe.MatchString(icao) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao must be 1–9 uppercase alphanumeric characters"), nil
		}
		region, _ := args["region"].(string)
		if !regionRe.MatchString(region) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: region must be 0–4 uppercase alphanumeric characters"), nil
		}
		expanded, _ := args["expanded"].(bool)

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		details, err := b.GetAirportDetails(ctx, icao, region, expanded)
		if errors.Is(err, bridge.ErrTimeout) {
			return mcpadapter.ErrorResult(fmt.Sprintf("TIMEOUT: the simulator did not answer for %q after 3 tries", icao)), nil
		}
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("AIRPORT_DETAILS_ERROR: %v", err)), nil
		}
		if details == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("AIRPORT_NOT_FOUND: airport %q not found", icao)), nil
		}

		return mcpadapter.JSONResult(details)
	})
}

func registerGetAirportTaxiways(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_airport_taxiways").
		Description("An airport's taxiway graph: names, paths (edges) and points (nodes).").
		StringParam("icao", "Airport ICAO").
		StringParam("region", "ICAO region; best left empty").
		NumberParam("max_paths", "1–2000 (default 500)").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if !icaoRe.MatchString(icao) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao must be 1–9 uppercase alphanumeric characters"), nil
		}
		region, _ := args["region"].(string)
		if !regionRe.MatchString(region) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: region must be 0–4 uppercase alphanumeric characters"), nil
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		taxiways, err := b.GetAirportTaxiways(ctx, icao, region)
		if errors.Is(err, bridge.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			return mcpadapter.ErrorResult(fmt.Sprintf("TIMEOUT: the simulator did not answer for %q; try again", icao)), nil
		}
		if err != nil && !errors.Is(err, bridge.ErrNotFound) {
			return mcpadapter.ErrorResult(fmt.Sprintf("TAXIWAY_ERROR: %v", err)), nil
		}
		if taxiways == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("TAXIWAY_NOT_FOUND: airport %q has no taxiway data", icao)), nil
		}

		maxPaths := 500
		if v, ok := args["max_paths"]; ok {
			if n, ok := v.(float64); ok && n >= 1 {
				if n > 2000 {
					n = 2000
				}
				maxPaths = int(n)
			}
		}

		result := map[string]any{
			"icao":        taxiways.ICAO,
			"name_count":  taxiways.NameCount,
			"point_count": taxiways.PointCount,
			"path_count":  taxiways.PathCount,
			"names":       taxiways.Names,
			"points":      taxiways.Points,
		}

		if taxiways.PathCount > maxPaths {
			result["paths"] = taxiways.Paths[:maxPaths]
			result["truncated"] = true
			result["truncated_to"] = maxPaths
		} else {
			result["paths"] = taxiways.Paths
		}

		return mcpadapter.JSONResult(result)
	})
}

func registerGetTaxiwayNames(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_taxiway_names").
		Description("An airport's taxiway names only.").
		StringParam("icao", "Airport ICAO").
		StringParam("region", "ICAO region; best left empty").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if !icaoRe.MatchString(icao) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao must be 1–9 uppercase alphanumeric characters"), nil
		}
		region, _ := args["region"].(string)
		if !regionRe.MatchString(region) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: region must be 0–4 uppercase alphanumeric characters"), nil
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		taxiways, err := b.GetAirportTaxiways(ctx, icao, region)
		if errors.Is(err, bridge.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			return mcpadapter.ErrorResult(fmt.Sprintf("TIMEOUT: the simulator did not answer for %q; try again", icao)), nil
		}
		if err != nil && !errors.Is(err, bridge.ErrNotFound) {
			return mcpadapter.ErrorResult(fmt.Sprintf("TAXIWAY_ERROR: %v", err)), nil
		}
		if taxiways == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("TAXIWAY_NOT_FOUND: airport %q has no taxiway data", icao)), nil
		}

		return mcpadapter.JSONResult(map[string]any{
			"icao":       taxiways.ICAO,
			"name_count": taxiways.NameCount,
			"names":      taxiways.Names,
		})
	})
}

func registerGetAirportParkings(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_airport_parkings").
		Description("An airport's parking stands, gates and ramps (full TAXI_PARKING records).").
		StringParam("icao", "Airport ICAO").
		StringParam("region", "ICAO region; best left empty").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if !icaoRe.MatchString(icao) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao must be 1–9 uppercase alphanumeric characters"), nil
		}
		region, _ := args["region"].(string)
		if !regionRe.MatchString(region) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: region must be 0–4 uppercase alphanumeric characters"), nil
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		parkings, err := b.GetAirportParkings(ctx, icao, region)
		if errors.Is(err, bridge.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			return mcpadapter.ErrorResult(fmt.Sprintf("TIMEOUT: the simulator did not answer for %q; try again", icao)), nil
		}
		if err != nil && !errors.Is(err, bridge.ErrNotFound) {
			return mcpadapter.ErrorResult(fmt.Sprintf("PARKING_ERROR: %v", err)), nil
		}
		if parkings == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("PARKING_NOT_FOUND: airport %q has no parking data", icao)), nil
		}

		return mcpadapter.JSONResult(parkings)
	})
}
