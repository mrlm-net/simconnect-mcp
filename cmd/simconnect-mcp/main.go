package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/server"
)

// Populated by -ldflags at build time (see .goreleaser.yml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// shutdownTimeout bounds the HTTP server's graceful shutdown.
const shutdownTimeout = 5 * time.Second

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

	// Interrupt (Ctrl+C) or termination ends the server; the mode then
	// releases what it holds (the SimConnect connection, AI aircraft it
	// spawned).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// If stdin is a pipe (not an interactive terminal), use the stdio transport.
	// This is the case when launched by Claude Code or another local MCP client.
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		err := serveStdio(ctx, m)
		closeMode(m)
		if err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
		return
	}

	r := server.New()
	if err := m.Mount(r); err != nil {
		log.Fatalf("failed to mount %s mode: %v", mode, err)
	}

	srv := &http.Server{Addr: listenAddr, Handler: r}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		closeMode(m)
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
		closeMode(m)
	}
}

// serveStdio serves until stdin closes (the client went away) or ctx ends.
// A read from stdin cannot be interrupted, so on a signal it returns without
// waiting for it.
func serveStdio(ctx context.Context, m modes.Mode) error {
	errc := make(chan error, 1)
	go func() { errc <- m.ServeStdio(ctx) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return nil
	}
}

// closeMode releases what the mode holds, if anything.
func closeMode(m modes.Mode) {
	if c, ok := m.(modes.Closer); ok {
		if err := c.Close(); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}
