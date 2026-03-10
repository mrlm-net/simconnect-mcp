package server

import (
	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/server/middleware"
)

// New returns a configured Gin router with the standard middleware stack:
//   - RecoveryHandler — converts panics to structured JSON 500 responses
//   - gin.Logger      — request/response logging to stdout
//   - RequestID       — ensures every request carries an X-Request-ID header
//   - CORS            — enforces cross-origin policy (localhost-only in release mode)
//
// NoRoute and NoMethod handlers are also registered to produce structured JSON
// errors instead of Gin's default plain-text responses.
func New() *gin.Engine {
	// Release mode suppresses debug output and enforces the CORS localhost
	// restriction regardless of whether GIN_MODE is set in the environment.
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	engine.Use(middleware.RecoveryHandler())
	engine.Use(gin.Logger())
	engine.Use(middleware.RequestID())
	engine.Use(middleware.CORS())

	engine.NoRoute(middleware.NoRouteHandler())
	engine.NoMethod(middleware.NoMethodHandler())

	return engine
}
