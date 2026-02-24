package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterSimVarTools registers list_simvars and get_simvar on s.
func RegisterSimVarTools(s *mcpadapter.Server, store corpus.DocStore) {
	s.AddTool(
		mcpadapter.NewTool("list_simvars").
			Description("List SimConnect simulation variables. Optionally filter by category and paginate.").
			StringParam("category", "Filter by category (optional)").
			NumberParam("page", "Page number, 1-indexed (default 1)").
			NumberParam("page_size", "Results per page, max 100 (default 20)").
			Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
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

	s.AddTool(
		mcpadapter.NewTool("get_simvar").
			Description("Get a single SimConnect simulation variable by name (case-insensitive).").
			StringParam("name", "SimVar name, e.g. \"PLANE ALTITUDE\"").
			Required("name").
			Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
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
