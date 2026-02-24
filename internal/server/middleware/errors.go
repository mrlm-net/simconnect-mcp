package middleware

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// errorBody is the inner error object returned in JSON error responses.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// errorEnvelope wraps errorBody under an "error" key for consistent response
// shape across all error conditions.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

// NoRouteHandler returns a gin.HandlerFunc that responds with a structured
// JSON 404 when no route matches the incoming request.
func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, errorEnvelope{
			Error: errorBody{Code: "NOT_FOUND", Message: "route not found"},
		})
	}
}

// NoMethodHandler returns a gin.HandlerFunc that responds with a structured
// JSON 405 when the route exists but the HTTP method is not registered.
func NoMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, errorEnvelope{
			Error: errorBody{Code: "METHOD_NOT_ALLOWED", Message: "method not allowed"},
		})
	}
}

// RecoveryHandler returns a gin.HandlerFunc that recovers from panics and
// responds with a structured JSON 500. Panic output is discarded to avoid
// leaking internal details to callers.
func RecoveryHandler() gin.HandlerFunc {
	return gin.RecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) {
		c.JSON(http.StatusInternalServerError, errorEnvelope{
			Error: errorBody{Code: "INTERNAL_ERROR", Message: "internal server error"},
		})
	})
}
