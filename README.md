# SimConnect MCP

SimConnect MCP exposes Microsoft Flight Simulator's SimConnect SDK documentation as a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server. AI assistants — Claude, GitHub Copilot, and others — can answer questions about SimVars, events, API functions, data structures, and error codes without leaving the chat.

The server is written in Go with Gin routing and operates in two modes: **documentation fetch** (cross-platform) and **live SimConnect data** (Windows-only).

## Documentation

Full documentation is available at **[simconnect-mcp.mrlm.net](https://simconnect-mcp.mrlm.net)** — getting started guides, configuration reference, MCP tool API docs, and architecture notes.

## Installation

### Prebuilt binaries (recommended)

Download the latest release for your platform from the [GitHub Releases page](https://github.com/mrlm-net/simconnect-mcp/releases). Extract the archive and place the binary in your `PATH`.

### go install

```sh
go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest
```

Requires Go 1.24+. The binary is installed as `simconnect-mcp`.

> **Note:** The `simconnect` mode binary for Windows (CGo/SimConnect SDK) is not available via `go install` due to CGo requirements. Download the Windows release binary instead.

### Build from source

```sh
git clone git@github.com:mrlm-net/simconnect-mcp.git
cd simconnect-mcp
go build -o simconnect-mcp ./cmd/simconnect-mcp/
```

## Prerequisites

- **Go 1.24+** (build from source or `go install` only)
- `docs` mode runs on any operating system — no additional prerequisites.
- `simconnect` mode requires:
  - **Windows 10/11 (x64)**
  - **Microsoft Flight Simulator 2020 or 2024** with SimConnect enabled
  - **SimConnect SDK** — installed via MSFS Developer Mode tools or the standalone SDK installer

## Quick Start

```sh
# Run in docs mode (cross-platform)
MCP_MODE=docs simconnect-mcp

# Run in live SimConnect mode (Windows only — requires MSFS 2020 or 2024 running)
MCP_MODE=simconnect simconnect-mcp
```

Or with `go run` from the repository root:

```sh
# docs mode
MCP_MODE=docs go run ./cmd/simconnect-mcp/

# simconnect mode (Windows only)
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

Add the following to your `claude_desktop_config.json` (or equivalent MCP client config) to connect Claude Desktop to the docs server.

**Using the installed binary** (recommended):

```json
{
  "mcpServers": {
    "simconnect": {
      "command": "simconnect-mcp",
      "env": {
        "MCP_MODE": "docs",
        "DOCS_MSFS_VERSION": "2024"
      }
    }
  }
}
```

**Using `go run`** (development / no binary installed):

```json
{
  "mcpServers": {
    "simconnect": {
      "command": "go",
      "args": ["run", "./cmd/simconnect-mcp/"],
      "cwd": "/path/to/simconnect-mcp",
      "env": {
        "MCP_MODE": "docs",
        "DOCS_MSFS_VERSION": "2024"
      }
    }
  }
}
```

## Available Tools

The server exposes 12 MCP tools in `docs` mode. See [docs/api/mcp-tools.md](docs/api/mcp-tools.md) for full parameter references, request/response examples, and error codes.

| Tool | Description |
|------|-------------|
| `list_simvar_categories` | List all SimVar category strings valid for the `category` filter of `list_simvars` |
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
| `search_docs` | Keyword search across all corpus types; all query words must appear in the name or description |

All `list_*` tools return a paginated envelope (`items`, `page`, `page_size`, `total_items`, `total_pages`). Default page size is 20; maximum is 100.

## Live SimConnect Mode

`simconnect` mode connects to a running instance of MSFS 2020 or MSFS 2024 via the SimConnect SDK and exposes live simulator data as MCP tools. It is Windows-only and must be built with the `-tags windows` flag.

The server reconnects automatically when the simulator restarts — no manual intervention is required.

**Simulation variables**

| Tool | Description |
|------|-------------|
| `get_simvar_value` | Read the current value of a single simulation variable from the running simulator |
| `get_simvar_values` | Read up to 20 simulation variables in a single request |
| `set_simvar_value` | Write a numeric simulation variable to the user aircraft |
| `transmit_event` | Send a Key Event ID to the simulator (e.g., toggle landing gear, set autopilot altitude) |
| `get_sim_state` | Return high-level simulator state: paused, running, aircraft title, position, and speed |

**Traffic**

| Tool | Description |
|------|-------------|
| `get_nearby_traffic` | List aircraft within a given radius — object ID, callsign, position, speed, heading |
| `get_traffic_with_phase` | Enriched traffic: adds vertical speed, ground track, flight phase, runway occupancy |

**Airports**

| Tool | Description |
|------|-------------|
| `get_airports_in_range` | List airports in the simulator's loaded scenery area, sorted by distance |
| `get_nearest_airport` | Return the single closest airport to the player aircraft |
| `get_airport_details` | Detailed facility data: runways, ATC frequencies, stands, approaches, SIDs, STARs |
| `get_airport_taxiways` | Taxiway network graph: names, directed path edges, and node positions |
| `get_taxiway_names` | Lightweight list of taxiway letter strings only (no paths or points) |
| `get_airport_parkings` | All parking stands, gates, and ramps with type, heading, and radius |

**Navigation facilities**

| Tool | Description |
|------|-------------|
| `get_vors_in_range` | List VOR stations within a radius |
| `get_vor_details` | Detailed VOR data: frequency, type, range, declination |
| `get_ndbs_in_range` | List NDB stations within a radius |
| `get_ndb_details` | Detailed NDB data: frequency and range |
| `get_waypoints_in_range` | List waypoints within a radius |
| `get_waypoint_details` | Detailed waypoint data: position and magvar |

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

All development happens on `main`. Browse open issues and submit bug reports or feature requests at [github.com/mrlm-net/simconnect-mcp/issues](https://github.com/mrlm-net/simconnect-mcp/issues).
