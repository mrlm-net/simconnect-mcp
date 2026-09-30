---
title: Changelog
description: Release history for SimConnect MCP.
order: 1
section: changelog
---

All notable changes to SimConnect MCP are documented here.

## [0.8.0] - 2026-09-30

A living sky around your airport. Scheduled arrivals come in from their origins en route and are handed to the arrival controller at their STAR entry; overflights cross the area at cruise level; and arrivals turn around on their stands into their later departures. Runways in use no longer flip in a calm wind, and each release's notes on GitHub are now its changelog.

### Added

- Turnarounds in scheduled traffic: an arrival whose airline and type depart again 40 min to 3 h after its STA stays on its stand and becomes that departure, the same aircraft, instead of a new aircraft on another stand. `spawn_departure` takes `turnaround_of` (one of our arrivals parked at the airport) to do the same by hand.
- En route arrivals and overflights in scheduled traffic:
  - **Arrivals** appear in the air 45 minutes before their STA, on a flight plan from their origin to the runway in use, and fly to their STAR entry as MSFS AI. There they are handed to an arrival controller flying the same STAR. An arrival that can't fly en route appears at its STAR entry as before.
  - **Overflights** cross the area within 100 NM of the first airport, between airports outside it, on routes whose great circle really crosses it. Each appears where its planned route enters the area, at its entry time. `get_schedule` lists them under `overflights`.
  - En route aircraft are created airborne as non-ATC AI and released to a waypoint chain, because MSFS 2024 places an enroute ATC aircraft on the ground at its departure airport.

### Fixed

- Departures push back with a tug (`traffic.DefaultTugTitle`), in `spawn_departure` (new `tug` parameter, default true) and the schedule. Before, they pushed back with no tug.
- `stop_schedule` without `remove` really lets the schedule's aircraft fly on. The schedule keeps running with no new spawns until they have departed, parked or left, and en route arrivals are still handed over. Before, it stopped at once, so nothing was ever removed. `stop_schedule` returns `flying_on`.
- A scheduled flight that fails is removed, so its retry under the same call sign isn't refused.
- An en route aircraft removed while the simulator is still creating it no longer stays in the simulator.

### Changed

- Release notes on GitHub are the version's section of CHANGELOG.md, not a commit list
- Runways in use stay in use while the wind allows them (up to 5 kt tailwind), in `get_active_runway`, the spawn tools and the schedule. In a calm or variable wind, arrivals and departures no longer get opposite runways from one spawn to the next.

## [0.7.0] - 2026-09-30

Scheduled traffic: a realistic airline schedule that runs itself at your airports, on the library's `TrafficManager` — departures board, push and depart on time, arrivals come in over their STARs, and the tower and landing sequences clear and space them. The website gains the traffic, ATC and schedule features and a scheduled traffic example.

### Added

- Scheduled traffic (simconnect and both modes, Windows, real bridge only): `start_schedule`, `stop_schedule` and `get_schedule` run a realistic airline schedule at airports on the library's `TrafficManager`. Departures appear on their stands before their STD and push at it, arrivals appear at a STAR entry in time for their STA, and the tower and landing sequences clear and sequence them. Departed and parked aircraft are removed. Spawns are spaced, limited and retried. Turnarounds, en route arrivals and overflights are not supported yet.
- Website: AI traffic, airborne ATC and scheduled traffic on the home page, a link to the simconnect tools reference, and a scheduled traffic example (Scenario 4)
- Tool counts: docs 15, simconnect 45, both 60

## [0.6.0] - 2026-09-30

The `github.com/mrlm-net/simconnect` library inside the server: its guides as docs tools; airport procedures, taxi routes, stands, weather, runway in use, ATIS, airways and flight plans; AI traffic of our own under ATC-style control; and, on the library's v0.16, airborne ATC — a tower per runway and landing sequences run by the server, with tools to read and work them. Plus fuel state, and airport requests that retry instead of failing.

### Added

