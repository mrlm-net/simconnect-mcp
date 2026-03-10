package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterStructureTools registers list_structures and get_structure on s.
func RegisterStructureTools(s *mcpadapter.Server, store corpus.DocStore, liveScrape bool) {
	listBuilder := mcpadapter.NewTool("list_structures").
		Description("List SimConnect data structures. Paginate with page and page_size.").
		NumberParam("page", "Page number, 1-indexed (default 1)").
		NumberParam("page_size", "Results per page, max 100 (default 20)")
	if liveScrape {
		listBuilder = listBuilder.BoolParam("confirm_live_scraping", "Set to true to confirm you accept responsibility for live HTTP requests to external documentation sites. Required when DOCS_LIVE_SCRAPE=true.")
	}
	s.AddTool(listBuilder.Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			if guard := liveScrapeGuard(args, liveScrape); guard != nil {
				return guard, nil
			}
			page := intArg(args, "page", 1)
			pageSize := intArg(args, "page_size", 20)
			if page < 1 || pageSize < 1 || pageSize > 100 {
				return mcpadapter.ErrorResult("INVALID_PAGE: page must be ≥1 and page_size must be 1–100"), nil
			}
			result, err := store.ListStructures(ctx, page, pageSize)
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(result)
		},
	)

	getBuilder := mcpadapter.NewTool("get_structure").
		Description("Get a single SimConnect data structure by name (case-insensitive).").
		StringParam("name", "Structure name, e.g. \"SIMCONNECT_DATA_INITPOSITION\"").
		Required("name")
	if liveScrape {
		getBuilder = getBuilder.BoolParam("confirm_live_scraping", "Set to true to confirm you accept responsibility for live HTTP requests to external documentation sites. Required when DOCS_LIVE_SCRAPE=true.")
	}
	s.AddTool(getBuilder.Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			if guard := liveScrapeGuard(args, liveScrape); guard != nil {
				return guard, nil
			}
			name, _ := args["name"].(string)
			st, err := store.GetStructure(ctx, name)
			if errors.Is(err, corpus.ErrNotFound) {
				return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: structure %q not found", name)), nil
			}
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(st)
		},
	)
}
