# SimConnect MCP

SimConnect MCP is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server for Microsoft Flight Simulator. AI assistants — Claude, GitHub Copilot, and others — can:

- answer questions about SimVars, events, API functions, data structures and error codes from the SimConnect SDK documentation;
- read and control the running simulator;
- bring your airport to life: an airline schedule that runs by itself, with AI departures, arrivals, turnarounds and overflights, cleared and sequenced by a tower and an approach controller. See the [AI Traffic & ATC guide](https://simconnect-mcp.mrlm.net/docs/ai-traffic).

The server is written in Go with Gin routing and operates in two modes: **documentation fetch** (cross-platform) and **live SimConnect data** (Windows-only) — or both at once with `MCP_MODE=both`.

## Documentation

Full documentation is available at **[simconnect-mcp.mrlm.net](https://simconnect-mcp.mrlm.net)** — getting started guides, configuration reference, MCP tool API docs, and architecture notes.

## Installation

### Prebuilt binaries (recommended)

Download the latest release for your platform from the [GitHub Releases page](https://github.com/mrlm-net/simconnect-mcp/releases). Extract the archive and place the binary in your `PATH`.

### go install

```sh
go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest
```

Requires Go 1.27+. The binary is installed as `simconnect-mcp`.

> **Note:** The `simconnect` mode binary for Windows (CGo/SimConnect SDK) is not available via `go install` due to CGo requirements. Download the Windows release binary instead.

### Build from source

```sh
git clone git@github.com:mrlm-net/simconnect-mcp.git
cd simconnect-mcp
go build -o simconnect-mcp ./cmd/simconnect-mcp/
```

## Prerequisites

- **Go 1.27+** (build from source or `go install` only)
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
| `MCP_MODE` | `docs` | `docs`, `simconnect`, `both` | Operating mode. `simconnect` requires Windows and `-tags windows` build flag. `both` serves the docs tools plus, on Windows with the simulator reachable at startup, the SimConnect tools. |
| `PORT` | `8080` | any port number | HTTP listen port. Applies to all modes. |
| `SIMCONNECT_APP_NAME` | `simconnect-mcp` | any string | App name registered with the SimConnect SDK. Used in `simconnect` and `both` modes. |
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

The server exposes 15 MCP tools in `docs` mode — 12 for the SimConnect SDK reference and 3 for the [`github.com/mrlm-net/simconnect`](https://github.com/mrlm-net/simconnect) Go library guides — and 58 in `simconnect` mode. `both` mode on Windows serves all 73 when SimConnect is reachable at startup, and the 15 docs tools otherwise. See [docs/mcp-tools-docs.md](docs/mcp-tools-docs.md) and [docs/mcp-tools-simconnect.md](docs/mcp-tools-simconnect.md) for full parameter references, request/response examples, and error codes.

**SimConnect SDK reference**

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
| `list_library_guides` | List the guides of the `mrlm-net/simconnect` Go library with their chapter headings |
| `get_library_guide` | Read a library guide (or one chapter of it) as Markdown |
| `search_library_docs` | Keyword search across the library guides; returns matching chapters with an excerpt |

All paginated `list_*` tools return an envelope (`items`, `page`, `page_size`, `total_items`, `total_pages`). Default page size is 20; maximum is 100.

**Go library guides**

The 45 guides of the `github.com/mrlm-net/simconnect` Go library (client, manager, facilities, `pkg/airport`, `pkg/nav`, `pkg/traffic`, `pkg/systems`, `pkg/avionics`, `pkg/addons`, `pkg/camera`) are embedded at the library version the server is built with — currently **v0.23.1**. The `/health` response (docs and both modes) reports it as `library_version`.

| Tool | Description |
|------|-------------|
| `list_library_guides` | List the library guides with their chapter headings, optionally filtered by section |
| `get_library_guide` | Read a guide as Markdown — the whole guide, or one `##` chapter by heading or unique prefix |
| `search_library_docs` | Keyword search across guide chapters; every query word must appear in the chapter, its heading, or the guide title |

## Live SimConnect Mode

`simconnect` mode connects to a running instance of MSFS 2020 or MSFS 2024 via the SimConnect SDK and exposes live simulator data as MCP tools. It is Windows-only and must be built with the `-tags windows` flag.

The server reconnects automatically when the simulator restarts — no manual intervention is required.

`simconnect` mode exposes 58 MCP tools; on Windows, `MCP_MODE=both` serves them together with the 15 docs tools (73 in all). See [docs/mcp-tools-simconnect.md](docs/mcp-tools-simconnect.md) for the full reference.

**Simulation variables**

| Tool | Description |
|------|-------------|
| `get_simvar_value` | Read the current value of a single simulation variable from the running simulator |
| `get_simvar_values` | Read up to 20 simulation variables in a single request |
| `set_simvar_value` | Write a numeric simulation variable to the user aircraft |
| `transmit_event` | Send a Key Event ID to the simulator (e.g., toggle landing gear, set autopilot altitude) |
| `get_sim_state` | Return high-level simulator state: paused, running, aircraft title, position, and speed |
| `get_fuel_state` | The user aircraft's fuel: total quantity, capacity, percent and weight, per tank |

**User aircraft**

| Tool | Description |
|------|-------------|
| `get_aircraft_systems` | Power, radios, engines, lights, doors by name, transponder, chocks, GPU, cabin signs and pushback state, read through the aircraft's systems profile (Fenix A320 family on its own L:vars) |
| `set_aircraft_control` | Open or close a door, set chocks, GPU, parking brake, cabin signs or external power, or call the cabin, the way the aircraft's profile says |
| `request_ground_service` | Ask for the sim's jetway, stairs, baggage, catering, ground power, fuel truck or pushback |
| `set_radio` | Set a COM active or standby frequency, swap a COM, or set the squawk |
| `set_atc_callsign` | Set the call sign the sim's ATC uses (ATC AIRLINE and ATC FLIGHT NUMBER) |
| `list_addons` | The installed MSFS packages: Community, Official and streamed, with streamed airports by ICAO |

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

The following tools are built on the [mrlm-net/simconnect](https://github.com/mrlm-net/simconnect) library (`pkg/airport`, `pkg/nav`). Weather is the simulator's ambient weather at the user aircraft (no gusts, ceiling or dewpoint), so runway-in-use and ATIS results are right for the airport the aircraft is at or near.

**Airport procedures & ground**

| Tool | Description |
|------|-------------|
| `get_airport_procedures` | List an airport's SIDs, STARs and approaches, or resolve one into its points (leg type, altitude and speed limits, IAF/FAF/MAP) |
| `plan_taxi_route` | ATC-style taxi route from a stand to a runway holding point, or from a runway exit to a stand — instruction, taxiways, crossings |
| `get_runway_entries_exits` | Taxiways onto a runway end (with runway remaining) and exits from it (distance, angle, high-speed, side) |
| `find_stands` | Parking stands that fit an aircraft — by wing span, airline and gates only |

**Weather & runway in use**

| Tool | Description |
|------|-------------|
| `get_weather` | Weather at the user aircraft: wind, visibility, temperature, QNH, precipitation, in-cloud, icing |
| `get_active_runway` | Departure and arrival runways in use, wind components, expected approach, transition altitude and level |
| `get_atis` | ATIS broadcast composed from the simulator's weather — as text and as spoken |

**Navigation & flight planning**

| Tool | Description |
|------|-------------|
| `get_fix` | Waypoint, VOR or NDB with position, frequency and the airways through it |
| `find_airway_route` | Airway route between two enroute fixes (crawled on demand, cached), e.g. `VOZ M725 OKF` |
| `plan_flight` | IFR flight plan between two airports: runways, SID, airways, STAR, approach, cruise level, time and fuel; optionally loads it into the simulator |

**AI traffic**

These tools run on the library's traffic engine, `pkg/traffic/world` (the airport map's): AI aircraft of our own with stand services, pushback, taxi, the tower, landing sequences, separation and the radio, at airports loaded around the user aircraft. The [AI Traffic & ATC guide](https://simconnect-mcp.mrlm.net/docs/ai-traffic) explains how it all fits together.

| Tool | Description |
|------|-------------|
| `list_aircraft_models` | Installed aircraft titles (and liveries) AI traffic can use, filtered by words |
| `spawn_departure` | Add a departure on a stand — pushback, taxi, line-up, take-off and SID, each on clearance; stand, runway, SID and model (type in the call sign's airline livery) chosen when not given |
| `spawn_arrival` | Add an arrival on a STAR or out on final — approach, landing, vacating and taxi to a stand |
| `list_our_traffic` | Our AI aircraft: state, position, speed, taxiway, holding point, lights and the clearances they take now |
| `atc_clearance` | Clear one of ours: `pushback`, `taxi`, `cross`, `lineup`, `takeoff`, `hold`, `abort`, `goaround`, or `remove` it from the simulator |
| `get_traffic_picture` | Every aircraft around the user aircraft or an airport, with phase (parked, taxiing, runway, departing, enroute, arriving) and airport; the user's and ours marked |
| `generate_schedule` | Realistic airline schedule for airports — call signs, types, routes, STD/STA — to pick flights to spawn (nothing is spawned) |

**Airborne ATC**

A tower per runway clears our line-ups, take-offs and crossings, and sends arrivals around when the runway won't be free. A landing sequence per runway end spaces our arrivals and the other traffic on final, with speed, a longer downwind or a hold.

| Tool | Description |
|------|-------------|
| `get_landing_sequence` | The landing sequence per runway end: order, wake category, spacing and why, distance to go, delays |
| `approach_instruction` | Instruct one of our arrivals: `up`, `down`, `slow`, `speed`, `hold`, `release`, `direct`, `joinfinal`, `holdat`, `goaround` |
| `get_atc_log` | The radio: the engine's controllers' and crews' latest transmissions |
| `get_conflicts` | Separation: closest pairs, losses, predicted conflicts and the resolutions given to ours |
| `separation_minima` | Wake categories, spacing on final, departure interval and runway occupancy for a pair of types |
| `set_player_clearance` | Tell the engine what the user's ATC cleared, so our traffic keeps off that runway and fits around the user's landing |
| `get_traffic_status` | The engine's state, settings and last error |
| `get_traffic_airport_info` | An airport as the engine works it: runways in use, ATIS, ILS, weather |

**Scheduled traffic**

| Tool | Description |
|------|-------------|
| `start_schedule` | Run scheduled traffic at airports: an airline timetable and light aircraft, departures with stand services and pushback, arrivals en route, turnarounds, overflights |
| `get_schedule` | The departure and arrival boards with status, stand, runway and estimates, and traffic time now |
| `stop_schedule` | Stop the schedule; its aircraft finish their flights, or are removed at once |
| `add_flights` | Add flights at chosen times, e.g. an arrival just before the user's ETA |
| `set_real_traffic` | Fly real-world traffic at an airport instead of the timetable |
| `observe_traffic` | Feed real-world sightings (ADS-B) to the engine, or drop them |
| `set_traffic_corridor` | Keep airliners around the user's flight in cruise: same way, opposite and crossing |

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

## License

Business Source License 1.1, see [LICENSE](LICENSE), for versions after v0.8.0. Non-commercial use is free: personal and hobby use, the flight-simulation community, education, research and non-profits. Commercial use, such as a paid add-on or product, a paid service or use inside a business, needs a separate licence; [open an issue](https://github.com/mrlm-net/simconnect-mcp/issues) to ask. Each version becomes Apache-2.0 four years after it is published, and versions up to and including v0.8.0 remain under Apache-2.0.
