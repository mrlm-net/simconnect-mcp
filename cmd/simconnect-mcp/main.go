package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mrlm-net/simconnect-mcp/internal/server"
)

// Populated by -ldflags at build time (see .goreleaser.yml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("simconnect-mcp %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}

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

	// If stdin is a pipe (not an interactive terminal), use the stdio transport.
	// This is the case when launched by Claude Code or another local MCP client.
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		if err := m.ServeStdio(context.Background()); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
		return
	}

	r := server.New()
	if err := m.Mount(r); err != nil {
		log.Fatalf("failed to mount %s mode: %v", mode, err)
	}

	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
