//go:build windows

package tools

import (
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterAll registers every live SimConnect tool. It is the single list
// shared by simconnect and both modes, so neither can miss a tool.
func RegisterAll(mcp *mcpadapter.Server, b bridge.Bridge) {
	RegisterSimVarTools(mcp, b)
	RegisterSetSimVarTool(mcp, b)
	RegisterEventTools(mcp, b)
	RegisterStateTools(mcp, b)
	RegisterTrafficTool(mcp, b)
	RegisterEnrichedTrafficTool(mcp, b)
	RegisterAirportTools(mcp, b)
	RegisterNavaidTools(mcp, b)
}
