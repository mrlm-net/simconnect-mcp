# Changelog

All notable changes to SimConnect MCP are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project uses [Semantic Versioning](https://semver.org/).

Full release history with release notes is also available on the [GitHub Releases page](https://github.com/mrlm-net/simconnect-mcp/releases).

## [0.3.0] - 2026-03-07

### Added

- `get_nearby_traffic` — scan for AI and multiplayer aircraft within a configurable radius (default 25 km, max 200 km); returns object ID, title, ATC callsign, airline, position, speed, heading, and on-ground flag
- `get_traffic_with_phase` — enriched traffic scan in a single SimConnect round-trip; adds vertical speed (fpm), actual ground track (from velocity vectors), inferred flight phase (PARKED / TAXI / CLIMB / CLIMB SHALLOW / LEVEL / DESCENT / APPROACH / FINAL), parking state, runway occupancy, and aircraft category
- `get_airports_in_range` — list airports in the simulator's reality bubble sorted by distance from the player; configurable radius (default 50 km, max 500 km)
- `get_nearest_airport` — return the single closest airport with ICAO, region, lat/lon, altitude (metres MSL), and distance (km)
- `get_airport_details` — query detailed facility data for a specific ICAO airport; returns full name, lat/lon, altitude (metres MSL); optional region parameter for disambiguation

## [0.2.0] - 2026-03-05

### Added

- `set_simvar_value` — write a numeric simulation variable to the user aircraft (autopilot targets, flight controls, and other writable SimVars)
- `get_sim_state` now returns full aircraft position (latitude, longitude, altitude), speed (ground speed, indicated airspeed, vertical speed), true heading, and on-ground flag
- `get_sim_state` now returns `simulator_version` string captured on connection open

### Fixed

- `get_sim_state` position, speed, and heading fields returned garbage values due to a struct alignment bug in the underlying SDK — fixed by upgrading to SDK v0.4.2 which uses a uniform float64 layout
- `transmit_event` events now reliably reach the simulator; previously used an incorrect event flag that caused events to be silently discarded in some scenarios
- `.mcp.json` Claude Code integration now runs in `both` mode, exposing documentation and live SimConnect tools simultaneously

### Changed

- Updated `github.com/mrlm-net/simconnect` SDK dependency from v0.3.7 to v0.4.2

## [0.1.1] - 2026-03-01

### Fixed

- Install command in hero no longer wraps to a second line; scrollbar hidden while scroll is preserved
- Removed empty Unreleased section from changelog

## [0.1.0] - 2026-03-01

### Added

- Documentation website at [simconnect-mcp.mrlm.net](https://simconnect-mcp.mrlm.net)
- Stdio transport — server auto-detects pipe and switches to stdio for Claude Code and other local MCP clients
- Claude Code integration via `.mcp.json` and documentation at `/docs/claude-code`
- GPS variables for MSFS 2020 (430 active) and MSFS 2024 (430 deprecated) across 5 subcategories
- GoReleaser cross-platform binary builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64)
- `--version` flag with ldflags-embedded version, commit, and build date
- Live SimConnect mode (`MCP_MODE=simconnect`, Windows only) with CGo/FFI bridge to SimConnect.dll
- `get_simvar_value` — read a single live simulation variable from a running simulator
- `get_simvar_values` — batch read up to 20 simulation variables
- `transmit_event` — send SimConnect client events to the simulator
- `get_sim_state` — simulator connection state snapshot
- Auto-reconnect on simulator restart
- 11 MCP tools for SimConnect SDK documentation: `list_simvars`, `get_simvar`, `list_events`, `get_event`, `list_functions`, `get_function`, `list_structures`, `get_structure`, `list_error_codes`, `get_error_code`, `search_docs`
- Full-text search across SimConnect SDK corpus
- Pagination support for all list tools (default 20, max 100 per page)
- MSFS 2020 and MSFS 2024 documentation corpus (1,800+ simulation variables)
- MCP-over-HTTP/SSE and streamable HTTP transports

### Changed

- Server startup checks `MCP_MODE` environment variable (`docs` default, `simconnect`, `both`)
