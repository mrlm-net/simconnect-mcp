# Technical Architecture — Milestone 1: SimConnect MCP Documentation Mode

**Date:** 2026-02-22
**Status:** Approved
**Branch:** `milestone/1-docs`
**Module:** `github.com/mrlm-net/simconnect-mcp`

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [ADR-001: MCP Library Selection](#2-adr-001-mcp-library-selection)
3. [ADR-002: MCP Transport](#3-adr-002-mcp-transport)
4. [ADR-003: Documentation Source Strategy](#4-adr-003-documentation-source-strategy)
5. [ADR-004: HTTP Framework Topology](#5-adr-004-http-framework-topology)
6. [ADR-005: MSFS Version Strategy](#6-adr-005-msfs-version-strategy)
7. [Component Breakdown](#7-component-breakdown)
8. [Interface Definitions](#8-interface-definitions)
9. [Data Schema](#9-data-schema)
10. [Dependency Graph](#10-dependency-graph)
11. [MCP Tool Catalogue](#11-mcp-tool-catalogue)
12. [Error Handling Contract](#12-error-handling-contract)
13. [Configuration Contract](#13-configuration-contract)
14. [Implementation Guidelines](#14-implementation-guidelines)

---

## 1. System Overview

The docs mode server exposes the SimConnect SDK reference corpus (SimVars, Input
Events, C API functions, data structures, error codes) as MCP tools. An AI agent
connects once, discovers tools via `tools/list`, then queries the corpus through
typed tool calls. No simulator connection is required.

### C4 Container Diagram

```
+------------------------------------------------------------+
|  AI Agent (Claude Desktop, Cursor, custom MCP client)      |
|  Transport: Streamable HTTP  POST /mcp  GET /mcp           |
+-----------------------------+------------------------------+
                              |
                              v
+------------------------------------------------------------+
|  simconnect-mcp binary  (MCP_MODE=docs)                    |
|                                                            |
|  +-------------------+   +----------------------------+   |
|  |  Gin HTTP layer   |   |  mcp-go MCPServer          |   |
|  |  GET /health      |   |  mounted via gin.WrapH     |   |
|  |  POST /mcp        |   |  at /mcp                   |   |
|  |  GET  /mcp        |   |                            |   |
|  +-------------------+   +----------------------------+   |
|                                    |                       |
|                           +--------v--------+             |
|                           |   DocStore      |             |
|                           |  (in-memory)    |             |
|                           +-----------------+             |
|                                    |                       |
|                           +--------v--------+             |
|                           | Embedded JSON   |             |
|                           | corpus assets   |             |
|                           | (//go:embed)    |             |
|                           +-----------------+             |
+------------------------------------------------------------+
```

### Startup Sequence

```
main()
  |
  +--> Read MCP_MODE env var (default: "docs")
  |
  +--> docs.New(cfg)
  |      |
  |      +--> corpus.LoadEmbedded()     <- reads //go:embed assets
  |      |      |
  |      |      +--> parse JSON files into DocStore
  |      |
  |      +--> tools.Register(mcpServer, store)
  |      |      |
  |      |      +--> registers 10 MCP tools
  |      |
  |      +--> returns (gin.IRoutes, error)
  |
  +--> server.New()  <- Gin engine
  |
  +--> mode.Mount(router)   <- mounts /health + WrapH(/mcp)
  |
  +--> router.Run(":8080")
```

---

## 2. ADR-001: MCP Library Selection

**Status:** Accepted

### Context

Two Go MCP libraries are available:

| Property | mark3labs/mcp-go v0.44.0 | modelcontextprotocol/go-sdk v1.3.1 |
|---|---|---|
| Governance | Community (mark3labs) | Official SDK (Anthropic/MCP org) |
| API stability | Pre-v1, breaking changes possible | v1.x — stable API contract |
| Transports | stdio, SSE, Streamable HTTP | stdio, custom via jsonrpc pkg |
| HTTP handler | `ServeHTTP` on both SSE+StreamableHTTP servers | Not natively exposed |
| `tools/list` | Built-in | Built-in |
| Spec version | 2025-11-25 | 2024-11-05 through 2025-06-18 |
| Gin integration | `gin.WrapH(server.ServeHTTP)` — straightforward | Requires custom bridge |
| Community adoption | Widely used in Go MCP ecosystem | Early adoption stage |
| Test helpers | `NewTestServer`, `NewTestStreamableHTTPServer` | Less tooling |

### Decision

**Use `github.com/mark3labs/mcp-go`.**

### Rationale

The critical integration requirement is mounting MCP under Gin's `/mcp` route using
`gin.WrapH`. mcp-go's `StreamableHTTPServer` and `SSEServer` both implement
`http.Handler` natively, making this a single-line mount. The official go-sdk exposes
no HTTP handler at all in its current form — a custom JSON-RPC transport bridge would
need to be written, adding hundreds of lines of protocol code outside the project's
scope.

mcp-go is pre-v1 but has been stable across 44 minor releases with high adoption in
the Go MCP community. Its API churn is low at the `server.NewMCPServer` and `AddTool`
level, which is the only surface area this project touches. The official SDK's v1.x
stability designation is less useful here because it lacks the HTTP transport we need.

The pre-v1 risk is mitigated by: pinning to a specific version in `go.mod`, writing a
thin adapter layer (`internal/mcpadapter`) so the mcp-go API surface is contained, and
monitoring for breaking changes on each dependency update.

### Consequences

- Add `github.com/mark3labs/mcp-go` to `go.mod`.
- Wrap mcp-go registration behind `internal/mcpadapter` to isolate the dependency.
- If mcp-go breaks its API before v1, the adapter is the only file requiring updates.
- The official SDK is monitored; if it ships native HTTP transport, re-evaluate at
  Milestone 2 kickoff.

---

## 3. ADR-002: MCP Transport

**Status:** Accepted

### Context

The MCP specification (2025-03-26) defines two standard transports: **stdio** and
**Streamable HTTP**. The older HTTP+SSE transport (spec 2024-11-05) is explicitly
deprecated. mcp-go supports all three as distinct server types.

Key constraints:

- The server runs as a long-lived process accessible to multiple clients (not a
  subprocess per-client model, which stdio requires).
- AI agent clients (Claude Desktop, Cursor) are moving to Streamable HTTP.
- Some deployed MCP clients still use the legacy SSE transport.
- The `/health` endpoint and potential future admin endpoints require an HTTP server
  regardless.

### Decision

**Primary transport: Streamable HTTP** (`POST /mcp`, `GET /mcp`).
**Compatibility alias: legacy SSE** (`GET /sse`, `POST /message`) served alongside.
**Stdio: not implemented** in Milestone 1.

### Implementation

```
GET  /health          -> Gin handler (health check)
POST /mcp             -> gin.WrapH(streamableHTTPServer)
GET  /mcp             -> gin.WrapH(streamableHTTPServer)
GET  /sse             -> gin.WrapH(sseServer)     [compatibility]
POST /message         -> gin.WrapH(sseServer)     [compatibility]
```

Both servers share the same underlying `*server.MCPServer` instance. They are
distinct transport wrappers over the same tool registry.

### Rationale

Streamable HTTP is the current spec and the direction all compliant clients are moving.
Clients that send `POST /mcp` with `Accept: application/json, text/event-stream`
negotiate the response format per-request, which is more flexible than the SSE-only
model.

The legacy SSE compatibility alias costs essentially nothing — same `MCPServer`,
different transport wrapper, two additional Gin routes. Removing it prematurely would
break users with older clients. The alias carries a deprecation notice in the OpenAPI
description returned at `GET /mcp` capability negotiation.

Stdio is omitted because this server is designed for multi-client access over HTTP, not
subprocess-per-client invocation.

### Consequences

- `internal/modes/docs/docs.go` creates one `*server.MCPServer`, wraps it in both
  `server.NewStreamableHTTPServer` and `server.NewSSEServer`, then mounts both.
- Session management is handled by mcp-go's built-in session ID manager.
- Origin header validation is the responsibility of Gin middleware (`internal/server`),
  not the MCP transport layer.

---

## 4. ADR-003: Documentation Source Strategy

**Status:** Accepted

### Context

The SimConnect documentation exists at `docs.flightsimulator.com`. Three loading
strategies were evaluated:

| Strategy | Build-time scrape + embed | Startup-time fetch (network) | Startup-time fetch (local file) |
|---|---|---|---|
| Offline reliability | Guaranteed | None | Operator-dependent |
| Startup latency | Zero (in-binary) | 5-30 s first boot | <1 s |
| Freshness | Manual update cycle | Always current | Manual update cycle |
| Binary size | +5-20 MB | Minimal | Minimal |
| CI dependency | Scraper must run in CI | None | None |
| ToS risk | Scraping at build time | Scraping at runtime | None |
| Operator complexity | None | None | Must supply files |

### Decision

**Build-time scraper producing embedded JSON, with optional runtime override via
environment variable.**

Specifically:
1. A `tools/scraper` CLI (not part of the server binary) scrapes
   `docs.flightsimulator.com` at release time and writes structured JSON files to
   `internal/corpus/assets/`.
2. Those JSON files are embedded into the server binary at compile time using
   `//go:embed`.
3. At startup, if `DOCS_OVERRIDE_PATH` env var is set, the server loads from that
   filesystem path instead of the embedded assets. This supports local development,
   testing, and operator customisation.

### Scraper ToS Note

`docs.flightsimulator.com` is operated by Asobo/Microsoft. The scraper must:
- Respect `robots.txt`
- Rate-limit requests (minimum 1 second between requests)
- Not run in CI on every push — only on tagged corpus-update releases
- Cache responses locally during development

This risk is classified Medium. The scraper is a development tool, not a production
runtime component. Legal review is flagged as a prerequisite before the first corpus
update release.

### Corpus File Layout

```
internal/corpus/
  assets/
    simvars_2020.json
    simvars_2024.json
    events_2020.json
    events_2024.json
    functions.json
    structures.json
    error_codes.json
  embed.go          <- //go:embed assets/*.json
  loader.go         <- LoadEmbedded(), LoadFromPath()
  store.go          <- DocStore implementation
  store_test.go

tools/scraper/      <- build-time CLI, NOT compiled into server binary
  main.go
  scraper.go
  parser.go
```

### Rationale

Embedding guarantees the server starts offline, in airgapped environments, and in
Docker containers without outbound network access. The 5-20 MB binary size increase is
acceptable for a developer tool. The override mechanism retains flexibility for
operators who maintain their own corpus files.

### Consequences

- `go generate ./internal/corpus/...` invokes the scraper to refresh assets.
- The scraper is in `tools/scraper/` and excluded from normal `go build ./...` via
  package isolation (not in `cmd/` or `internal/`).
- Unit tests embed a miniature fixture corpus; they never depend on the embedded
  production assets or network.

---

## 5. ADR-004: HTTP Framework Topology

**Status:** Accepted

### Context

mcp-go's transport servers implement `http.Handler`. There are three ways to integrate
them with the existing Gin setup:

| Option | Description | Pros | Cons |
|---|---|---|---|
| A: Gin owns all routes | Mount MCP via `gin.WrapH` under `/mcp` | Single HTTP server, Gin middleware applies everywhere | Gin wraps MCP; small overhead |
| B: Split servers | MCP runs its own `net/http` server on port 8081; Gin on 8080 | Cleanest separation | Two ports, harder deployment, no shared middleware |
| C: Replace Gin with net/http | Remove Gin entirely; plain `net/http` mux | Fewer dependencies | Lose Gin middleware ecosystem; more boilerplate |

### Decision

**Option A: Gin owns all routes. MCP transport servers mounted via `gin.WrapH`.**

### Rationale

Gin is already the project's HTTP framework. `gin.WrapH` accepts any `http.Handler`
and mounts it at a prefix — this is an established, idiomatic pattern. The existing
`/health` endpoint stays as a Gin handler and benefits from Gin's response helpers.
Future endpoints (admin, metrics) follow the same pattern without additional
infrastructure.

Option B introduces operational complexity (two ports, two health checks, split
middleware) for no gain at Milestone 1 scale. Option C removes infrastructure that
already works and provides request logging, recovery middleware, and structured JSON
helpers that are explicitly required by the conventions in `CLAUDE.md`.

### Route Map

```go
// internal/modes/docs/docs.go

r.GET("/health", healthHandler)
r.Any("/mcp", gin.WrapH(streamableHTTPServer))
r.GET("/sse", gin.WrapH(sseServer))          // legacy compat
r.POST("/message", gin.WrapH(sseServer))     // legacy compat
```

`gin.Any` is used for `/mcp` to capture both POST and GET in a single line.
Internally, `StreamableHTTPServer.ServeHTTP` delegates on method.

### Middleware Stack

Applied globally via `server.New()`:

1. `gin.Recovery()` — panic recovery, returns 500 JSON error
2. `gin.Logger()` — structured access log
3. `middleware.RequestID()` — injects `X-Request-ID` header (custom, to be
   implemented in `internal/server/middleware/`)
4. `middleware.CORS()` — permissive for localhost origins in development (custom)

Origin validation required by the MCP spec is handled at the Gin middleware layer,
not inside mcp-go, giving consistent enforcement across all routes.

### Consequences

- `internal/server/server.go` is extended to accept middleware options.
- `internal/server/middleware/` is a new package for shared middleware.
- The Gin import remains in `go.mod` and is used by both modes.

---

## 6. ADR-005: MSFS Version Strategy

**Status:** Accepted

### Context

Microsoft Flight Simulator ships in two major versions with different SimConnect
documentation:

- **MSFS 2020** (SDK ~0.21): current installed base, mature documentation
- **MSFS 2024** (SDK ~1.x): newer, some variables and events differ or are added

Options evaluated:
1. MSFS 2020 only (simpler, known corpus)
2. MSFS 2024 only (forward-looking)
3. Both, operator-selectable via env var
4. Both, merged into single corpus with version annotation on each item

### Decision

**Serve both corpora. Default to MSFS 2024. Operator selects via `DOCS_MSFS_VERSION`
env var (`2020` | `2024` | `both`). In `both` mode, items carry a `versions` field
listing which simulators they apply to.**

### Rationale

Operators running MSFS 2020 must not be forced to upgrade. Operators running MSFS 2024
should get the current documentation by default. The `both` mode enables a single
server to answer queries about either simulator, which is the highest-value mode for
AI agents that assist users regardless of which simulator they run.

Corpus files are kept separate (`simvars_2020.json`, `simvars_2024.json`) rather than
merged at scrape time, which preserves the ability to update them independently and
makes the scraper simpler. The `DocStore` merges them at load time when `both` is
selected, deduplicating by canonical name and annotating the `Versions` field.

### Consequences

- Two sets of corpus files per collection type that varies between versions (SimVars,
  Events).
- Shared collections (Functions, Structures, ErrorCodes) have one file, version-
  agnostic.
- `DocStore.Search` and `DocStore.GetSimVar` operate on the merged view; callers
  receive `Versions []string` on each result indicating applicability.
- `GET /health` response includes `docs_version: "2024"` (or `"2020"` or `"both"`).

---

## 7. Component Breakdown

### Package Map

```
cmd/simconnect-mcp/
  main.go                        Entry point. Reads env vars, wires dependencies,
                                 starts Gin. No business logic.

internal/server/
  server.go                      Gin engine factory with middleware stack.
  middleware/
    requestid.go                 X-Request-ID injection.
    cors.go                      CORS for local dev origins.
    errors.go                    Structured JSON error response helpers.

internal/modes/docs/
  docs.go                        Mode entry point. Implements Mode interface.
                                 Wires corpus -> DocStore -> MCP tools -> Gin routes.

internal/corpus/
  embed.go                       //go:embed assets/*.json — single file.
  loader.go                      LoadEmbedded() and LoadFromPath() functions.
  store.go                       DocStore struct and all query methods.
  store_test.go                  Unit tests with fixture corpus.
  types.go                       Go structs: SimVar, Event, Function, Structure,
                                 ErrorCode, Corpus.
  assets/
    simvars_2020.json
    simvars_2024.json
    events_2020.json
    events_2024.json
    functions.json
    structures.json
    error_codes.json

internal/mcpadapter/
  adapter.go                     Thin wrapper: NewServer(), AddTool() delegating to
                                 mcp-go. Isolates mcp-go API surface.
  handler.go                     ToolHandler type alias and adapter helpers.

internal/modes/docs/tools/
  simvars.go                     RegisterSimVarTools(server, store)
  events.go                      RegisterEventTools(server, store)
  functions.go                   RegisterFunctionTools(server, store)
  structures.go                  RegisterStructureTools(server, store)
  errorcodes.go                  RegisterErrorCodeTools(server, store)
  search.go                      RegisterSearchTool(server, store)

tools/scraper/                   Build-time CLI. Not in server binary.
  main.go
  scraper.go
  parser.go
  testdata/
    sample_simvars.html
```

### Responsibilities Summary

| Package | Responsibility |
|---|---|
| `cmd/simconnect-mcp` | Entry point, env config, dependency wiring, server start |
| `internal/server` | Gin engine, middleware, shared HTTP utilities |
| `internal/corpus` | Data types, embedded JSON loading, in-memory query store |
| `internal/mcpadapter` | mcp-go isolation; `MCPServer` creation and tool registration API |
| `internal/modes/docs` | Mode wiring: loads corpus, registers tools, mounts routes |
| `internal/modes/docs/tools` | One file per tool group; pure tool registration logic |
| `tools/scraper` | Build-time HTML scraper and JSON emitter; dev dependency only |

---

## 8. Interface Definitions

### 8.1 Mode Interface

Every mode (docs, simconnect) implements this interface. The main function calls `Mount`
to attach the mode's routes to the shared Gin engine.

```go
// internal/modes/mode.go

package modes

import "github.com/gin-gonic/gin"

// Mode represents an operating mode of the MCP server.
// Implementations are responsible for loading their own dependencies
// and registering all routes onto the provided router group.
type Mode interface {
    // Mount attaches this mode's routes (MCP endpoints, health, etc.)
    // to the provided Gin router. Mount is called once at startup.
    // It returns an error if the mode cannot initialise (e.g., corpus
    // fails to load).
    Mount(r *gin.Engine) error

    // HealthInfo returns key-value pairs to include in the /health
    // response payload. Keys must be snake_case strings.
    HealthInfo() map[string]any
}
```

### 8.2 DocStore Interface

All MCP tool handlers receive a `DocStore` by value. No tool handler holds a reference
to any mcp-go type directly — they only depend on `DocStore`.

```go
// internal/corpus/store.go

package corpus

import "context"

// DocStore is the read-only in-memory index of SimConnect documentation.
// All methods are safe for concurrent use.
type DocStore interface {
    // ListSimVars returns a paginated, optionally-filtered list of SimVars.
    // category is case-insensitive; empty string matches all categories.
    // page is 1-indexed. pageSize must be between 1 and 200.
    ListSimVars(ctx context.Context, category string, page, pageSize int) ([]SimVar, int, error)

    // GetSimVar returns the SimVar with the given name (case-insensitive).
    // Returns ErrNotFound if no match exists.
    GetSimVar(ctx context.Context, name string) (SimVar, error)

    // ListEvents returns a paginated list of Input Events.
    ListEvents(ctx context.Context, page, pageSize int) ([]Event, int, error)

    // GetEvent returns the Event with the given name (case-insensitive).
    // Returns ErrNotFound if no match exists.
    GetEvent(ctx context.Context, name string) (Event, error)

    // ListFunctions returns all C API functions, paginated.
    ListFunctions(ctx context.Context, page, pageSize int) ([]Function, int, error)

    // GetFunction returns the Function with the given name (case-insensitive).
    // Returns ErrNotFound if no match exists.
    GetFunction(ctx context.Context, name string) (Function, error)

    // ListStructures returns all data structures, paginated.
    ListStructures(ctx context.Context, page, pageSize int) ([]Structure, int, error)

    // GetStructure returns the Structure with the given name (case-insensitive).
    // Returns ErrNotFound if no match exists.
    GetStructure(ctx context.Context, name string) (Structure, error)

    // ListErrorCodes returns all SIMCONNECT_EXCEPTION values, paginated.
    ListErrorCodes(ctx context.Context, page, pageSize int) ([]ErrorCode, int, error)

    // GetErrorCode returns the ErrorCode with the given name (case-insensitive).
    // Returns ErrNotFound if no match exists.
    GetErrorCode(ctx context.Context, name string) (ErrorCode, error)

    // Search performs a case-insensitive keyword search across all corpus
    // item types. Returns up to limit results. type_ filters by item type
    // ("simvar", "event", "function", "structure", "errorcode", or "" for all).
    Search(ctx context.Context, query, type_ string, limit int) (SearchResults, error)
}

// ErrNotFound is returned by Get* methods when no item matches.
var ErrNotFound = errors.New("not found")
```

### 8.3 DocLoader Interface

```go
// internal/corpus/loader.go

package corpus

// DocLoader loads the raw corpus data from a source and returns a populated
// Corpus value ready for indexing by NewDocStore.
type DocLoader interface {
    Load() (Corpus, error)
}

// LoadEmbedded returns a DocLoader that reads from the embedded JSON assets.
// This is the default loader used in production.
func LoadEmbedded() DocLoader

// LoadFromPath returns a DocLoader that reads JSON files from the given
// filesystem directory path. Used when DOCS_OVERRIDE_PATH is set.
func LoadFromPath(path string) DocLoader
```

### 8.4 MCP Tool Handler Signature

All MCP tool handlers in `internal/modes/docs/tools/` follow this shape. They receive
a `DocStore` at registration time via closure capture, never via global state.

```go
// internal/mcpadapter/handler.go

package mcpadapter

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// ToolHandlerFunc is the function signature for all MCP tool handlers
// in this project. The mcp-go library calls this function with the
// request context and the parsed tool call arguments.
type ToolHandlerFunc func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// ErrorResult constructs a standard MCP error result with a structured
// JSON body. Use this for all error returns from tool handlers.
func ErrorResult(code string, message string) *mcp.CallToolResult

// TextResult constructs a standard MCP text/plain result.
func TextResult(content string) *mcp.CallToolResult

// JSONResult constructs a standard MCP result with a JSON-serialised payload.
// v must be JSON-marshallable.
func JSONResult(v any) (*mcp.CallToolResult, error)
```

### 8.5 Config

```go
// internal/modes/docs/config.go

package docs

// Config holds all runtime configuration for docs mode.
// Values are sourced from environment variables in main.go and passed
// explicitly — no global reads inside the package.
type Config struct {
    // MSFSVersion controls which corpus to load.
    // Valid values: "2020", "2024", "both". Default: "2024".
    MSFSVersion string

    // OverridePath, when non-empty, loads corpus from this filesystem path
    // instead of the embedded assets.
    OverridePath string

    // ListenAddr is the address the HTTP server binds to. Default: ":8080".
    ListenAddr string
}
```

---

## 9. Data Schema

All corpus types are defined in `internal/corpus/types.go`.

```go
package corpus

// SimVar represents a single Simulation Variable from the SimConnect SDK.
type SimVar struct {
    // Name is the canonical SimVar name as used in the SDK (e.g. "PLANE ALTITUDE").
    Name string `json:"name"`

    // Description is the full human-readable description from the SDK docs.
    Description string `json:"description"`

    // Units lists the valid unit strings for this variable (e.g. ["feet", "meters"]).
    Units []string `json:"units"`

    // Settable indicates whether SimConnect_SetDataOnSimObject can modify this var.
    Settable bool `json:"settable"`

    // Category is the SDK documentation category (e.g. "Aircraft Position and Speed").
    Category string `json:"category"`

    // Versions lists which simulators define this variable.
    // Valid values: "2020", "2024".
    Versions []string `json:"versions"`

    // IndexedBy, when non-empty, indicates the variable accepts an integer index
    // (e.g. "engine number"). Format: "1-indexed engine number".
    IndexedBy string `json:"indexed_by,omitempty"`

    // Deprecated indicates this variable is marked deprecated in the SDK docs.
    Deprecated bool `json:"deprecated,omitempty"`

    // DeprecatedReason is populated when Deprecated is true.
    DeprecatedReason string `json:"deprecated_reason,omitempty"`
}

// Event represents a SimConnect Input Event (Key Event ID).
type Event struct {
    // Name is the canonical event name (e.g. "LANDING_LIGHTS_TOGGLE").
    Name string `json:"name"`

    // Description is the full human-readable description.
    Description string `json:"description"`

    // Parameters lists positional parameters this event accepts.
    Parameters []EventParam `json:"parameters,omitempty"`

    // Versions lists which simulators define this event.
    Versions []string `json:"versions"`

    // Deprecated indicates this event is superseded.
    Deprecated bool `json:"deprecated,omitempty"`
}

// EventParam describes a single parameter for an Input Event.
type EventParam struct {
    Index       int    `json:"index"`        // 0-based parameter position
    Name        string `json:"name"`
    Description string `json:"description"`
    Type        string `json:"type"`         // "DWORD", "float", etc.
}

// Function represents a C API function in the SimConnect SDK.
type Function struct {
    // Name is the C function name (e.g. "SimConnect_Open").
    Name string `json:"name"`

    // Description is the full human-readable description.
    Description string `json:"description"`

    // Signature is the C function signature as a string.
    Signature string `json:"signature"`

    // Parameters lists each parameter with name, type, and description.
    Parameters []FunctionParam `json:"parameters"`

    // ReturnType is the C return type (e.g. "HRESULT").
    ReturnType string `json:"return_type"`

    // Remarks contains additional SDK notes about usage.
    Remarks string `json:"remarks,omitempty"`
}

// FunctionParam describes one parameter of a C API function.
type FunctionParam struct {
    Name        string `json:"name"`
    Type        string `json:"type"`
    Direction   string `json:"direction"` // "in", "out", "in/out"
    Description string `json:"description"`
    Optional    bool   `json:"optional,omitempty"`
}

// Structure represents a C data structure in the SimConnect SDK.
type Structure struct {
    // Name is the C struct name (e.g. "SIMCONNECT_RECV").
    Name string `json:"name"`

    // Description is the full human-readable description.
    Description string `json:"description"`

    // Fields lists each member field.
    Fields []StructField `json:"fields"`

    // Remarks contains additional SDK notes.
    Remarks string `json:"remarks,omitempty"`
}

// StructField describes one field in a C structure.
type StructField struct {
    Name        string `json:"name"`
    Type        string `json:"type"`
    Description string `json:"description"`
}

// ErrorCode represents one value of the SIMCONNECT_EXCEPTION enum.
type ErrorCode struct {
    // Name is the enum value name (e.g. "SIMCONNECT_EXCEPTION_NONE").
    Name string `json:"name"`

    // Value is the integer value of the enum member.
    Value int `json:"value"`

    // Description is the full human-readable description.
    Description string `json:"description"`
}

// Corpus is the top-level container for all loaded documentation.
// JSON files are deserialised into this structure by the corpus loaders.
type Corpus struct {
    SimVars    []SimVar    `json:"simvars"`
    Events     []Event     `json:"events"`
    Functions  []Function  `json:"functions"`
    Structures []Structure `json:"structures"`
    ErrorCodes []ErrorCode `json:"error_codes"`
}

// SearchResult is a single hit returned by DocStore.Search.
type SearchResult struct {
    // Type is the item type: "simvar", "event", "function", "structure", "errorcode".
    Type string `json:"type"`

    // Name is the item's canonical name.
    Name string `json:"name"`

    // Excerpt is a short snippet from the description showing relevance.
    Excerpt string `json:"excerpt"`
}

// SearchResults groups all hits from a search query.
type SearchResults struct {
    Query   string         `json:"query"`
    Total   int            `json:"total"`
    Results []SearchResult `json:"results"`
}

// Page is a generic pagination wrapper returned by List* methods.
type Page[T any] struct {
    Items      []T `json:"items"`
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
    TotalItems int `json:"total_items"`
    TotalPages int `json:"total_pages"`
}
```

---

## 10. Dependency Graph

```
cmd/simconnect-mcp/main
    |
    +-- internal/server
    |       |
    |       +-- internal/server/middleware
    |
    +-- internal/modes/docs
            |
            +-- internal/modes/docs/tools/simvars
            +-- internal/modes/docs/tools/events
            +-- internal/modes/docs/tools/functions
            +-- internal/modes/docs/tools/structures
            +-- internal/modes/docs/tools/errorcodes
            +-- internal/modes/docs/tools/search
            |       (all tools packages import:)
            |       +-- internal/corpus
            |       +-- internal/mcpadapter
            |
            +-- internal/corpus
            |       +-- [standard library only]
            |       +-- embed (stdlib)
            |
            +-- internal/mcpadapter
                    +-- github.com/mark3labs/mcp-go/server
                    +-- github.com/mark3labs/mcp-go/mcp

Legend: --> = imports
Constraint: internal/corpus imports NO external packages except stdlib.
Constraint: internal/modes/simconnect does NOT import internal/corpus.
Constraint: internal/mcpadapter is the ONLY package that imports mcp-go directly.
```

### Import Isolation Rules

1. `internal/corpus` must have zero external dependencies. It is pure data and
   business logic — no HTTP, no MCP, no Gin.
2. `internal/mcpadapter` is the sole gateway to the mcp-go library. No other package
   imports `github.com/mark3labs/mcp-go` directly.
3. `internal/modes/simconnect` (Milestone 2) must never import `internal/corpus`.
4. `cmd/simconnect-mcp/main.go` may import `internal/server` and one mode package.
   It must not import `internal/corpus` directly.

---

## 11. MCP Tool Catalogue

These are the ten MCP tools exposed by docs mode. All tool names are snake_case.
All tools return structured JSON results (not plain text).

| Tool Name | Description | Required Args | Optional Args |
|---|---|---|---|
| `list_simvars` | Paginated list of SimVars with optional category filter | — | `category`, `page` (default 1), `page_size` (default 50) |
| `get_simvar` | Look up a SimVar by exact name (case-insensitive) | `name` | — |
| `list_events` | Paginated list of Input Events | — | `page`, `page_size` |
| `get_event` | Look up an Input Event by exact name | `name` | — |
| `list_functions` | Paginated list of C API functions | — | `page`, `page_size` |
| `get_function` | Look up a C API function by exact name | `name` | — |
| `list_structures` | Paginated list of C data structures | — | `page`, `page_size` |
| `get_structure` | Look up a C structure by exact name | `name` | — |
| `list_error_codes` | Paginated list of SIMCONNECT_EXCEPTION values | — | `page`, `page_size` |
| `get_error_code` | Look up a SIMCONNECT_EXCEPTION value by name | `name` | — |
| `search_docs` | Keyword search across all corpus types | `query` | `type` (filter), `limit` (default 20, max 100) |

Pagination defaults: `page=1`, `page_size=50`, `page_size` max `200`.

---

## 12. Error Handling Contract

All tool handler errors return a structured JSON body as the MCP tool result content.
HTTP-level errors (e.g., malformed JSON-RPC) are handled by mcp-go internally.

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "SimVar 'PLANE_ALTITUDE' not found. Did you mean 'PLANE ALTITUDE'?",
    "request_id": "a1b2c3d4"
  }
}
```

Error codes returned by tool handlers:

| Code | Condition |
|---|---|
| `NOT_FOUND` | `GetSimVar`, `GetEvent`, etc. found no match |
| `INVALID_ARGUMENT` | Required argument missing or fails validation |
| `INVALID_PAGE` | `page` < 1 or `page_size` out of range |
| `INTERNAL_ERROR` | Unexpected error (JSON marshal failure, etc.) |

The `/health` endpoint always returns `200 OK` with JSON:

```json
{
  "status": "ok",
  "mode": "docs",
  "docs_loaded": true,
  "docs_source": "embedded",
  "docs_version": "2024",
  "simvar_count": 1247,
  "event_count": 834
}
```

When `docs_loaded` is `false`, all MCP tool calls return `INTERNAL_ERROR` with
message `"Corpus not loaded"`.

---

## 13. Configuration Contract

All configuration is sourced from environment variables read in `main.go` and
passed as a `docs.Config` struct. No package below `main.go` reads environment
variables directly.

| Env Var | Default | Values | Description |
|---|---|---|---|
| `MCP_MODE` | `docs` | `docs`, `simconnect` | Operating mode |
| `PORT` | `8080` | Any valid port | HTTP listen port |
| `DOCS_MSFS_VERSION` | `2024` | `2020`, `2024`, `both` | Which simulator corpus to serve |
| `DOCS_OVERRIDE_PATH` | (empty) | Filesystem path | Override embedded corpus with local files |
| `GIN_MODE` | `debug` | `debug`, `release`, `test` | Gin logging verbosity |

---

## 14. Implementation Guidelines

### File Creation Order

Implement packages in dependency order to keep the build green at each step:

1. `internal/corpus/types.go` — data types only, no logic
2. `internal/corpus/embed.go` + `assets/*.json` (fixture files for now)
3. `internal/corpus/loader.go`
4. `internal/corpus/store.go` + `store_test.go`
5. `internal/mcpadapter/adapter.go` + `handler.go`
6. `internal/server/middleware/` packages
7. `internal/server/server.go` update
8. `internal/modes/docs/tools/` (one file per tool group)
9. `internal/modes/docs/docs.go`
10. `cmd/simconnect-mcp/main.go` update
11. `tools/scraper/` (after server is working)

### Testing Strategy

- `internal/corpus` tests use a small in-memory fixture corpus, never the embedded
  production assets. This keeps test execution sub-second and prevents test failures
  from corpus format changes.
- `internal/modes/docs/tools` tests use a `MockDocStore` implementing `DocStore`.
- Integration tests in `tests/` start the full server using
  `server.NewTestStreamableHTTPServer` and verify tool call results end-to-end.
- No test may require a network connection or a running simulator.

### Naming Conventions

- Go package names: `corpus`, `docs`, `mcpadapter`, `middleware`, `server`
- MCP tool names: `list_simvars`, `get_simvar` (snake_case, verb_noun)
- Error codes: `NOT_FOUND`, `INVALID_ARGUMENT` (SCREAMING_SNAKE_CASE)
- JSON field names: `snake_case` throughout corpus and API responses
- Environment variables: `SCREAMING_SNAKE_CASE` with `DOCS_` prefix for mode-specific vars

### Build Tags

- `internal/modes/simconnect/simconnect.go` carries `//go:build windows` (unchanged)
- All docs mode files carry no build tag — they compile on all platforms
- `tools/scraper/` carries no build tag but is excluded from `./internal/...` and
  `./cmd/...` build paths

### go:generate

Add to `internal/corpus/embed.go`:
```go
//go:generate go run ../../tools/scraper/main.go -out assets/
```

This allows `go generate ./internal/corpus/...` to refresh the corpus assets.
