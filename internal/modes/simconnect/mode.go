//go:build windows

package simconnect

import (
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
)

// simconnectMode implements modes.Mode for the live SimConnect data mode.
type simconnectMode struct {
	cfg    Config
	bridge bridge.Bridge
}

// New creates a new simconnectMode with the given configuration.
func New(cfg Config) modes.Mode {
	return &simconnectMode{cfg: cfg}
}

// Ensure simconnectMode implements modes.Mode at compile time.
var _ modes.Mode = (*simconnectMode)(nil)
