package main

import (
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/docs"
)

func init() {
	registerMode("docs", func() (modes.Mode, string, error) {
		cfg := docs.ConfigFromEnv()
		return docs.New(cfg), cfg.ListenAddr, nil
	})
}
