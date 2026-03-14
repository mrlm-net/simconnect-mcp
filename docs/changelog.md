---
title: Changelog
description: Release history for SimConnect MCP.
order: 1
section: changelog
---

All notable changes to SimConnect MCP are documented here.

## [0.5.4] - 2026-03-14

### Fixed

- Website: 404 page navigation links crash fixed — `data-sveltekit-reload` forces full page load instead of client-side routing, which cannot resolve layout data from the GitHub Pages fallback context

### Changed

- README: live SimConnect tools table expanded to all 19 current tools (simulation variables, traffic, airports, navigation facilities)

## [0.5.3] - 2026-03-14

### Added

- Website: custom 404 error page with aviation-themed random messages, header, footer, and navigation back to home

### Fixed

- Website: 404 page `siteConfig` crash on GitHub Pages — import directly from config instead of `page.data`

## [0.5.2] - 2026-03-14

### Added

- `list_simvar_categories` — returns a sorted list of all SimVar category strings; use the exact values as the `category` filter for `list_simvars`
- HTTP 404 responses now rotate through aviation-themed messages at random

### Fixed

- `get_traffic_with_phase` — parked AI aircraft reported as airborne by SimConnect (on_ground=false, alt < 100 ft, GS < 2 kts, |VS| < 100 fpm) are now correctly classified as `PARKED` instead of `LEVEL`
- `search_docs` — token-based matching; "parking brake" now finds "BRAKE PARKING POSITION" regardless of word order

## [0.5.1] - 2026-03-13

### Added

- `Dockerfile` for docs mode — multi-stage build, distroless nonroot runtime, exposes port 8080; run with `docker run -p 8080:8080 ghcr.io/mrlm-net/simconnect-mcp:latest`

## [0.5.0] - 2026-03-13

### Added

- `get_airport_taxiways` — return the taxiway network graph for a specific airport by ICAO code; response contains three correlated arrays: `names` (taxiway letter strings), `paths` (directed edges with start/end node indices and a name reference), and `points` (graph nodes including hold-short positions)
- `get_airport_parkings` — return all parking stands, gates, and ramps at a specific airport by ICAO code; each entry includes type, name, suffix, number, heading, radius, and position offsets from the airport reference point

### Fixed

- `get_airport_details`, `get_airport_taxiways`, and `get_airport_parkings` now validate `icao` (1–9 uppercase alphanumeric) and `region` (0–4 uppercase alphanumeric) before passing values to the SimConnect SDK

### Changed

- Upgraded `golang.org/x/net` to v0.52.0

## [0.4.0] - 2026-03-11

### Added

- `get_vors_in_range` — list VOR navigation stations in the simulator's reality bubble sorted by distance; configurable radius (default 200 km, max 500 km); each entry includes ICAO, region, lat/lon, altitude (metres MSL), frequency (Hz), magnetic variation, and distance (km)
- `get_vor_details` — query detailed VOR data by ICAO code; returns position, frequency (Hz and MHz), magnetic variation, nav range (NM), and capability flags (`is_nav`, `is_dme`, `is_tacan`, `has_glide_slope`, `has_back_course`)
- `get_ndbs_in_range` — list NDB navigation stations sorted by distance; configurable radius (default 200 km, max 500 km)
- `get_ndb_details` — query detailed NDB data by ICAO code; returns position, frequency (Hz and kHz), type, range, magnetic variation, name, and terminal flag
- `get_waypoints_in_range` — list waypoints sorted by distance; configurable radius (default 100 km, max 500 km) and count limit (default 200, max 1000)
- `get_waypoint_details` — query detailed waypoint data by ICAO code; returns position, type, magnetic variation, number of airways routes, and terminal flag

## [0.3.20] - 2026-03-08

### Fixed

- `get_simvar_values` now registers all requested variables on a single data definition and issues one `RequestDataOnSimObject` call, fixing `E_FAIL (0x80004005)` errors caused by concurrent `AddToDataDefinition` calls in the previous per-variable goroutine approach

## [0.3.0] - 2026-03-07

### Added

- `get_nearby_traffic` — scan for AI and multiplayer aircraft within a configurable radius (default 25 km, max 200 km)
- `get_traffic_with_phase` — enriched traffic scan with vertical speed, ground track, inferred flight phase, parking state, runway occupancy, and aircraft category
- `get_airports_in_range` — list airports in the simulator's reality bubble sorted by distance; configurable radius (default 50 km, max 500 km)
- `get_nearest_airport` — return the single closest airport with ICAO, region, lat/lon, altitude (metres MSL), and distance (km)
- `get_airport_details` — query detailed facility data for a specific ICAO airport including name, coordinates, runways, ATC frequencies, and optional stands/approaches/SIDs/STARs

## [0.2.0] - 2026-03-05

### Added

- Live SimConnect mode (`MCP_MODE=simconnect`, Windows only) — CGo/FFI bridge to SimConnect.dll via `github.com/mrlm-net/simconnect` SDK
- `get_simvar_value` — read a single live simulation variable from the running simulator
- `get_simvar_values` — batch read up to 20 simulation variables in one call
- `set_simvar_value` — write a numeric simulation variable to the user aircraft
- `transmit_event` — send a named SimConnect client event to the simulator
- `get_sim_state` — simulator connection state snapshot with auto-reconnect on simulator restart

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
