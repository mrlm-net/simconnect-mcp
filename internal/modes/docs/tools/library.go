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
		Description(fmt.Sprintf("List the guides of the mrlm-net/simconnect Go library (%s): SimConnect client and manager, "+
			"facilities, input events, client data areas, pkg/airport (ground layouts, taxi routing, SID/STAR/approach procedures), "+
			"pkg/nav (airways, weather, active runway, ATIS, flight plans) and pkg/traffic (AI traffic, departures, arrivals, "+
			"schedules, sequencing, separation). Each guide lists its chapter headings; read one with get_library_guide.", lib.Version)).
		StringParam("section", "Optional section filter: "+strings.Join(lib.Sections(), ", ")+".").
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
		Description("Read a guide of the mrlm-net/simconnect Go library as Markdown. Pass chapter to read a single \"##\" chapter "+
			"(its heading, or a unique prefix of it) instead of the whole guide — guides can be long.").
		StringParam("slug", "Guide slug from list_library_guides or search_library_docs, e.g. \"airport-layout\" (required).").
		StringParam("chapter", "Optional chapter heading, e.g. \"Taxi graph\".").
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
		Description("Search the mrlm-net/simconnect Go library guides. Every word of the query must appear in a chapter, its "+
			"heading or its guide's title (order-independent, case-insensitive). Use Go identifiers or concepts, e.g. "+
			"\"RouteToRunway\", \"holding pattern\", \"wake separation\", \"weather reader\". Returns chapters with an excerpt; "+
			"read one with get_library_guide(slug, chapter).").
		StringParam("query", "Search words (required).").
		NumberParam("limit", "Maximum results, 1–50 (default 10).").
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
