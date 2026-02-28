package main

import (
	"log"
	"os"

	"github.com/mrlm-net/simconnect-mcp/internal/server"
)

func main() {
	mode := os.Getenv("MCP_MODE")
	if mode == "" {
		mode = "docs"
	}

	factory, ok := modeRegistry[mode]
	if !ok {
		log.Fatalf("unknown MCP_MODE: %q (available on this platform: %v)", mode, availableModes())
	}

	m, listenAddr, err := factory()
	if err != nil {
		log.Fatalf("failed to configure %s mode: %v", mode, err)
	}

	r := server.New()
	if err := m.Mount(r); err != nil {
		log.Fatalf("failed to mount %s mode: %v", mode, err)
	}

	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
