// Package corpus defines the data types for the SimConnect SDK documentation
// corpus and the ErrNotFound sentinel used by DocStore implementations.
//
// This file is the single source of truth for the corpus data contract.
// All downstream packages (DocStore, MCP tool handlers, scraper) depend on
// the types defined here. There are no external dependencies — only stdlib.
package corpus

import (
	"errors"
	"time"
)

// ── First-class domain types ──────────────────────────────────────────────────
//
// Every first-class type carries a SourceURL field as required by ADR-006.
// SourceURL holds the canonical docs.flightsimulator.com URL from which the
// item was scraped, enabling per-document attribution in MCP tool responses.
//
// Subordinate value objects (EventParam, FunctionParam, StructField) do NOT
// carry SourceURL — attribution is inherited from the enclosing first-class type.

// SimVar represents a single Simulation Variable from the SimConnect SDK.
// SimVars are readable (and optionally writable) aircraft and world state
// values that clients query via SimConnect_RequestDataOnSimObject.
type SimVar struct {
	// Name is the canonical SimVar name as used in the SDK.
	// Example: "PLANE ALTITUDE".
	Name string `json:"name"`

	// Description is the full human-readable description from the SDK docs.
	Description string `json:"description"`

	// Units lists the valid unit strings for this variable.
	// Example: ["feet", "meters"].
	// Never empty for a well-formed SimVar.
	Units []string `json:"units"`

	// Settable indicates whether SimConnect_SetDataOnSimObject can modify
	// this variable.
	Settable bool `json:"settable"`

	// Category is the SDK documentation grouping.
	// Example: "Aircraft Position and Speed".
	Category string `json:"category"`

	// Versions lists the simulator versions that define this variable.
	// Valid values: "2020", "2024".
	// Never empty for a well-formed SimVar.
	Versions []string `json:"versions"`

	// IndexedBy, when non-empty, indicates the variable accepts an integer
	// index suffix (e.g. engine number).
	// Example: "1-indexed engine number".
	IndexedBy string `json:"indexed_by,omitempty"`

	// Deprecated indicates this variable is marked deprecated in the SDK docs.
	Deprecated bool `json:"deprecated,omitempty"`

	// DeprecatedReason is set when Deprecated is true and the SDK docs
	// provide a reason or replacement variable.
	DeprecatedReason string `json:"deprecated_reason,omitempty"`

	// SourceURL is the canonical docs.flightsimulator.com URL from which
	// this SimVar was scraped. Required by ADR-006.
	SourceURL string `json:"source_url"`
}

// Event represents a SimConnect Input Event (Key Event ID).
// Events are discrete simulator actions clients trigger via
// SimConnect_TransmitClientEvent.
type Event struct {
	// Name is the canonical event name.
	// Example: "LANDING_LIGHTS_TOGGLE".
	Name string `json:"name"`

	// Description is the full human-readable description.
	Description string `json:"description"`

	// Parameters lists the positional parameters this event accepts.
	// Omitted from JSON output when the slice is empty.
	Parameters []EventParam `json:"parameters,omitempty"`

	// Versions lists the simulator versions that define this event.
	// Valid values: "2020", "2024".
	// Never empty for a well-formed Event.
	Versions []string `json:"versions"`

	// Deprecated indicates this event is superseded in the SDK docs.
	Deprecated bool `json:"deprecated,omitempty"`

	// DeprecatedReason is set when Deprecated is true and the SDK docs
	// provide a reason or replacement event.
	DeprecatedReason string `json:"deprecated_reason,omitempty"`

	// SourceURL is the canonical docs.flightsimulator.com URL from which
	// this Event was scraped. Required by ADR-006.
	SourceURL string `json:"source_url"`
}

// EventParam describes a single parameter accepted by an Input Event.
// It is a subordinate value object of Event and does not carry SourceURL.
type EventParam struct {
	// Index is the 0-based positional index of this parameter.
	Index int `json:"index"`

	// Name is the parameter name as used in the SDK docs.
	Name string `json:"name"`

	// Description is the full human-readable description of the parameter.
	Description string `json:"description"`

	// Type is the C data type of the parameter.
	// Example: "DWORD", "float".
	Type string `json:"type"`
}

