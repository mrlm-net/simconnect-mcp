package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that enforces cross-origin request policy.
//
// In release mode only localhost origins are permitted; non-localhost origins
// receive a 403 JSON error. In debug/test mode all origins are allowed so that
// local development tooling (e.g. Vite dev server on a different port) works
// without configuration.
//
// Allowed headers include Mcp-Session-Id to support MCP protocol session
// tracking alongside the standard Content-Type and X-Request-ID headers.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && !isLocalhost(origin) && gin.Mode() == gin.ReleaseMode {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "cross-origin request not allowed",
				},
			})
			return
		}
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Mcp-Session-Id")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID, Mcp-Session-Id")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// isLocalhost reports whether origin refers to a loopback address.
// It strips the scheme and port before comparing the host.
func isLocalhost(origin string) bool {
	s := origin
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// Strip port — use LastIndex so IPv6 brackets are handled simply.
	if i := strings.LastIndex(s, ":"); i >= 0 {
		s = s[:i]
	}
	return s == "localhost" || s == "127.0.0.1" || s == "::1"
}
