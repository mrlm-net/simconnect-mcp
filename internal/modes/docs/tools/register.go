package tools

import (
	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"github.com/mrlm-net/simconnect-mcp/internal/libdocs"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterAll registers every documentation tool: the MSFS SDK reference
// (store) and the mrlm-net/simconnect library guides (lib).
func RegisterAll(s *mcpadapter.Server, store corpus.DocStore, lib *libdocs.Library, liveScrape bool) {
	RegisterSimVarTools(s, store, liveScrape)
	RegisterEventTools(s, store, liveScrape)
	RegisterFunctionTools(s, store, liveScrape)
	RegisterStructureTools(s, store, liveScrape)
	RegisterErrorCodeTools(s, store, liveScrape)
	RegisterSearchTool(s, store, liveScrape)
	RegisterLibraryTools(s, lib)
}
