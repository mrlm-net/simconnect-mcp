//go:build windows

package simconnect

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/simconnect/tools"
)

// buildMCPServer creates a SimConnect bridge, attempts to connect, registers
// all SimConnect tools, and returns the configured MCP server.
func (m *simconnectMode) buildMCPServer(ctx context.Context) (*mcpadapter.Server, bridge.Bridge, error) {
	b := bridge.NewSimConnectBridge()
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := b.Open(tctx, m.cfg.AppName); err != nil {
		log.Printf("[simconnect] warning: initial connection failed: %v (will retry automatically)", err)
	}
	mcp := mcpadapter.NewServer("simconnect-mcp", "1.0.0")
	tools.RegisterSimVarTools(mcp, b)
	tools.RegisterSetSimVarTool(mcp, b)
	tools.RegisterEventTools(mcp, b)
	tools.RegisterStateTools(mcp, b)
	tools.RegisterTrafficTool(mcp, b)
	tools.RegisterEnrichedTrafficTool(mcp, b)
	tools.RegisterAirportTools(mcp, b)
	tools.RegisterNavaidTools(mcp, b)
	return mcp, b, nil
}

// Mount implements modes.Mode. It creates a real SimConnect bridge and mounts
// all routes onto the provided Gin engine.
func (m *simconnectMode) Mount(r *gin.Engine) error {
	mcp, b, err := m.buildMCPServer(context.Background())
	if err != nil {
		return err
	}
	m.bridge = b
	mcp.MountStreamableHTTP(r, "/mcp")
	mcp.MountSSE(r, "/sse", "/message")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, m.HealthInfo())
	})
	return nil
}

// MountWithBridge mounts all routes using the provided Bridge implementation.
// Use this in tests to inject a mock bridge.
func (m *simconnectMode) MountWithBridge(r *gin.Engine, b bridge.Bridge) error {
	m.bridge = b

	tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := b.Open(tctx, m.cfg.AppName); err != nil {
		log.Printf("[simconnect] warning: initial connection failed: %v (will retry automatically)", err)
	}

	mcp := mcpadapter.NewServer("simconnect-mcp", "1.0.0")
	tools.RegisterSimVarTools(mcp, b)
	tools.RegisterSetSimVarTool(mcp, b)
	tools.RegisterEventTools(mcp, b)
	tools.RegisterStateTools(mcp, b)
	tools.RegisterTrafficTool(mcp, b)
	tools.RegisterEnrichedTrafficTool(mcp, b)
	tools.RegisterAirportTools(mcp, b)
	tools.RegisterNavaidTools(mcp, b)

	mcp.MountStreamableHTTP(r, "/mcp")
	mcp.MountSSE(r, "/sse", "/message")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, m.HealthInfo())
	})

	return nil
}

// ServeStdio runs the MCP server over stdin/stdout (stdio transport).
func (m *simconnectMode) ServeStdio(ctx context.Context) error {
	mcp, b, err := m.buildMCPServer(ctx)
	if err != nil {
		return err
	}
	m.bridge = b
	return mcp.ServeStdio(ctx)
}
