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

	// corpus state — populated in Mount/ServeStdio
	store corpus.DocStore
	corp  corpus.Corpus

	// simconnect state — populated when the bridge connects
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

// buildMCPServer loads the corpus, attempts a SimConnect connection, and
// returns a fully configured MCP server. Called by Mount and ServeStdio.
func (m *bothMode) buildMCPServer(ctx context.Context) (*mcpadapter.Server, error) {
	var loader corpus.DocLoader
	if m.docsCfg.OverridePath != "" {
		loader = corpus.LoadFromPathVersion(m.docsCfg.OverridePath, m.docsCfg.MSFSVersion)
	} else {
		loader = corpus.LoadEmbeddedVersion(m.docsCfg.MSFSVersion)
	}
	c, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("both mode: load corpus: %w", err)
	}
	m.corp = c
	m.store = corpus.NewDocStore(c)

	mcp := mcpadapter.NewServer("simconnect-mcp", "1.0.0")
	doctools.RegisterSimVarTools(mcp, m.store, false)
	doctools.RegisterEventTools(mcp, m.store, false)
	doctools.RegisterFunctionTools(mcp, m.store, false)
	doctools.RegisterStructureTools(mcp, m.store, false)
	doctools.RegisterErrorCodeTools(mcp, m.store, false)
	doctools.RegisterSearchTool(mcp, m.store, false)

	b := bridge.NewSimConnectBridge()
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := b.Open(tctx, m.appName); err != nil {
		log.Printf("[both] SimConnect unavailable, running docs-only: %v", err)
	} else {
		m.br = b
		m.scReady = true
		sctools.RegisterSimVarTools(mcp, b)
		sctools.RegisterSetSimVarTool(mcp, b)
		sctools.RegisterEventTools(mcp, b)
		sctools.RegisterStateTools(mcp, b)
		sctools.RegisterTrafficTool(mcp, b)
		sctools.RegisterEnrichedTrafficTool(mcp, b)
		sctools.RegisterAirportTools(mcp, b)
	}
	return mcp, nil
}

// Mount loads the corpus, attempts a SimConnect bridge connection, registers
// all available tools onto a single MCP server, and mounts routes on r.
func (m *bothMode) Mount(r *gin.Engine) error {
	mcp, err := m.buildMCPServer(context.Background())
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

// ServeStdio runs the MCP server over stdin/stdout (stdio transport).
func (m *bothMode) ServeStdio(ctx context.Context) error {
	mcp, err := m.buildMCPServer(ctx)
	if err != nil {
		return err
	}
	return mcp.ServeStdio(ctx)
}

// HealthInfo returns merged status fields for both subsystems.
func (m *bothMode) HealthInfo() map[string]any {
	source := "embedded"
	if m.docsCfg.OverridePath != "" {
		source = m.docsCfg.OverridePath
	}

	h := map[string]any{
		"status":           "ok",
		"mode":             "both",
		"docs_loaded":      true,
		"docs_source":      source,
		"msfs_version":     m.docsCfg.MSFSVersion,
		"simvar_count":     m.store.SimVarCount(),
		"event_count":      m.store.EventCount(),
		"scraped_at":       m.corp.ScrapedAt.Format(time.RFC3339),
		"sdk_version":      m.corp.SDKVersion,
		"simconnect_ready": m.scReady,
	}

	if m.scReady && m.br != nil {
		state := m.br.State()
		h["sim_connected"] = state == bridge.StateConnected
		h["connection_state"] = state.String()
		h["app_name"] = m.appName
	}

	return h
}
