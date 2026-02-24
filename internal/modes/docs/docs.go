// Package docs implements the documentation-fetch mode (Milestone 1).
// It exposes SimConnect SDK documentation via MCP tool endpoints,
// allowing AI agents to query reference material without a running simulator.
package docs

import (
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
// The corpus is not loaded until Mount is called.
func New(cfg Config) modes.Mode {
	return &docsMode{cfg: cfg}
}

// Mount loads the corpus, wires up the MCP server with all tool registrations,
// and registers the /mcp, /sse, /message, and /health routes on r.
// It returns a non-nil error when the corpus cannot be loaded; in that case
// no routes are registered and the caller must handle the failure.
func (m *docsMode) Mount(r *gin.Engine) error {
	// 1. Select loader based on whether an override path is configured.
	// Pass MSFSVersion explicitly so the corpus respects the config value
	// rather than re-reading the env var independently.
	var loader corpus.DocLoader
	if m.cfg.OverridePath != "" {
		loader = corpus.LoadFromPathVersion(m.cfg.OverridePath, m.cfg.MSFSVersion)
	} else {
		loader = corpus.LoadEmbeddedVersion(m.cfg.MSFSVersion)
	}

	// 2. Load the corpus; fail fast so the caller receives a clear error.
	c, err := loader.Load()
	if err != nil {
		return fmt.Errorf("docs mode: load corpus: %w", err)
	}
	m.corpusData = c

	// 3. Build the in-memory store from the loaded corpus.
	m.store = corpus.NewDocStore(c)

	// 4. Create the MCP server.
	mcp := mcpadapter.NewServer("simconnect-mcp-docs", "1.0.0")

	// 5. Register all MCP tool groups.
	tools.RegisterSimVarTools(mcp, m.store)
	tools.RegisterEventTools(mcp, m.store)
	tools.RegisterFunctionTools(mcp, m.store)
	tools.RegisterStructureTools(mcp, m.store)
	tools.RegisterErrorCodeTools(mcp, m.store)
	tools.RegisterSearchTool(mcp, m.store)

	// 6. Mount MCP transports onto the Gin engine.
	mcp.MountStreamableHTTP(r, "/mcp")
	mcp.MountSSE(r, "/sse", "/message")

	// 7. Register the health endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, m.HealthInfo())
	})

	return nil
}

// HealthInfo returns mode-specific status fields for the /health endpoint.
// All fields are computed from already-loaded state; no I/O is performed.
func (m *docsMode) HealthInfo() map[string]any {
	source := "embedded"
	if m.cfg.OverridePath != "" {
		source = m.cfg.OverridePath
	}
	return map[string]any{
		"status":       "ok",
		"mode":         "docs",
		"docs_loaded":  true,
		"docs_source":  source,
		"msfs_version": m.cfg.MSFSVersion,
		"simvar_count": m.store.SimVarCount(),
		"event_count":  m.store.EventCount(),
		"scraped_at":   m.corpusData.ScrapedAt.Format(time.RFC3339),
		"sdk_version":  m.corpusData.SDKVersion,
	}
}
