package modes

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Mode is implemented by each operating mode (docs, simconnect).
type Mode interface {
	// Mount registers all HTTP routes for this mode on the given Gin engine.
	Mount(r *gin.Engine) error
	// ServeStdio runs the MCP server over stdin/stdout (stdio transport).
	// Used by local MCP clients such as Claude Code.
	ServeStdio(ctx context.Context) error
	// HealthInfo returns mode-specific status fields for the /health endpoint.
	HealthInfo() map[string]any
}