- `list_library_guides`, `get_library_guide`, and `search_library_docs` tools (docs and both modes) — serve the 35 guides of the `github.com/mrlm-net/simconnect` Go library (client, manager, facilities, `pkg/airport`, `pkg/nav`, `pkg/traffic`, airborne separation, examples), embedded at the library version in `go.mod`; `get_library_guide` reads a whole guide or a single `##` chapter by heading or unique prefix, `search_library_docs` requires every query word to appear in a chapter, its heading, or the guide title
- `/health` in docs and both modes now reports `library_version`
- Library guides are refreshed with `go generate ./internal/libdocs/`; a test fails if the embedded guides drift from the library version in `go.mod`
- Airport procedures & ground tools (simconnect and both modes, Windows): `get_airport_procedures`, `plan_taxi_route`, `get_runway_entries_exits`, `find_stands`
- Weather & runway in use tools: `get_weather` (weather at the user aircraft; supersedes the planned milestone-6 `get_weather`), `get_active_runway`, `get_atis`
- Navigation & flight planning tools: `get_fix`, `find_airway_route`, `plan_flight` (`load_into_sim=true` changes the simulator's flight plan)
- `internal/live` runtime running the `mrlm-net/simconnect` library's `pkg/airport` and `pkg/nav` loaders on the bridge's SimConnect connection; the ten tools are registered only with the real bridge
- AI traffic tools (simconnect and both modes, Windows, real bridge only) on the library's `pkg/traffic`: `spawn_departure`, `spawn_arrival` (add AI aircraft to the simulator), `atc_clearance` (pushback, taxi, cross, lineup, takeoff, hold, abort, goaround, remove), `list_our_traffic`, `get_traffic_picture`, `list_aircraft_models`, `generate_schedule`
- Airborne ATC (simconnect and both modes, Windows, real bridge only) on the library's v0.16: a tower per runway (line-up, take-off and crossing clearances, automatic go-arounds) and a landing sequence per runway end (delays absorbed by speed, a longer downwind and a stacked hold) for our traffic not held for clearances; tools `get_landing_sequence`, `approach_instruction`, `get_atc_log`, `get_conflicts`, `separation_minima`
- `get_fuel_state` (simconnect and both modes): the user aircraft's fuel, with total quantity, capacity, percent full and weight (lb and kg), plus the center, left main and right main tanks. Tanks the aircraft does not have are left out. It completes milestone 6, whose `get_weather` came in milestone 7.
- Tool counts: docs 15, simconnect 42, both 57

### Changed

- Upgraded `github.com/mrlm-net/simconnect` SDK dependency from v0.6.0 to v0.6.1 — empty unit strings in data definitions now select the SimConnect default unit instead of raising `UNRECOGNIZED_ID` (mrlm-net/simconnect#263)
- Upgraded `github.com/mrlm-net/simconnect` SDK dependency from v0.6.1 to v0.15.0
- Upgraded `github.com/mrlm-net/simconnect` to v0.16.0 (airborne ATC). The traffic tools fly better as they are:
  - turns are the airframe's standard turn instead of MSFS AI's late hard turns at waypoints;
  - no height step when the injected final takes over;
  - an injected take-off builds up as the engines spool;
  - pushbacks give way to each other;
  - taxi clearances leave out short stub taxiways.
- Minimum Go version raised to 1.27.1 (was 1.25 with a `go1.25.8` toolchain); Docker builder image moved to `golang:1.27-alpine`
- Bridge SimConnect definition and request IDs now start at 1,000,002, clear of the library's default IDs

### Fixed

- AI aircraft spawned by the traffic tools are removed from the simulator when the server stops (Ctrl+C, SIGTERM, or stdin closing in stdio mode); the HTTP server now shuts down gracefully and the SimConnect connection is closed
- `both` mode on Windows now registers the six navaid tools (`get_vors_in_range`, `get_vor_details`, `get_ndbs_in_range`, `get_ndb_details`, `get_waypoints_in_range`, `get_waypoint_details`), which were previously missing
- Documentation: corrected the tool counts, documented `list_simvar_categories` and `MCP_MODE=both`, and brought the `llm.txt` tool list up to date
- Airport details: SimConnect now and then drops a facility request's messages. `get_airport_details` retries up to three times (20 s each, 200 ms apart) and answers `TIMEOUT` if the simulator never answers, instead of "not found" after 45 s.
- Taxiways and parkings: an airport without the data (`TAXIWAY_NOT_FOUND`, `PARKING_NOT_FOUND`) is now told apart from one the simulator did not answer for (`TIMEOUT`, try again).

### Security

- Upgraded `golang.org/x/net` v0.52.0 → v0.59.0 (GO-2026-5025, GO-2026-5027, GO-2026-5028, GO-2026-5029, GO-2026-5030) and dependent `golang.org/x/crypto`, `golang.org/x/sys`, `golang.org/x/text`
- Go 1.27.1 toolchain resolves the standard-library advisories reported by `govulncheck` against Go 1.25.8

## [0.5.8] - 2026-03-15

### Changed

- Upgraded `github.com/mrlm-net/simconnect` SDK dependency from v0.4.2 to v0.6.0

## [0.5.7] - 2026-03-15

### Added

- `get_taxiway_names` tool — lightweight alternative to `get_airport_taxiways` that returns only the taxiway letter/name strings (no paths or points); avoids token-limit issues for large airports

### Fixed

- `get_simvar_value` and `get_simvar_values`: unknown or misspelled SimVar names now return a structured `UNKNOWN_VARIABLE` error instead of timing out with a generic message
- `get_nearby_traffic` and `get_traffic_with_phase`: `sim_time` (Zulu seconds since midnight) is now included at the envelope level in all responses
- `get_airport_details`: supplying an incorrect `region` code no longer silently drops the result — the bridge automatically retries with an empty region and logs a WARN
- `get_airport_taxiways`: added `max_paths` parameter (default 500, max 2000) to cap large responses; response includes `truncated` and `truncated_to` fields when paths are capped

## [0.5.6] - 2026-03-14

### Added

- Website: SDK CTA section — promotes the `github.com/mrlm-net/simconnect` Go SDK with copyable `go get` command and links to `simconnect.mrlm.net`
- Website: "Support the project" sponsor section with "Sponsor via Revolut" link

## [0.5.5] - 2026-03-14

### Fixed

- Website: 404 on `/docs/*` URLs crashed — `(docs)` and `(marketing)` group layouts now fall back to directly-imported `siteConfig` when layout server data is unavailable (GitHub Pages static 404 fallback context)
- Website: added `(docs)/+error.svelte` so docs-group 404s render inside the docs layout chrome instead of double-rendering a header/footer

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
