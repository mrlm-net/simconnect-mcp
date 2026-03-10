// Package docs implements the documentation-fetch mode (Milestone 1).
// It exposes SimConnect SDK documentation via MCP tool endpoints,
// allowing AI agents to query reference material without a running simulator.
package docs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/docs/tools"
)

// docsMode is the concrete implementation of modes.Mode for documentation fetch.
// It holds the loaded corpus and derived store as unexported fields so that no
// global state leaks out of this package.
type docsMode struct {
	cfg        Config
	store      corpus.DocStore
	corpusData corpus.Corpus
}

// New constructs a docs mode instance from the provided Config.
// The corpus is not loaded until Mount or ServeStdio is called.
func New(cfg Config) modes.Mode {
	return &docsMode{cfg: cfg}
}

// buildMCPServer loads the corpus and returns a fully configured MCP server
// with all doc tools registered. Called by both Mount and ServeStdio.
func (m *docsMode) buildMCPServer() (*mcpadapter.Server, error) {
	var loader corpus.DocLoader
	if m.cfg.OverridePath != "" {
		loader = corpus.LoadFromPathVersion(m.cfg.OverridePath, m.cfg.MSFSVersion)
	} else {
		loader = corpus.LoadEmbeddedVersion(m.cfg.MSFSVersion)
	}

	c, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("docs mode: load corpus: %w", err)
	}
	m.corpusData = c
	m.store = corpus.NewDocStore(c)

	mcp := mcpadapter.NewServer("simconnect-mcp-docs", "1.0.0")
	tools.RegisterSimVarTools(mcp, m.store, m.cfg.LiveScrape)
	tools.RegisterEventTools(mcp, m.store, m.cfg.LiveScrape)
	tools.RegisterFunctionTools(mcp, m.store, m.cfg.LiveScrape)
	tools.RegisterStructureTools(mcp, m.store, m.cfg.LiveScrape)
	tools.RegisterErrorCodeTools(mcp, m.store, m.cfg.LiveScrape)
	tools.RegisterSearchTool(mcp, m.store, m.cfg.LiveScrape)
	return mcp, nil
}

// Mount loads the corpus, wires up the MCP server with all tool registrations,
// and registers the /mcp, /sse, /message, and /health routes on r.
func (m *docsMode) Mount(r *gin.Engine) error {
	mcp, err := m.buildMCPServer()
	if err != nil {
		return err
	}
	mcp.MountStreamableHTTP(r, "/mcp")
	mcp.MountSSE(r, "/sse", "/message")
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, m.HealthInfo())
	})
	return nil
}

// ServeStdio loads the corpus and runs the MCP server over stdin/stdout.
// This is the stdio transport used by Claude Code and other local MCP clients.
func (m *docsMode) ServeStdio(ctx context.Context) error {
	mcp, err := m.buildMCPServer()
	if err != nil {
		return err
	}
	return mcp.ServeStdio(ctx)
}

// HealthInfo returns mode-specific status fields for the /health endpoint.
// All fields are computed from already-loaded state; no I/O is performed.
func (m *docsMode) HealthInfo() map[string]any {
	source := "embedded"
	if m.cfg.OverridePath != "" {
		source = m.cfg.OverridePath
	}
	if m.cfg.LiveScrape {
		source = "live"
	}
	return map[string]any{
		"status":       "ok",
		"mode":         "docs",
		"docs_loaded":  true,
		"docs_source":  source,
		"live_scrape":  m.cfg.LiveScrape,
		"msfs_version": m.cfg.MSFSVersion,
		"simvar_count": m.store.SimVarCount(),
		"event_count":  m.store.EventCount(),
		"scraped_at":   m.corpusData.ScrapedAt.Format(time.RFC3339),
		"sdk_version":  m.corpusData.SDKVersion,
	}
}
