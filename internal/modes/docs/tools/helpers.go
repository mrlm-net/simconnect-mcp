package tools

import "github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"

// liveScrapeGuard returns a non-nil result when live-scrape mode is active and
// the caller has not set confirm_live_scraping: true. The returned result is a
// success (not an error) that explains what the caller must do to proceed.
// Returns nil when the handler may proceed normally.
func liveScrapeGuard(args map[string]any, enabled bool) *mcpadapter.CallToolResult {
	if !enabled {
		return nil
	}
	if confirm, _ := args["confirm_live_scraping"].(bool); confirm {
		return nil
	}
	return mcpadapter.TextResult(
		"This server is running with DOCS_LIVE_SCRAPE=true, which means the documentation " +
			"corpus may be sourced from live HTTP requests to external websites. " +
			"To confirm you understand and accept responsibility for any consequences " +
			"of making live web requests, add \"confirm_live_scraping\": true to your " +
			"tool arguments and retry.",
	)
}

// intArg extracts an integer argument from an MCP args map.
// JSON numbers arrive as float64 from encoding/json; this coerces them.
// Falls back to defaultVal if the key is absent or the wrong type.
func intArg(args map[string]any, key string, defaultVal int) int {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return defaultVal
}
