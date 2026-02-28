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

// Mount implements modes.Mode. It creates a real SimConnect bridge and mounts
// all routes onto the provided Gin engine.
func (m *simconnectMode) Mount(r *gin.Engine) error {
	b := bridge.NewSimConnectBridge()
	return m.MountWithBridge(r, b)
}

// MountWithBridge mounts all routes using the provided Bridge implementation.
// Use this in tests to inject a mock bridge.
func (m *simconnectMode) MountWithBridge(r *gin.Engine, b bridge.Bridge) error {
	m.bridge = b

	// Attempt initial connection — non-fatal if the simulator is not running yet.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := b.Open(ctx, m.cfg.AppName); err != nil {
		log.Printf("[simconnect] warning: initial connection failed: %v (will retry automatically)", err)
	}

	// Create the MCP server and register all tool groups.
	mcp := mcpadapter.NewServer("simconnect-mcp", "1.0.0")
	tools.RegisterSimVarTools(mcp, b)
	tools.RegisterEventTools(mcp, b)
	tools.RegisterStateTools(mcp, b)

	// Mount MCP transports onto the Gin engine.
	mcp.MountStreamableHTTP(r, "/mcp")
	mcp.MountSSE(r, "/sse", "/message")

	// Health endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, m.HealthInfo())
	})

	return nil
}
