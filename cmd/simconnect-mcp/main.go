package main

import (
	"log"
	"os"

	"github.com/mrlm-net/simconnect-mcp/internal/modes/docs"
	"github.com/mrlm-net/simconnect-mcp/internal/server"
)

func main() {
	mode := os.Getenv("MCP_MODE")
	if mode == "" {
		mode = "docs"
	}

	r := server.New()

	var listenAddr string
	switch mode {
	case "docs":
		cfg := docs.ConfigFromEnv()
		listenAddr = cfg.ListenAddr
		m := docs.New(cfg)
		if err := m.Mount(r); err != nil {
			log.Fatalf("failed to start docs mode: %v", err)
		}
	default:
		log.Fatalf("unknown MCP_MODE: %q (supported: docs)", mode)
	}

	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
