package modes

import "github.com/gin-gonic/gin"

// Mode is implemented by each operating mode (docs, simconnect).
type Mode interface {
	// Mount registers all HTTP routes for this mode on the given Gin engine.
	Mount(r *gin.Engine) error
	// HealthInfo returns mode-specific status fields for the /health endpoint.
	HealthInfo() map[string]any
}
