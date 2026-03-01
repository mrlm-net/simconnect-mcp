//go:build !windows

// Package both implements the combined docs+simconnect mode.
// On non-Windows platforms SimConnect is unavailable, so this build
// silently falls back to docs-only and exposes the same Mode interface.
package both

import (
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/docs"
)

// New returns a docs-only Mode on non-Windows platforms.
// The listen address is taken from the docs config (PORT env var).
func New() (modes.Mode, string, error) {
	cfg := docs.ConfigFromEnv()
	return docs.New(cfg), cfg.ListenAddr, nil
}
