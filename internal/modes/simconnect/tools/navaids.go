//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterNavaidTools registers the six navaid MCP tools onto the provided server:
// get_vors_in_range, get_vor_details, get_ndbs_in_range, get_ndb_details,
// get_waypoints_in_range, get_waypoint_details.
func RegisterNavaidTools(mcp *mcpadapter.Server, b bridge.Bridge) {
	registerGetVORsInRange(mcp, b)
	registerGetVORDetails(mcp, b)
	registerGetNDBsInRange(mcp, b)
	registerGetNDBDetails(mcp, b)
	registerGetWaypointsInRange(mcp, b)
	registerGetWaypointDetails(mcp, b)
}

func registerGetVORsInRange(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_vors_in_range").
		Description("Return a list of VOR navigation stations in the simulator's reality bubble, " +
			"sorted by distance from the player aircraft. Each entry includes ICAO code, region, " +
			"lat/lon, altitude (metres MSL), frequency (Hz), magnetic variation, and distance (km). " +
			"Use radius_km to limit results; defaults to 200 km, maximum 500 km.").
		NumberParam("radius_km", "Maximum distance from player aircraft in kilometres (default 200, max 500).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		maxDistKM := 200.0
		if v, ok := args["radius_km"]; ok {
			if n, ok := v.(float64); ok && n > 0 {
				if n > 500 {
					n = 500
				}
				maxDistKM = n
			}
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"error": "not connected to simulator",
				"vors":  []any{},
			})
		}

		all, err := b.GetVORs(ctx)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error": fmt.Sprintf("VOR list failed: %v", err),
				"vors":  []any{},
			})
		}

		filtered := make([]bridge.VOREntry, 0, len(all))
		for _, v := range all {
			if v.DistanceKM <= maxDistKM {
				filtered = append(filtered, v)
			}
		}

		return mcpadapter.JSONResult(map[string]any{
			"radius_km": maxDistKM,
			"count":     len(filtered),
			"vors":      filtered,
		})
	})
}

func registerGetVORDetails(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_vor_details").
		Description("Return detailed data for a specific VOR navigation station by ICAO code. " +
			"Includes position, frequency, magnetic variation, nav range, and capability flags " +
			"(is_nav, is_dme, is_tacan, has_glide_slope, has_back_course). " +
			"Leave region empty (default) for best results.").
		StringParam("icao", "ICAO identifier of the VOR (e.g. \"VOR\", \"OPO\").").
		StringParam("region", "Optional ICAO region code (e.g. \"LP\"). Leave empty to match any region.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if icao == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao is required"), nil
		}
		region, _ := args["region"].(string)

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		details, err := b.GetVORDetails(ctx, icao, region)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("VOR_DETAILS_ERROR: %v", err)), nil
		}
		if details == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("VOR_NOT_FOUND: VOR %q not found", icao)), nil
		}

		return mcpadapter.JSONResult(details)
	})
}

func registerGetNDBsInRange(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_ndbs_in_range").
		Description("Return a list of NDB navigation stations in the simulator's reality bubble, " +
			"sorted by distance from the player aircraft. Each entry includes ICAO code, region, " +
			"lat/lon, altitude (metres MSL), frequency (Hz), magnetic variation, and distance (km). " +
			"Use radius_km to limit results; defaults to 200 km, maximum 500 km.").
		NumberParam("radius_km", "Maximum distance from player aircraft in kilometres (default 200, max 500).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		maxDistKM := 200.0
		if v, ok := args["radius_km"]; ok {
			if n, ok := v.(float64); ok && n > 0 {
				if n > 500 {
					n = 500
				}
				maxDistKM = n
			}
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"error": "not connected to simulator",
				"ndbs":  []any{},
			})
		}

		all, err := b.GetNDBs(ctx)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error": fmt.Sprintf("NDB list failed: %v", err),
				"ndbs":  []any{},
			})
		}

		filtered := make([]bridge.NDBEntry, 0, len(all))
		for _, n := range all {
			if n.DistanceKM <= maxDistKM {
				filtered = append(filtered, n)
			}
		}

		return mcpadapter.JSONResult(map[string]any{
			"radius_km": maxDistKM,
			"count":     len(filtered),
			"ndbs":      filtered,
		})
	})
}

