package middleware

import (
	"io"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

// notFoundMessages is a pool of aviation-themed 404 messages rotated at random.
var notFoundMessages = []string{
	"TERRAIN PULL UP — no route found at this altitude",
	"Unable to comply: no flight plan filed for this route",
	"Cleared to your destination via… actually, no, not cleared",
	"TCAS advisory: descend immediately, this URL does not exist",
	"Squawk 7700 — this endpoint is a navigation emergency",
	"Traffic alert: route not in radar contact",
	"Check your charts — this waypoint is not in our database",
	"Fuel state: zero pounds remaining on this route",
	"This frequency is not monitored",
	"You have deviated significantly from the glidepath",
	"SimConnect exception: SIMCONNECT_EXCEPTION_URL_NOT_FOUND (probably)",
	"ATC: say again your request, we have no record of that route",
	"Gear up, flaps up, 404 up",
	"Negative, that route is reserved for VFR traffic only",
	"Go-around: route not stabilised by 500 ft AGL",
	"Wind shear alert — this path leads nowhere",
	"This is a NOTAM: the requested route is closed until further notice",
}

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
// The message is chosen at random from notFoundMessages for entertainment.
func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		msg := notFoundMessages[rand.Intn(len(notFoundMessages))]
		c.JSON(http.StatusNotFound, errorEnvelope{
			Error: errorBody{Code: "NOT_FOUND", Message: msg},
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
