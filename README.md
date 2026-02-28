# SimConnect MCP

SimConnect MCP exposes Microsoft Flight Simulator's SimConnect SDK documentation as a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server. AI assistants — Claude, GitHub Copilot, and others — can answer questions about SimVars, events, API functions, data structures, and error codes without leaving the chat.

The server is written in Go with Gin routing and operates in two modes reflecting the project milestones: **documentation fetch** (cross-platform, current) and **live SimConnect data** (Windows-only, planned).

## Milestones

| Milestone | Mode | `MCP_MODE` value | Status |
|-----------|------|------------------|--------|
| M1 | Documentation fetch | `docs` | Current |
| M2 | Live SimConnect data | `simconnect` | Planned |

## Prerequisites

- **Go 1.24+**
- Windows SDK + SimConnect SDK — required for Milestone 2 only

## Quick Start

```sh
git clone git@github.com:mrlm-net/simconnect-mcp.git
cd simconnect-mcp
go mod download
MCP_MODE=docs go run ./cmd/simconnect-mcp/
```

Expected output (Gin startup log):

```
[GIN-debug] [WARNING] Creating an engine instance with the Logger and Recovery middleware already attached.
[GIN-debug] GET    /health                   --> ...
[GIN-debug] POST   /mcp                      --> ...
[GIN-debug] GET    /sse                       --> ...
[GIN-debug] POST   /message                  --> ...
[GIN-debug] Listening and serving HTTP on :8080
```

## Environment Variables

| Variable | Default | Valid values | Description |
|----------|---------|--------------|-------------|
| `MCP_MODE` | `docs` | `docs` | Operating mode. `simconnect` is available in Milestone 2. |
| `PORT` | `8080` | any port number | HTTP listen port. |
| `DOCS_MSFS_VERSION` | `2024` | `2020`, `2024`, `both` | SDK version of the corpus to serve. |
| `DOCS_OVERRIDE_PATH` | *(embedded)* | filesystem path | Override the embedded corpus with local JSON files. See [Security note](#security-note). |
| `GIN_MODE` | `debug` | `debug`, `release` | Gin operating mode. `release` enforces localhost-only CORS (DNS rebinding protection). |

## MCP Client Configuration

Add the following to your `claude_desktop_config.json` (or equivalent MCP client config) to connect Claude Desktop to the docs server:

```json
{
  "mcpServers": {
    "simconnect": {
      "command": "go",
      "args": ["run", "./cmd/simconnect-mcp/"],
      "env": {
        "MCP_MODE": "docs",
        "DOCS_MSFS_VERSION": "2024"
      }
    }
  }
}
```

The working directory must be the repository root when `go run` resolves the package path.

## Available Tools

The server exposes 11 MCP tools in `docs` mode. See [docs/api/mcp-tools.md](docs/api/mcp-tools.md) for full parameter references, request/response examples, and error codes.

| Tool | Description |
|------|-------------|
| `list_simvars` | List simulation variables, optionally filtered by category, with pagination |
| `get_simvar` | Fetch a single simulation variable by name (case-insensitive) |
| `list_events` | List client input events (Key Event IDs) with pagination |
| `get_event` | Fetch a single client event by name (case-insensitive) |
| `list_functions` | List SimConnect C API functions with pagination |
| `get_function` | Fetch a single SDK API function by name (case-insensitive) |
| `list_structures` | List SimConnect C data structures with pagination |
| `get_structure` | Fetch a single data structure by name (case-insensitive) |
| `list_error_codes` | List `SIMCONNECT_EXCEPTION` enum values with pagination |
| `get_error_code` | Fetch an error code by name or integer value |
| `search_docs` | Full-text search across all corpus types (SimVars, events, functions, structures, error codes) |

All `list_*` tools return a paginated envelope (`items`, `page`, `page_size`, `total_items`, `total_pages`). Default page size is 20; maximum is 100.

## Refreshing the Corpus

The embedded corpus is hand-authored from the SimConnect SDK documentation. To regenerate it from the SDK docs website, run the scraper tool:

```sh
go run ./tools/scraper/ -out internal/corpus/assets/ -version both
```

You can also trigger regeneration via `go generate` inside the corpus package:

```sh
go generate ./internal/corpus/...
```

The `-version` flag accepts `2020`, `2024`, or `both`.

## Running Tests

```sh
# Run all unit tests
go test ./...

# Run with race detector
go test -race ./...
```

Unit tests live alongside the code in `_test.go` files. Integration tests are in `tests/`.

## Security Note

`DOCS_OVERRIDE_PATH` is an operator-level configuration option. It must point to a trusted local directory and must never be derived from user-provided input. In multi-tenant deployments, leave it unset so the server uses the embedded corpus.

`GIN_MODE=release` is recommended for any non-development deployment. In release mode, the CORS middleware rejects requests from non-localhost origins (DNS rebinding protection). In `debug` mode all origins are permitted, which is convenient for local development tooling but unsuitable for production.

## Contributing

Active development for Milestone 1 happens on the `milestone/1-docs` branch. Issues are labelled `milestone-1` or `milestone-2`.

Browse open issues and submit bug reports or feature requests at [github.com/mrlm-net/simconnect-mcp/issues](https://github.com/mrlm-net/simconnect-mcp/issues).