func registerGetNDBDetails(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_ndb_details").
		Description("Return detailed data for a specific NDB navigation station by ICAO code. " +
			"Includes position, frequency (Hz and kHz), type, range, magnetic variation, " +
			"name, and terminal flag. " +
			"Leave region empty (default) for best results.").
		StringParam("icao", "ICAO identifier of the NDB (e.g. \"BKK\", \"LIS\").").
		StringParam("region", "Optional ICAO region code (e.g. \"LP\"). Leave empty to match any region.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if icao == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao is required"), nil
		}
		region, _ := args["region"].(string)

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		details, err := b.GetNDBDetails(ctx, icao, region)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("NDB_DETAILS_ERROR: %v", err)), nil
		}
		if details == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("NDB_NOT_FOUND: NDB %q not found", icao)), nil
		}

		return mcpadapter.JSONResult(details)
	})
}

func registerGetWaypointsInRange(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_waypoints_in_range").
		Description("Return a list of waypoints in the simulator's reality bubble, " +
			"sorted by distance from the player aircraft. Each entry includes ICAO code, region, " +
			"lat/lon, altitude (metres MSL), magnetic variation, and distance (km). " +
			"Use radius_km to limit results (default 100 km, max 500 km) and limit to cap the count.").
		NumberParam("radius_km", "Maximum distance from player aircraft in kilometres (default 100, max 500).").
		NumberParam("limit", "Maximum number of waypoints to return (default 200, max 1000).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		maxDistKM := 100.0
		if v, ok := args["radius_km"]; ok {
			if n, ok := v.(float64); ok && n > 0 {
				if n > 500 {
					n = 500
				}
				maxDistKM = n
			}
		}
		maxCount := 200
		if v, ok := args["limit"]; ok {
			if n, ok := v.(float64); ok && n > 0 {
				if n > 1000 {
					n = 1000
				}
				maxCount = int(n)
			}
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"error":     "not connected to simulator",
				"waypoints": []any{},
			})
		}

		all, err := b.GetWaypoints(ctx)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error":     fmt.Sprintf("waypoint list failed: %v", err),
				"waypoints": []any{},
			})
		}

		filtered := make([]bridge.WaypointEntry, 0, len(all))
		for _, w := range all {
			if w.DistanceKM <= maxDistKM {
				filtered = append(filtered, w)
			}
		}
		if len(filtered) > maxCount {
			filtered = filtered[:maxCount]
		}

		return mcpadapter.JSONResult(map[string]any{
			"radius_km": maxDistKM,
			"limit":     maxCount,
			"count":     len(filtered),
			"waypoints": filtered,
		})
	})
}

func registerGetWaypointDetails(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_waypoint_details").
		Description("Return detailed data for a specific waypoint by ICAO code. " +
			"Includes position, type, magnetic variation, number of airways routes, " +
			"and terminal flag. " +
			"Leave region empty (default) for best results.").
		StringParam("icao", "ICAO identifier of the waypoint (e.g. \"ABRIX\", \"TANGO\").").
		StringParam("region", "Optional ICAO region code (e.g. \"LP\"). Leave empty to match any region.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, _ := args["icao"].(string)
		if icao == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: icao is required"), nil
		}
		region, _ := args["region"].(string)

		if b.State() != bridge.StateConnected {
			return mcpadapter.ErrorResult("BRIDGE_DISCONNECTED: not connected to simulator"), nil
		}

		details, err := b.GetWaypointDetails(ctx, icao, region)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("WAYPOINT_DETAILS_ERROR: %v", err)), nil
		}
		if details == nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("WAYPOINT_NOT_FOUND: waypoint %q not found", icao)), nil
		}

		return mcpadapter.JSONResult(details)
	})
}
