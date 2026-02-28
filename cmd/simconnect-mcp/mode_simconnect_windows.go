//go:build windows

package main

import (
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/simconnect"
)

func init() {
	registerMode("simconnect", func() (modes.Mode, string, error) {
		cfg := simconnect.ConfigFromEnv()
		return simconnect.New(cfg), cfg.ListenAddr, nil
	})
}
