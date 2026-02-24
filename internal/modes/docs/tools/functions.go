package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterFunctionTools registers list_functions and get_function on s.
func RegisterFunctionTools(s *mcpadapter.Server, store corpus.DocStore) {
	s.AddTool(
		mcpadapter.NewTool("list_functions").
			Description("List SimConnect API functions. Paginate with page and page_size.").
			NumberParam("page", "Page number, 1-indexed (default 1)").
			NumberParam("page_size", "Results per page, max 100 (default 20)").
			Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			page := intArg(args, "page", 1)
			pageSize := intArg(args, "page_size", 20)
			if page < 1 || pageSize < 1 || pageSize > 100 {
				return mcpadapter.ErrorResult("INVALID_PAGE: page must be ≥1 and page_size must be 1–100"), nil
			}
			result, err := store.ListFunctions(ctx, page, pageSize)
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(result)
		},
	)

	s.AddTool(
		mcpadapter.NewTool("get_function").
			Description("Get a single SimConnect API function by name (case-insensitive).").
			StringParam("name", "Function name, e.g. \"SimConnect_Open\"").
			Required("name").
			Build(),
		func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
			name, _ := args["name"].(string)
			fn, err := store.GetFunction(ctx, name)
			if errors.Is(err, corpus.ErrNotFound) {
				return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: function %q not found", name)), nil
			}
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("INTERNAL_ERROR: %v", err)), nil
			}
			return mcpadapter.JSONResult(fn)
		},
	)
}
