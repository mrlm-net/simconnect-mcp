package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterErrorCodeTools registers list_error_codes and get_error_code on s.
func RegisterErrorCodeTools(s *mcpadapter.Server, store corpus.DocStore, liveScrape bool) {
	listBuilder := mcpadapter.NewTool("list_error_codes").
		Description("List SimConnect exception/error codes. Paginate with page and page_size.").
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
			result, err := store.ListErrorCodes(ctx, page, pageSize)
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(result)
		},
	)

	getBuilder := mcpadapter.NewTool("get_error_code").
		Description("Get a SimConnect error code by name (e.g. \"SIMCONNECT_EXCEPTION_NONE\") or integer value (e.g. 0).").
		StringParam("name", "Error code name (optional if value provided)").
		NumberParam("value", "Integer error code value (optional if name provided)")
	if liveScrape {
		getBuilder = getBuilder.BoolParam("confirm_live_scraping", "Set to true to confirm you accept responsibility for live HTTP requests to external documentation sites. Required when DOCS_LIVE_SCRAPE=true.")
	}
	s.AddTool(getBuilder.Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			if guard := liveScrapeGuard(args, liveScrape); guard != nil {
				return guard, nil
			}
			name, hasName := args["name"].(string)
			_, hasValue := args["value"]

			if !hasName && !hasValue {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: provide name or value"), nil
			}

			// Try lookup by name first.
			if hasName && strings.TrimSpace(name) != "" {
				ec, err := store.GetErrorCode(ctx, name)
				if err == nil {
					return mcpadapter.JSONResult(ec)
				}
				if !errors.Is(err, corpus.ErrNotFound) {
					return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
				}
			}

			// Fall back to value scan.
			if hasValue {
				targetVal := intArg(args, "value", -1)
				page := 1
				for {
					p, err := store.ListErrorCodes(ctx, page, 100)
					if err != nil {
						return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
					}
					for _, ec := range p.Items {
						if ec.Value == targetVal {
							return mcpadapter.JSONResult(ec)
						}
					}
					if page >= p.TotalPages {
						break
					}
					page++
				}
			}

			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: error code %q not found", name)), nil
		},
	)
}
