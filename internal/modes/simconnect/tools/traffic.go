//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterTrafficTool registers the get_nearby_traffic MCP tool.
func RegisterTrafficTool(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_nearby_traffic").
		Description("Return a list of aircraft (AI and player) within a given radius of the user aircraft. "+
			"Each entry includes object ID, title, ATC callsign, airline, position, speed, heading, and on-ground flag. "+
			"The player aircraft is always included. Radius defaults to 25 km if not specified.").
		NumberParam("radius_meters", "Search radius in metres (default 25000, max 200000).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		radius := uint32(25_000)
		if v, ok := args["radius_meters"]; ok {
			switch n := v.(type) {
			case float64:
				if n > 0 && n <= 200_000 {
					radius = uint32(n)
				} else if n > 200_000 {
					radius = 200_000
				}
			}
		}

		if b.State() != bridge.StateConnected {
			return mcpadapter.JSONResult(map[string]any{
				"error":   "not connected to simulator",
				"traffic": []any{},
			})
		}

		entries, err := b.GetTraffic(ctx, radius)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error":   fmt.Sprintf("traffic scan failed: %v", err),
				"traffic": []any{},
			})
		}

		if entries == nil {
			entries = []bridge.TrafficEntry{}
		}

		return mcpadapter.JSONResult(map[string]any{
			"sim_time":      b.GetSimTime(),
			"radius_meters": radius,
			"count":         len(entries),
			"traffic":       entries,
		})
	})
}
