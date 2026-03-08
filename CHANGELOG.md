# Changelog

All notable changes to SimConnect MCP are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project uses [Semantic Versioning](https://semver.org/).

Full release history with release notes is also available on the [GitHub Releases page](https://github.com/mrlm-net/simconnect-mcp/releases).

## [0.3.17] - 2026-03-08

### Fixed

- `Settable` detection now uses `checkmark_stem` CSS class instead of the outer `checkmark` wrapper — both the green tick (✓ settable) and the red X (✗ not settable) share an outer `<span class="checkmark">`, so the previous detection incorrectly marked every variable with any checkmark icon as settable; `checkmark_stem` only appears in the green tick variant, correctly distinguishing settable from read-only variables
- Re-scraped both corpora with the corrected settable detection: MSFS 2020 377 settable variables, MSFS 2024 366 settable variables

## [0.3.16] - 2026-03-08

### Fixed

- `ParseSimVarPage` now detects the column layout from each table's header row instead of assuming fixed column indices — pages with a `Parameters` column (e.g. `Aircraft_System_Variables.htm`) use a 5-column layout where the current parser would misread Parameters as Description and Description as Units; header-aware detection correctly maps each field regardless of column count (3, 4, or 5 columns)
- `Settable` field now correctly reads the CSS checkmark icon (`<span class="checkmark">`) used in MSFS 2020 and 2024 docs instead of looking for literal "yes"/"no" text; previously all variables were written with `settable: false`
- `ParseEventPage` now processes all tables per page — event pages use the same multi-table section layout as simvar pages; previously only the first table was parsed. Event corpus grew: MSFS 2020 263 → 1 403 entries, MSFS 2024 295 → 1 556 entries

## [0.3.15] - 2026-03-08

### Fixed

- `ParseSimVarPage` now processes **all tables** on a page instead of only the first one — SimVar pages like `Aircraft_Misc_Variables.htm` have multiple named sections (Aircraft States, Aircraft Position, Airspeed, Temperature, etc.) each rendered as a separate `<table>`; previously only the first table was parsed, silently dropping all remaining sections including fundamental variables such as `PLANE ALTITUDE`, `PLANE LATITUDE`, `PLANE LONGITUDE`, and over 150 others
- Re-scraped both corpora with the corrected parser: MSFS 2020 simvar corpus grew from 895 → 1 861 entries; MSFS 2024 from 929 → 2 042 entries

## [0.3.14] - 2026-03-08

### Fixed

- `get_airport_details` 200 ms post-END drain window is now context-aware — if the caller's context is cancelled during the drain, the wait is cut short immediately instead of always burning a fixed 200 ms regardless of cancellation

### Changed

- Added inline comments documenting the `Close()`-vs-`handleMessage` channel-close safety invariant, the `DwOutOf==0` airport-list completion edge case, the intentionally over-wide `[9]byte` ICAO cast for the MSFS 2020 33-byte stride, and the in-place `[:0]` filter idiom used in `deduplicateDetails`

## [0.3.13] - 2026-03-08

### Fixed

- `get_airport_details` now returns a deep copy of the facility data snapshot instead of a raw pointer into the live per-call state — eliminates a theoretical Go data race where a late `FACILITY_DATA` packet (dispatched after the 200 ms drain window but before the pending-map cleanup) could write to the shared struct concurrently with the caller serialising the result to JSON; all slice fields (`runways`, `frequencies`, `stands`, `helipads`, `approaches`, `departures`, `arrivals`) are copied into independent backing arrays under `state.mu` before the lock is released

## [0.3.12] - 2026-03-08

### Fixed

- `get_simvar_value` and `get_simvar_values` no longer hang indefinitely when an unknown variable name or unit is requested — each call now has a 5-second per-request deadline; if SimConnect sends no response (typically because it dispatched a `SIMCONNECT_RECV_EXCEPTION` whose packet sequence number cannot be correlated back to the waiting goroutine), the call returns an error instead of blocking forever
- `get_simvar_values` now fetches all variables in parallel instead of sequentially — a 20-variable batch completes in roughly one round-trip instead of twenty

## [0.3.11] - 2026-03-08

### Fixed

- `get_airports_in_range` and `get_nearest_airport` no longer have the same pool use-after-free race that affected `get_airport_details` — `AIRPORT_LIST` packets are now decoded inline in the SDK dispatch goroutine (inside `handleMessage`) before the pool buffer is released, instead of being forwarded through a subscription channel where the buffer pointer was no longer valid
- `get_nearby_traffic` and `get_traffic_with_phase` have the same fix applied — `SIMOBJECT_DATA_BYTYPE` packets are decoded inline in `handleMessage`; both tools previously used `SubscribeWithType` which forwarded raw pool-buffer-backed message pointers to a channel, where they were read after `Release()` had been called

## [0.3.10] - 2026-03-08

### Fixed

- `get_airport_details` no longer returns empty runways or missing frequencies on the first call — the previous implementation forwarded raw SimConnect buffer pointers through a Go channel, where they were read after the SDK had already returned the underlying memory to its pool (use-after-free). The bridge now decodes all facility data inline in the SDK dispatch goroutine, while the buffer is still valid, eliminating the race entirely
- `get_airport_details` late-arriving `FACILITY_DATA` records are no longer silently dropped — SimConnect's `FACILITY_DATA_END` and `FACILITY_DATA` messages are dispatched from different internal paths, so END can arrive before all DATA records for the same request. The fix waits 200 ms after all END messages are received, allowing any lagging DATA records to be processed before the result is returned; this replaces the previous `default:` drain loop that exited immediately if the channel was momentarily empty

## [0.3.9] - 2026-03-08

### Fixed

- `get_airport_details` expanded-only fields (`stands`, `helipads`, `approaches`, `departures`, `arrivals`) are now omitted from the JSON response entirely when `expanded=false` — they previously appeared as empty arrays, which was confusing
- Replaced zero-value defaults for inactive `facilityReqSet` slots with `noReq = 0xFFFFFFFF` sentinel; all inactive request ID slots now carry a value that can never match a real SimConnect request ID, eliminating potential switch dispatch ambiguity when multiple slots were zero
- `deduplicateDetails` now nil-guards all expanded-only slice fields before operating on them, preventing a panic on `slice[:0]` against an uninitialised nil slice

## [0.3.8] - 2026-03-08

### Added

- `get_airport_details` with `expanded=true` now also returns instrument approaches (`approaches[]`), departure procedures / SIDs (`departures[]`), and arrival procedures / STARs (`arrivals[]`)
  - Each approach includes type (ILS/VOR/RNAV/GPS/…), runway (e.g. `"08L"`), and navigation capability flags (`has_lnav`, `has_lnavvnav`, `has_lp`, `has_lpv`)
  - Each SID/STAR includes name (e.g. `"AGOL1A"`), runway transition count, and enroute transition count
  - `approach_count`, `departure_count`, `arrival_count` fields added at the top level

### Changed

- `get_airport_details` expanded mode now fires 8 facility requests (was 5); buffer increased to 1 024 messages to handle airports with many procedures
- `get_airport_details` `expanded=false` default now always includes ATC frequencies (moved from expanded-only to default); stands, helipads, approaches, SIDs, and STARs remain expanded-only

### Fixed

- `get_airports_in_range` test corrected: `IsICAOCode` accepts any 4-character code whose first letter is in the ICAO prefix table — codes like `EDB1` are valid and were never filtered by the SDK function

## [0.3.7] - 2026-03-08

### Added

- `get_airport_details` now returns magnetic variation (`magvar_deg`), closed status (`is_closed`), and the airport's actual region code (`region`) from SimConnect facility data
- `get_airport_details` now includes runways by name (e.g. `08L/26R`) — standard short/long name from `PRIMARY_NUMBER` / `PRIMARY_DESIGNATOR` and `SECONDARY_NUMBER` / `SECONDARY_DESIGNATOR`
- `get_airport_details` now returns a `helipads[]` section with lat/lon/alt, heading, dimensions, surface type, and helipad type (H / Square / Circle / Medical); `helipad_count` is included at the top level
- `get_airport_details` accepts a new `expanded` boolean parameter; ATC frequencies are now only included when `expanded=true` (reduces default response size and request overhead)

### Fixed

- `get_nearby_traffic` and `get_airports_in_range` duplicate results resolved — airport list now deduplicates by ICAO before returning
- `get_airports_in_range` standard-mode filter switched from a strict 4-uppercase-letter regex to `convert.IsICAOCode()` from the SDK, correctly admitting alphanumeric codes like `LPCC` and `LKPR` that were previously excluded
- `get_airport_details` timeout increased from 30 s to 45 s to handle cold-cache first-call latency (SimConnect fetches facility data from disk on the first query; subsequent calls for the same airport are instant)
- Removed `EDGE_LIGHTS`, `CENTER_LIGHTS`, `PRIMARY_CLOSED`, and `SECONDARY_CLOSED` from the runway facility definition — these are MSFS 2024-only fields that caused `AddToFacilityDefinition` to fail silently in MSFS 2020, preventing the runway request from ever completing and leaving the call stuck until timeout

## [0.3.6] - 2026-03-07

### Fixed

- `get_airport_details` runway `length_ft` / `width_ft` fields renamed to `length_m` / `width_m` — SimConnect's `LENGTH` and `WIDTH` fields return **metres**, not feet (empirically confirmed: EDDM runways report 4 000, matching their 4 000 m length)
- `get_airport_details` tool description now warns that the `region` parameter must match the simulator's internal value exactly; omitting it (the default) is strongly recommended

## [0.3.5] - 2026-03-07

### Fixed

- `get_airport_details` runways no longer intermittently return empty for large airports (LKPR, EDDM). SimConnect dispatches `FACILITY_DATA` records and `FACILITY_DATA_END` from different internal paths; runway records can be queued in the channel *after* their `FACILITY_DATA_END`. The fix drains the channel non-blockingly after all four END messages are received so no late-arriving data records are abandoned.
- Extracted `applyFacilityData` helper to share decoding logic between the main receive loop and the post-END drain loop.

## [0.3.4] - 2026-03-07

### Fixed

- `get_airport_details` now returns correct `runway_count` and `stand_count` even when the response arrives via the timeout path (previously counts were always 0 on timeout)
- Subscription buffer increased from 64 → 512 messages to prevent silent message loss at large airports (e.g. LKPR with 100+ parking stands); the SDK dispatcher drops messages silently when the buffer is full
- Timeout increased from 5 s → 15 s to allow large airports enough time to deliver all four `FACILITY_DATA_END` acknowledgements

## [0.3.3] - 2026-03-07

### Changed

- `get_airport_details` now issues four parallel `RequestFacilityData` calls (airport base info, runways, parking stands, ATC frequencies) instead of one, and waits for all four `FACILITY_DATA_END` messages before returning.

### Added

- `get_airport_details` response now includes:
  - `runway_count` and `runways[]` — each with `heading_deg`, `length_ft`, `width_ft`, `surface`
  - `stand_count` and `stands[]` — each with `number`, `type` (Gate Small / Ramp GA / …), `heading_deg`
  - `frequencies[]` — each with `type` (Tower / ATIS / Ground / …), `freq_mhz`, `name`

## [0.3.2] - 2026-03-07

### Changed

- `get_airports_in_range` now returns only standard ICAO airports by default (exactly 4 uppercase letters, e.g. `EDDM`, `ETSE`). Non-standard identifiers such as `EDB1`, `EDF8V`, or `GSAD3/EDB3` are excluded unless `expanded=true` is passed.

### Added

- `expanded` boolean parameter on `get_airports_in_range`: set to `true` to include all entries from the simulator's reality bubble (private fields, military strips, simulator-only codes).

## [0.3.1] - 2026-03-07

### Fixed

- SDK manager logger now writes to stderr instead of stdout; previously the default `slog.TextHandler(os.Stdout)` inside the SimConnect SDK would inject log lines into the stdio MCP pipe, causing Claude Desktop to receive non-JSON output and drop or time out `get_simvar_value` / `get_simvar_values` calls

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
