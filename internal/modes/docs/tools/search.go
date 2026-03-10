package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

func isValidSearchType(t string) bool {
	switch t {
	case "simvar", "event", "function", "structure", "error_code", "all", "":
		return true
	}
	return false
}

// RegisterSearchTool registers search_docs on s.
func RegisterSearchTool(s *mcpadapter.Server, store corpus.DocStore, liveScrape bool) {
	builder := mcpadapter.NewTool("search_docs").
		Description("Full-text search across SimConnect documentation. Returns matching SimVars, events, functions, structures, and error codes.").
		StringParam("query", "Search query (required, non-empty)").
		StringParam("type", "Corpus type to search: simvar, event, function, structure, error_code, all (default all)").
		NumberParam("limit", "Maximum results to return, max 100 (default 20)").
		Required("query")
	if liveScrape {
		builder = builder.BoolParam("confirm_live_scraping", "Set to true to confirm you accept responsibility for live HTTP requests to external documentation sites. Required when DOCS_LIVE_SCRAPE=true.")
	}
	s.AddTool(builder.Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			if guard := liveScrapeGuard(args, liveScrape); guard != nil {
				return guard, nil
			}
			query, _ := args["query"].(string)
			if strings.TrimSpace(query) == "" {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: query must not be empty"), nil
			}

			typeFilter, _ := args["type"].(string)
			if !isValidSearchType(typeFilter) {
				return mcpadapter.ErrorResult(fmt.Sprintf(
					"INVALID_ARGUMENT: unknown type %q — must be one of simvar, event, function, structure, error_code, all",
					typeFilter,
				)), nil
			}
			if typeFilter == "" {
				typeFilter = "all"
			}

			limit := intArg(args, "limit", 20)
			if limit < 1 || limit > 100 {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: limit must be 1–100"), nil
			}

			results, err := store.Search(ctx, query, typeFilter, limit)
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(results)
		},
	)
}
