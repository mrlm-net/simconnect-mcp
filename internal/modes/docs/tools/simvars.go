package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterSimVarTools registers list_simvar_categories, list_simvars, and get_simvar on s.
func RegisterSimVarTools(s *mcpadapter.Server, store corpus.DocStore, liveScrape bool) {
	catBuilder := mcpadapter.NewTool("list_simvar_categories").
		Description("List all available SimVar category filter values. " +
			"Use these exact strings as the category parameter for list_simvars. " +
			"Note: some categories contain '/' without spaces (e.g. 'AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES') " +
			"and others use ' / ' with spaces (e.g. 'AIRCRAFT BRAKE / LANDING GEAR VARIABLES').")
	if liveScrape {
		catBuilder = catBuilder.BoolParam("confirm_live_scraping", "Set to true to confirm you accept responsibility for live HTTP requests to external documentation sites. Required when DOCS_LIVE_SCRAPE=true.")
	}
	s.AddTool(catBuilder.Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			if guard := liveScrapeGuard(args, liveScrape); guard != nil {
				return guard, nil
			}
			cats, err := store.ListSimVarCategories(ctx)
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(map[string]any{
				"total":      len(cats),
				"categories": cats,
			})
		},
	)


	listBuilder := mcpadapter.NewTool("list_simvars").
		Description("List SimConnect simulation variables. Optionally filter by category and paginate.").
		StringParam("category", "Filter by category (optional)").
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
			category, _ := args["category"].(string)
			page := intArg(args, "page", 1)
			pageSize := intArg(args, "page_size", 20)
			if page < 1 || pageSize < 1 || pageSize > 100 {
				return mcpadapter.ErrorResult("INVALID_PAGE: page must be ≥1 and page_size must be 1–100"), nil
			}
			result, err := store.ListSimVars(ctx, category, page, pageSize)
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(result)
		},
	)

	getBuilder := mcpadapter.NewTool("get_simvar").
		Description("Get a single SimConnect simulation variable by name (case-insensitive).").
		StringParam("name", "SimVar name, e.g. \"PLANE ALTITUDE\"").
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
			sv, err := store.GetSimVar(ctx, name)
			if errors.Is(err, corpus.ErrNotFound) {
				return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: simvar %q not found", name)), nil
			}
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(sv)
		},
	)
}
