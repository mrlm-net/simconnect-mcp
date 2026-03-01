package main

import (
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/both"
)

func init() {
	registerMode("both", func() (modes.Mode, string, error) {
		return both.New()
	})
}