// Function represents a C API function in the SimConnect SDK.
// These are the functions clients call directly against SimConnect.dll.
type Function struct {
	// Name is the C function name.
	// Example: "SimConnect_Open".
	Name string `json:"name"`

	// Description is the full human-readable description.
	Description string `json:"description"`

	// Signature is the complete C function signature as a string.
	// Example: "SIMCONNECTAPI SimConnect_Open(HANDLE * phSimConnect, ...)".
	Signature string `json:"signature"`

	// Parameters lists each parameter with its name, type, direction, and
	// description. The slice is always present even if empty (never nil).
	Parameters []FunctionParam `json:"parameters"`

	// ReturnType is the C return type.
	// Example: "HRESULT".
	ReturnType string `json:"return_type"`

	// Remarks contains additional SDK usage notes.
	// Omitted from JSON output when empty.
	Remarks string `json:"remarks,omitempty"`

	// SourceURL is the canonical docs.flightsimulator.com URL from which
	// this Function was scraped. Required by ADR-006.
	SourceURL string `json:"source_url"`
}

// FunctionParam describes one parameter of a C API function.
// It is a subordinate value object of Function and does not carry SourceURL.
type FunctionParam struct {
	// Name is the C parameter name.
	Name string `json:"name"`

	// Type is the C data type.
	// Example: "HANDLE *", "DWORD", "const char *".
	Type string `json:"type"`

	// Direction indicates data flow relative to the caller.
	// Valid values: "in", "out", "in/out".
	Direction string `json:"direction"`

	// Description is the full human-readable description of the parameter.
	Description string `json:"description"`

	// Optional is true when the SDK documents this as an optional trailing
	// parameter (typically a reserved value that may be omitted or set to 0).
	Optional bool `json:"optional,omitempty"`
}

// Structure represents a C data structure defined by the SimConnect SDK.
// Clients receive or pass pointers to these structures via the SimConnect API.
type Structure struct {
	// Name is the C struct or union name.
	// Example: "SIMCONNECT_RECV", "SIMCONNECT_DATA_LATLONALT".
	Name string `json:"name"`

	// Description is the full human-readable description.
	Description string `json:"description"`

	// Fields lists each member field.
	// The slice is always present even if empty (never nil).
	Fields []StructField `json:"fields"`

	// Remarks contains additional SDK usage notes.
	// Omitted from JSON output when empty.
	Remarks string `json:"remarks,omitempty"`

	// SourceURL is the canonical docs.flightsimulator.com URL from which
	// this Structure was scraped. Required by ADR-006.
	SourceURL string `json:"source_url"`
}

// StructField describes one member field of a C structure.
// It is a subordinate value object of Structure and does not carry SourceURL.
type StructField struct {
	// Name is the C field name.
	Name string `json:"name"`

	// Type is the C data type of the field.
	// Example: "DWORD", "double", "char[256]".
	Type string `json:"type"`

	// Description is the full human-readable description of the field.
	Description string `json:"description"`
}

// ErrorCode represents one value of the SIMCONNECT_EXCEPTION enumeration.
// The simulator sends these codes in SIMCONNECT_RECV_EXCEPTION packets to
// indicate which client request triggered an error.
type ErrorCode struct {
	// Name is the enum member name.
	// Example: "SIMCONNECT_EXCEPTION_NONE", "SIMCONNECT_EXCEPTION_ERROR".
	Name string `json:"name"`

	// Value is the integer ordinal of the enum member.
	// Declared as int (not uint32) because the SDK documentation presents
	// these as small non-negative integers and JSON unmarshalling handles
	// int cleanly without overflow risk at the documented scale.
	Value int `json:"value"`

	// Description is the full human-readable description.
	Description string `json:"description"`

	// SourceURL is the canonical docs.flightsimulator.com URL from which
	// this ErrorCode was scraped. Required by ADR-006.
	SourceURL string `json:"source_url"`
}

// ── Corpus aggregate ──────────────────────────────────────────────────────────

