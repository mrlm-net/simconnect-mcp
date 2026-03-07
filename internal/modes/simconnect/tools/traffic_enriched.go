//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterEnrichedTrafficTool registers the get_traffic_with_phase MCP tool.
func RegisterEnrichedTrafficTool(mcp *mcpadapter.Server, b bridge.Bridge) {
	tool := mcpadapter.NewTool("get_traffic_with_phase").
		Description("Return nearby aircraft with enriched telemetry: vertical speed (fpm), " +
			"actual ground track (vs magnetic heading), inferred flight phase " +
			"(PARKED/TAXI/CLIMB/CLIMB SHALLOW/LEVEL/DESCENT/APPROACH/FINAL), " +
			"parking state, runway occupancy, and aircraft category. " +
			"Uses a single SimConnect round-trip — same latency as get_nearby_traffic. " +
			"Radius defaults to 25 km if not specified.").
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

		entries, err := b.GetEnrichedTraffic(ctx, radius)
		if err != nil {
			return mcpadapter.JSONResult(map[string]any{
				"error":   fmt.Sprintf("enriched traffic scan failed: %v", err),
				"traffic": []any{},
			})
		}

		if entries == nil {
			entries = []bridge.EnrichedTrafficEntry{}
		}

		return mcpadapter.JSONResult(map[string]any{
			"radius_meters": radius,
			"count":         len(entries),
			"traffic":       entries,
		})
	})
}
