//go:build windows

package tools

import (
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
func RegisterAll(mcp *mcpadapter.Server, b bridge.Bridge) {
	RegisterSimVarTools(mcp, b)
	RegisterSetSimVarTool(mcp, b)
	RegisterEventTools(mcp, b)
	RegisterStateTools(mcp, b)
	RegisterTrafficTool(mcp, b)
	RegisterEnrichedTrafficTool(mcp, b)
	RegisterAirportTools(mcp, b)
	RegisterNavaidTools(mcp, b)

	if p, ok := b.(interface{ Manager() manager.Manager }); ok && p.Manager() != nil {
		rt := live.NewRuntime(p.Manager())
		RegisterLiveTools(mcp, rt)
		RegisterLiveTrafficTools(mcp, rt, rt)
	}
}

// RegisterLiveTools registers the tools built on the library: airport
// procedures, taxi routes, stands, weather, runway in use, ATIS, fixes,
// airway routes and flight plans. RegisterLiveTrafficTools adds the AI traffic tools.
func RegisterLiveTools(mcp *mcpadapter.Server, src live.Source) {
	RegisterLiveAirportTools(mcp, src)
	RegisterLiveNavTools(mcp, src)
}