// Corpus is the top-level container for all loaded SimConnect SDK documentation.
// The corpus loaders (LoadEmbedded, LoadFromPath) deserialise JSON into this
// structure. DocStore implementations index the slices for efficient lookup.
//
// Version-specific collections (SimVars, Events) are merged by the DocStore
// when DOCS_MSFS_VERSION is "both"; the Versions field on each item indicates
// which simulators define it.
type Corpus struct {
	// ScrapedAt records when this corpus snapshot was generated by the scraper.
	// Used by the /health endpoint and for cache-staleness checks.
	ScrapedAt time.Time `json:"scraped_at"`

	// SDKVersion is the SDK version string at the time of scraping.
	// Example: "0.21.1" (MSFS 2020), "1.2.0" (MSFS 2024).
	SDKVersion string `json:"sdk_version"`

	// SimVars contains all Simulation Variables in this corpus snapshot.
	SimVars []SimVar `json:"simvars"`

	// Events contains all Input Events in this corpus snapshot.
	Events []Event `json:"events"`

	// Functions contains all C API functions. Version-agnostic.
	Functions []Function `json:"functions"`

	// Structures contains all C data structures. Version-agnostic.
	Structures []Structure `json:"structures"`

	// ErrorCodes contains all SIMCONNECT_EXCEPTION enum values. Version-agnostic.
	ErrorCodes []ErrorCode `json:"error_codes"`
}

// ── Search types ──────────────────────────────────────────────────────────────

// SearchResult is a single hit returned by DocStore.Search.
// SourceURL is included so MCP tool responses can surface attribution links
// directly in search results without a follow-up Get* call.
type SearchResult struct {
	// Type identifies the corpus category of this hit.
	// Valid values: "simvar", "event", "function", "structure", "error_code".
	Type string `json:"type"`

	// Name is the item's canonical name as it appears in the SDK.
	Name string `json:"name"`

	// Excerpt is a short snippet from the description providing query context.
	// Typically 120-160 characters, trimmed at a word boundary.
	Excerpt string `json:"excerpt"`

	// SourceURL is the canonical docs.flightsimulator.com URL for this item.
	// Required by ADR-006 for attribution in tool responses.
	SourceURL string `json:"source_url"`
}

// SearchResults groups all hits returned by a single DocStore.Search call.
type SearchResults struct {
	// Query is the search string as provided by the caller.
	Query string `json:"query"`

	// Total is the number of results returned in this response.
	// Equal to len(Results). Not a global match count — the caller supplies
	// a limit parameter to DocStore.Search.
	Total int `json:"total"`

	// Results contains the ordered search hits, most relevant first.
	Results []SearchResult `json:"results"`
}

// ── Pagination ────────────────────────────────────────────────────────────────

// Page is a generic pagination envelope returned by MCP tool handlers for
// all List* operations. DocStore.List* methods return ([]T, int, error);
// the tool handler wraps the slice into a Page[T] before serialising.
//
// Page is defined in types.go (not store.go) because it is part of the
// public data contract consumed by MCP tool handlers and test fixtures,
// not an implementation detail of the DocStore.
type Page[T any] struct {
	// Items contains the records for this page.
	Items []T `json:"items"`

	// Page is the 1-indexed current page number.
	Page int `json:"page"`

	// PageSize is the maximum number of items per page as requested.
	PageSize int `json:"page_size"`

	// TotalItems is the total number of items across all pages.
	TotalItems int `json:"total_items"`

	// TotalPages is the total number of pages given TotalItems and PageSize.
	// Computed as ceil(TotalItems / PageSize). Zero when TotalItems is zero.
	TotalPages int `json:"total_pages"`
}

// ── Sentinels ─────────────────────────────────────────────────────────────────

// ErrNotFound is returned by all DocStore.Get* methods when no item matches
// the requested name. Callers must use errors.Is(err, ErrNotFound) for
// comparison — never compare the error string directly.
//
// Defined in types.go rather than store.go so that packages importing only
// the types (e.g. tool handlers that need to check the error without
// importing the full store) do not need an additional import.
var ErrNotFound = errors.New("not found")
