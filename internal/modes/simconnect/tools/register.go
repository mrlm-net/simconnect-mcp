//go:build windows

package tools

import (
	"os"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/manager"
)

// RegisterAll registers every live SimConnect tool. It is the single list
// shared by simconnect and both modes, so neither can miss a tool.
//
// The airport, weather and navigation tools built on the mrlm-net/simconnect
// library need the bridge's manager; a bridge without one (the mock) gets
// the bridge tools only.
//
// It returns the cleanup to run before the bridge closes: it takes the AI
// aircraft the traffic tools spawned out of the simulator, which keeps them
// after we disconnect. It returns how many it removed.
func RegisterAll(mcp *mcpadapter.Server, b bridge.Bridge) (cleanup func() int) {
	RegisterSimVarTools(mcp, b)
	RegisterSetSimVarTool(mcp, b)
	RegisterEventTools(mcp, b)
	RegisterStateTools(mcp, b)
	RegisterFuelTool(mcp, b)
	RegisterTrafficTool(mcp, b)
	RegisterEnrichedTrafficTool(mcp, b)
	RegisterAirportTools(mcp, b)
	RegisterNavaidTools(mcp, b)

	if p, ok := b.(interface{ Manager() manager.Manager }); ok && p.Manager() != nil {
		rt := live.NewRuntime(p.Manager())
		RegisterLiveTools(mcp, rt)
		RegisterLiveAircraftTools(mcp, live.NewAircraft(p.Manager(), os.Getenv("SIMCONNECT_AIRCRAFT_PROFILES")))
		tw := live.NewTrafficWorld(p.Manager(), os.Getenv("SIMCONNECT_TRAFFIC_DATA"))
		RegisterWorldTrafficTools(mcp, tw)
		return func() int {
			n := tw.Close() // no more spawns, then our aircraft go
			rt.Close()
			return n
		}
	}
	return func() int { return 0 }
}

// RegisterLiveTools registers the tools built on the library: airport
// procedures, taxi routes, stands, weather, runway in use, ATIS, fixes,
// airway routes and flight plans. RegisterLiveTrafficTools adds the AI traffic tools,
// RegisterLiveATCTools the airborne ATC tools, RegisterLiveScheduleTools
// the scheduled traffic tools.
func RegisterLiveTools(mcp *mcpadapter.Server, src live.Source) {
	RegisterLiveAirportTools(mcp, src)
	RegisterLiveNavTools(mcp, src)
}
