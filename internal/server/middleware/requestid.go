package middleware

import (
	"crypto/rand"
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
)

type contextKey string

const requestIDKey contextKey = "request-id"

// requestIDRe accepts only safe characters for a request ID: alphanumeric and
// hyphens, 1–64 characters. Values that don't match are discarded and a fresh
// UUID is generated instead, preventing log injection via the header.
var requestIDRe = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,64}$`)

// RequestID is a middleware that assigns a unique request ID to every incoming
// request. If the client supplies an X-Request-ID header the value is reused
// (provided it matches the allowed pattern); otherwise a random UUID v4 is
// generated. The ID is echoed back in the response header and stored in the
// Gin context for downstream handlers.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !requestIDRe.MatchString(id) {
			id = newUUID()
		}
		c.Header("X-Request-ID", id)
		c.Set(string(requestIDKey), id)
		c.Next()
	}
}

// GetRequestID retrieves the request ID from a Gin context.
func GetRequestID(c *gin.Context) string {
	v, _ := c.Get(string(requestIDKey))
	s, _ := v.(string)
	return s
}

// newUUID generates a random UUID v4 using crypto/rand.
// Panics if the system random source is unavailable — that is an unrecoverable
// environmental failure and the server cannot operate safely without it.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("middleware: failed to generate request ID: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits 10xx
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
