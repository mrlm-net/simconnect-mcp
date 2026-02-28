# SimConnect MCP

SimConnect MCP exposes Microsoft Flight Simulator's SimConnect SDK documentation as a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server. AI assistants — Claude, GitHub Copilot, and others — can answer questions about SimVars, events, API functions, data structures, and error codes without leaving the chat.

The server is written in Go with Gin routing and operates in two modes reflecting the project milestones: **documentation fetch** (cross-platform) and **live SimConnect data** (Windows-only).

## Milestones

| Milestone | Mode | `MCP_MODE` value | Status |
|-----------|------|------------------|--------|
| M1 | Documentation fetch | `docs` | Complete |
| M2 | Live SimConnect data | `simconnect` | Complete |

## Prerequisites

- **Go 1.24+**
- Milestone 1 (`docs` mode) runs on any operating system — no additional prerequisites.
- Milestone 2 (`simconnect` mode) requires:
  - **Windows 10/11 (x64)**
  - **Microsoft Flight Simulator 2020 or 2024** with SimConnect enabled
  - **SimConnect SDK** — installed via MSFS Developer Mode tools or the standalone SDK installer

## Quick Start

```sh
git clone git@github.com:mrlm-net/simconnect-mcp.git
cd simconnect-mcp
go mod download

# Run in docs mode (default, cross-platform)
MCP_MODE=docs go run ./cmd/simconnect-mcp/

# Run in live SimConnect mode (Windows only — requires MSFS 2020 or 2024 running)
MCP_MODE=simconnect go run -tags windows ./cmd/simconnect-mcp/
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
| `MCP_MODE` | `docs` | `docs`, `simconnect` | Operating mode. `simconnect` requires Windows and `-tags windows` build flag. |
| `PORT` | `8080` | any port number | HTTP listen port. Applies to both `docs` and `simconnect` modes. |
| `SIMCONNECT_APP_NAME` | `simconnect-mcp` | any string | App name registered with the SimConnect SDK. Used in `simconnect` mode only. |
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

## Live SimConnect Mode (Milestone 2)

`simconnect` mode connects to a running instance of MSFS 2020 or MSFS 2024 via the SimConnect SDK and exposes live simulator data as MCP tools. It is Windows-only and must be built with the `-tags windows` flag.

The server reconnects automatically when the simulator restarts — no manual intervention is required.

| Tool | Description |
|------|-------------|
| `get_simvar_value` | Read the current value of a single simulation variable from the running simulator |
| `get_simvar_values` | Read multiple simulation variables in a single request |
| `transmit_event` | Send a Key Event ID to the simulator (e.g., toggle landing gear, set autopilot altitude) |
| `get_sim_state` | Return high-level simulator state: paused, running, aircraft title, and connection status |

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

Milestone 1 development lives on the `milestone/1-docs` branch; Milestone 2 development lives on `milestone/2-simconnect`. Issues are labelled `milestone-1` or `milestone-2`.

Browse open issues and submit bug reports or feature requests at [github.com/mrlm-net/simconnect-mcp/issues](https://github.com/mrlm-net/simconnect-mcp/issues).
