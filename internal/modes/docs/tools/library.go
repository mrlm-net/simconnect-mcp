package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/libdocs"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterLibraryTools registers list_library_guides, get_library_guide and
// search_library_docs: the guides of the mrlm-net/simconnect Go library.
func RegisterLibraryTools(s *mcpadapter.Server, lib *libdocs.Library) {
	registerListLibraryGuides(s, lib)
	registerGetLibraryGuide(s, lib)
	registerSearchLibraryDocs(s, lib)
}

func registerListLibraryGuides(s *mcpadapter.Server, lib *libdocs.Library) {
	tool := mcpadapter.NewTool("list_library_guides").
		Description(fmt.Sprintf("List the guides of the mrlm-net/simconnect Go library (%s) with their chapters "+
			"(client, manager, pkg/airport, pkg/nav, pkg/traffic and more); read one with get_library_guide.", lib.Version)).
		StringParam("section", strings.Join(lib.Sections(), ", ")).
		Build()

	s.AddTool(tool, func(_ context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		section, _ := args["section"].(string)
		type entry struct {
			libdocs.Guide
			Chapters []string `json:"chapters"`
		}
		guides := lib.Guides(section)
		if len(guides) == 0 {
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: no guides in section %q — sections are %s",
				section, strings.Join(lib.Sections(), ", "))), nil
		}
		out := make([]entry, 0, len(guides))
		for _, g := range guides {
			out = append(out, entry{Guide: *g, Chapters: g.Headings()})
		}
		return mcpadapter.JSONResult(map[string]any{
			"library": "github.com/mrlm-net/simconnect",
			"version": lib.Version,
			"total":   len(out),
			"guides":  out,
		})
	})
}

func registerGetLibraryGuide(s *mcpadapter.Server, lib *libdocs.Library) {
	tool := mcpadapter.NewTool("get_library_guide").
		Description("Read a guide of the mrlm-net/simconnect Go library as Markdown, whole or one \"##\" chapter.").
		StringParam("slug", "Guide slug, e.g. \"airport-layout\"").
		StringParam("chapter", "Chapter heading or unique prefix").
		Required("slug").
		Build()

	s.AddTool(tool, func(_ context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		slug, _ := args["slug"].(string)
		g, ok := lib.Guide(slug)
		if !ok {
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: no guide %q — use list_library_guides", slug)), nil
		}
		chapter, _ := args["chapter"].(string)
		if strings.TrimSpace(chapter) == "" {
			return mcpadapter.TextResult("# " + g.Title + "\n\n" + g.Body), nil
		}
		c, ok := g.Chapter(chapter)
		if !ok {
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: guide %q has no chapter %q — chapters are: %s",
				g.Slug, chapter, strings.Join(g.Headings(), "; "))), nil
		}
		return mcpadapter.TextResult("# " + g.Title + "\n\n## " + c.Heading + "\n\n" + c.Text), nil
	})
}

func registerSearchLibraryDocs(s *mcpadapter.Server, lib *libdocs.Library) {
	tool := mcpadapter.NewTool("search_library_docs").
		Description("Search the mrlm-net/simconnect library guides by Go identifiers or concepts (all words must match); "+
			"returns chapters with an excerpt.").
		StringParam("query", "Search words").
		NumberParam("limit", "1–50 (default 10)").
		Required("query").
		Build()

	s.AddTool(tool, func(_ context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		query, _ := args["query"].(string)
		if strings.TrimSpace(query) == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: query must not be empty"), nil
		}
		limit := intArg(args, "limit", 10)
		if limit < 1 || limit > 50 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: limit must be 1–50"), nil
		}
		hits := lib.Search(query, limit)
		return mcpadapter.JSONResult(map[string]any{
			"query":   query,
			"total":   len(hits),
			"results": hits,
		})
	})
}
