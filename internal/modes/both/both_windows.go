//go:build windows

// Package both implements the combined docs+simconnect mode.
// On Windows, all docs tools and all SimConnect live-data tools are registered
// onto a single MCP server. If the SimConnect bridge cannot connect at startup
// (simulator not running, DLL missing) the server falls back to docs-only and
// logs a warning — the SimConnect tools are simply not registered.
package both

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect-mcp/internal/modes"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/docs"
	doctools "github.com/mrlm-net/simconnect-mcp/internal/modes/docs/tools"
	"github.com/mrlm-net/simconnect-mcp/internal/modes/simconnect"
	sctools "github.com/mrlm-net/simconnect-mcp/internal/modes/simconnect/tools"
)

// bothMode holds state for the combined docs+simconnect operating mode.
type bothMode struct {
	docsCfg docs.Config
	appName string

	// corpus state — populated in Mount
	store corpus.DocStore
	corp  corpus.Corpus

	// simconnect state — populated in Mount only when the bridge connects
	br      bridge.Bridge
	scReady bool
}

// Ensure bothMode satisfies modes.Mode at compile time.
var _ modes.Mode = (*bothMode)(nil)

// New returns a combined Mode for Windows.
// Docs config is read from DOCS_* and PORT env vars.
// SimConnect app name is read from SIMCONNECT_APP_NAME (default "simconnect-mcp").
func New() (modes.Mode, string, error) {
	dcfg := docs.ConfigFromEnv()
	sccfg := simconnect.ConfigFromEnv()
	return &bothMode{
		docsCfg: dcfg,
		appName: sccfg.AppName,
	}, dcfg.ListenAddr, nil
}

// Mount loads the corpus, attempts a SimConnect bridge connection, registers
// all available tools onto a single MCP server, and mounts routes on r.
// A SimConnect connection failure is non-fatal: the server starts docs-only
// and logs a warning.
func (m *bothMode) Mount(r *gin.Engine) error {
	// 1. Load corpus for docs tools.
	var loader corpus.DocLoader
	if m.docsCfg.OverridePath != "" {
		loader = corpus.LoadFromPathVersion(m.docsCfg.OverridePath, m.docsCfg.MSFSVersion)
	} else {
		loader = corpus.LoadEmbeddedVersion(m.docsCfg.MSFSVersion)
	}
	c, err := loader.Load()
	if err != nil {
		return fmt.Errorf("both mode: load corpus: %w", err)
	}
	m.corp = c
	m.store = corpus.NewDocStore(c)

	// 2. Single MCP server for all tools.
	mcp := mcpadapter.NewServer("simconnect-mcp", "1.0.0")

	// 3. Register docs tools — always present.
	doctools.RegisterSimVarTools(mcp, m.store)
	doctools.RegisterEventTools(mcp, m.store)
	doctools.RegisterFunctionTools(mcp, m.store)
	doctools.RegisterStructureTools(mcp, m.store)
	doctools.RegisterErrorCodeTools(mcp, m.store)
	doctools.RegisterSearchTool(mcp, m.store)

	// 4. Attempt SimConnect bridge — optional, non-fatal.
	b := bridge.NewSimConnectBridge()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := b.Open(ctx, m.appName); err != nil {
		log.Printf("[both] SimConnect unavailable, running docs-only: %v", err)
	} else {
		m.br = b
		m.scReady = true
		sctools.RegisterSimVarTools(mcp, b)
		sctools.RegisterEventTools(mcp, b)
		sctools.RegisterStateTools(mcp, b)
	}

	// 5. Mount MCP transports once.
	mcp.MountStreamableHTTP(r, "/mcp")
	mcp.MountSSE(r, "/sse", "/message")

	// 6. Health endpoint.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, m.HealthInfo())
	})

	return nil
}

// HealthInfo returns merged status fields for both subsystems.
func (m *bothMode) HealthInfo() map[string]any {
	source := "embedded"
	if m.docsCfg.OverridePath != "" {
		source = m.docsCfg.OverridePath
	}

	h := map[string]any{
		"status":              "ok",
		"mode":                "both",
		"docs_loaded":         true,
		"docs_source":         source,
		"msfs_version":        m.docsCfg.MSFSVersion,
		"simvar_count":        m.store.SimVarCount(),
		"event_count":         m.store.EventCount(),
		"scraped_at":          m.corp.ScrapedAt.Format(time.RFC3339),
		"sdk_version":         m.corp.SDKVersion,
		"simconnect_ready":    m.scReady,
	}

	if m.scReady && m.br != nil {
		state := m.br.State()
		h["sim_connected"] = state == bridge.StateConnected
		h["connection_state"] = state.String()
		h["app_name"] = m.appName
	}

	return h
}
