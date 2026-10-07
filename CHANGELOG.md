# Changelog

All notable changes to SimConnect MCP are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project uses [Semantic Versioning](https://semver.org/).

Full release history with release notes is also available on the [GitHub Releases page](https://github.com/mrlm-net/simconnect-mcp/releases).

## [0.12.0] - 2026-10-07

On simconnect v0.25.0: our traffic gives way to a crossing tail, climbs to its cleared level and shuts down when parked; aircraft can follow one another on the ground; and the website tells you how to use SimConnect MCP commercially.

### Added

- `atc_clearance` `follow` with `behind`: one of ours stays behind another of ours on the ground wherever their ways meet, told the way ground says it (the library's v0.25).
- Website: a "Commercial use" section on the home page, before "Support the project", with a contact at support@mrlm.net.

### Changed

- Upgraded `github.com/mrlm-net/simconnect` from v0.24.0 to v0.25.0. With it:
  - our traffic gives way to a crossing tail;
  - departures climb to their cleared level;
  - arrivals stop their engines once parked, and get one dog-leg, lengthened in place;
  - an arrival's route is no longer taken from the wrong STAR point when it was asked for before the aircraft's first position.

## [0.11.0] - 2026-10-07

TCAS for our traffic, on simconnect v0.24.0: traffic and resolution advisories with coordinated RAs, AI aircraft that taxi round pushbacks and climb out with an eased pitch, and docs brought up to date with v0.10.0.

### Added

- `get_tcas`: TCAS II for our airborne traffic (the library's v0.24): traffic and resolution advisories now, and the latest as they happened. Each of ours flies its RA, reports it on the frequency and is coordinated with the other of ours; `list_our_traffic` shows each aircraft's advisory as `tcas`.
- Tool counts: docs 15, simconnect 59, both 74

### Changed

- Upgraded `github.com/mrlm-net/simconnect` from v0.23.1 to v0.24.0. With it:
  - routes and taxiing aircraft keep clear of pushbacks;
  - gear goes up at 150–400 ft with an eased take-off pitch;
  - runway entries at the same distance come in a fixed order.
- The traffic engine writes its console lines to stderr (the library's new `Output` option) instead of stdout.

- Docs brought up to date with v0.10.0:
  - Scenario 4 in Examples shows the traffic engine's `get_schedule` and radio shapes.
  - The `/health` examples in Getting Started use the real fields (`sim_connected`, `connection_state`).
  - `DOCS_LIVE_SCRAPE` and `confirm_live_scraping` are documented, and the README's environment table lists the two new variables.
  - Tool counts in the API index, guide sections and the library version in the docs-mode reference are corrected.
  - Prepar3D and FSX mentions are removed (the server supports MSFS 2020 and 2024).
  - The website's home page gets cards for the user aircraft tools, real-world traffic and traffic around you in cruise, and its hero says Go 1.27+.

## [0.10.0] - 2026-10-07

AI traffic on the library's traffic engine, the airport map's: stand services and tugs from each airport's fleet, crews on the radio, flights at chosen times, real-world traffic, airliners around you in cruise, and your own runway clearances respected. And the user aircraft: its systems through the library's aircraft profiles, its doors, ground equipment and the sim's ground services, its radios, squawk and ATC call sign, and the add-ons installed. The server now runs on simconnect v0.23.1, with 45 library guides.

### Added

- User aircraft tools (simconnect and both modes, Windows), on the library's `pkg/systems`, `pkg/avionics` and `pkg/addons`:
  - `get_aircraft_systems`: power, COM radios and frequencies, engines, parking brake, lights, doors by name, transponder, flaps and gear, chocks and GPU, cabin signs and the sim's pushback state, read through the aircraft's systems profile: the standard SimVars, the library's profile for the model on top (the Fenix A320 family on its own L:vars and tablet), then local overrides. It names the profile, the aircraft's package and what can be operated.
  - `set_aircraft_control`: open or close a door by its name, set the chocks, GPU, parking brake, seat belt and no smoking signs or external power, or call the cabin, the way the aircraft's profile says.
  - `request_ground_service`: the sim's jetway, stairs, baggage, catering, ground power, fuel truck or pushback.
  - `set_radio`: a COM's active or standby frequency, a COM swap (the Fenix's RMP transfer key), or the squawk.
  - `set_atc_callsign`: the call sign the sim's ATC uses (ATC AIRLINE, ATC FLIGHT NUMBER).
  - `list_addons`: the installed MSFS packages, Community, Official and streamed, with streamed airports by ICAO.
- `SIMCONNECT_AIRCRAFT_PROFILES`: a directory of local aircraft profile overrides (`pkg/systems` JSON), winning per value over the shipped profiles.
- AI traffic on the library's traffic engine, `pkg/traffic/world` (the airport map's), on the server's connection: departures get stand services from each airport's fleet (fuel truck; stairs and GPU at remote stands) and a tug, crews talk to delivery, ground, tower and approach, and the engine runs again after a simulator reconnect with its settings. New tools:
  - `add_flights`: flights at chosen times in the running schedule (std_in_min / sta_in_min).
  - `set_real_traffic`, `observe_traffic`: fly real-world aircraft from a feed (ADS-B sightings) instead of the timetable.
  - `set_traffic_corridor`: airliners around the user's flight in cruise (same way, opposite, crossing).
  - `set_player_clearance`: what the user's ATC cleared, so our traffic keeps off that runway and fits around the user's landing.
  - `get_traffic_status`, `get_traffic_airport_info`: the engine's state, and an airport as it works it (runways in use, ATIS, ILS).
- `SIMCONNECT_TRAFFIC_DATA`: where the traffic engine keeps each airport's airways and its data files.
- Tool counts: docs 15, simconnect 58, both 73

### Changed

- Upgraded `github.com/mrlm-net/simconnect` from v0.18.4 to v0.23.1. The library guides served by the docs tools now include 45 guides (new: aircraft systems profiles, radios and transponder, installed add-ons, replaceable dictionaries, VFR traffic and the traffic world engine). The airport, navigation, traffic and ATC tools pick up the library's fixes since v0.18.4, among them the runway surface from the facility data, STAR descents with their constraints, and variable wind.
- The traffic tools moved from the server's own runtime (the library's `TrafficManager`) to the library's traffic engine. Tool names stay; what changed:
  - `atc_clearance` takes more actions (`pushstart`, `startup`, `upto`, `lineupbehind`, `entry`, `rush`, `land`, `standto`, `depart`, `manual`), and any action but `remove` makes the aircraft manual.
  - `approach_instruction` adds `speed`, `joinfinal` and `holdat`.
  - `spawn_departure` adds `fuel`, `push_in_min`, `stand_use` and `squawk`, and drops `turnaround_of`. `spawn_arrival` adds `turnaround` and `dwell_min`, and drops `spawn_nm`.
  - `list_our_traffic`, `get_landing_sequence`, `get_conflicts`, `get_schedule` and `get_atc_log` return the engine's views; `get_atc_log` is now the radio.
  - `start_schedule` adds `ifr`, `vfr`, `generator`, `offset_min` and `others`.
  - `stop_schedule` without `remove` returns at once; the aircraft finish their flights.
- Stdio transport: anything else printed to stdout (the traffic engine logs there) goes to stderr, so it can't break the protocol.

## [0.9.0] - 2026-10-02

New versions are under the Business Source License 1.1: free for non-commercial use, Apache-2.0 four years after each release. The server now runs on simconnect v0.18.4, with 39 library guides (radio, phraseology, traffic decisions, camera), and the docs gain an AI Traffic & ATC guide.

### Added

- Docs: an **AI Traffic & ATC** guide (a new Guides section and a top link on the website): what you need, spawning by hand or by schedule, what happens to each flight, the tower and approach controller, things to ask, and limits. The home page, README and Getting Started now describe the traffic features. The README lists the airborne ATC, scheduled traffic and fuel tools, and no longer says features that shipped in v0.6.0 are unavailable.

### Changed

- License: new versions are under the Business Source License 1.1 instead of Apache-2.0. Non-commercial use (personal and hobby use, the flight-simulation community, education, research, non-profits) is allowed; commercial use, such as a paid add-on or product, a paid service or use inside a business, needs a separate licence. Each version becomes Apache-2.0 four years after it is published. Versions up to and including v0.8.0 stay under Apache-2.0.
- Upgraded `github.com/mrlm-net/simconnect` from v0.16.0 to v0.18.4. The library guides served by the docs tools now include 39 guides (new: traffic radio and phraseology, how the traffic decides, and the add-on camera). Airport tools pick up the library's layout fixes, for example `get_runway_entries_exits` now lists entry Z onto runway 24 at LKPR.

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
- Airport procedures & ground tools (simconnect and both modes, Windows): `get_airport_procedures` (SIDs, STARs and approaches, or one resolved into its points), `plan_taxi_route` (ATC-style taxi route between a stand and a runway, departure or arrival), `get_runway_entries_exits`, `find_stands` (stands that fit a wing span, airline, gates only)
- Weather & runway in use tools: `get_weather` (weather at the user aircraft; SimConnect gives no gusts, ceiling or dewpoint — supersedes the planned milestone-6 `get_weather`), `get_active_runway` (runways in use, wind components, expected approach, transition altitude and level), `get_atis` (ATIS as text and as spoken)
- Navigation & flight planning tools: `get_fix` (waypoint, VOR or NDB with its airways), `find_airway_route` (airway route between two fixes; the airway network is crawled on demand, cached, radius capped at 400 NM), `plan_flight` (IFR plan with SID, airways, STAR, approach, cruise level, time and fuel; `load_into_sim=true` loads it as the simulator's flight plan via a `.pln` in `%TEMP%\simconnect-mcp` and `SimConnect_FlightPlanLoad`)
- `internal/live` runtime: runs the `mrlm-net/simconnect` library's `pkg/airport` and `pkg/nav` loaders (layouts and taxi graphs, procedures, fixes, airway crawls, weather) on the bridge's SimConnect manager; the ten tools above are registered only with the real bridge, not the mock
- AI traffic tools on the library's `pkg/traffic` (simconnect and both modes, Windows, real bridge only): `spawn_departure` and `spawn_arrival` add AI aircraft of our own to the simulator (stand, runway, SID/STAR and model — the type in the call sign's airline livery — chosen when not given), `atc_clearance` clears them (`pushback`, `taxi`, `cross`, `lineup`, `takeoff`, `hold`, `abort`, `goaround`, `remove`), `list_our_traffic` follows them, `get_traffic_picture` shows every aircraft around the user aircraft or an airport with its phase, `list_aircraft_models` lists the installed titles and liveries, `generate_schedule` builds a realistic airline schedule; at most 32 of ours, at an airport loaded around the user aircraft
- Airborne ATC (simconnect and both modes, Windows, real bridge only), on the library's v0.16:
  - **Runway clearances:** the live runtime runs a tower per runway. It clears our departures' line-up and take-off and our runway crossings in mixed mode, and sends arrivals around when the runway won't be free (published missed approach, re-sequenced).
  - **Landing sequence:** a sequence per runway end (weather-dependent spacing, at least 5 NM). Our arrivals lose their delays by speed, then a longer downwind, then a stacked hold.
  - **New tools:** `get_landing_sequence` (the sequence and the tower), `approach_instruction` (up, down, slow, hold, release, direct, go-around), `get_atc_log`, `get_conflicts` (prediction with the least disturbing resolution as advice) and `separation_minima` (wake categories, spacing on final, departure interval, runway occupancy).
  - **Spawns:** departures spawned without `hold_for_clearances` now wait for the tower at the runway, and `spawn_arrival` passes the approach's published missed approach.
- `get_fuel_state` (simconnect and both modes): the user aircraft's fuel, with total quantity, capacity, percent full and weight (lb and kg), plus the center, left main and right main tanks. Tanks the aircraft does not have are left out. It completes milestone 6, whose `get_weather` came in milestone 7.
- Tool counts: docs 15, simconnect 42, both 57

### Changed

- Bridge SimConnect data definition and request IDs now start at 1,000,002, clear of the library components' fixed default IDs (`airport.ProcedureLoader` uses 8400–8508)
- Upgraded `github.com/mrlm-net/simconnect` SDK dependency from v0.6.0 to v0.6.1 — empty unit strings in data definitions now select the SimConnect default unit instead of raising `UNRECOGNIZED_ID` (mrlm-net/simconnect#263)
- Upgraded `github.com/mrlm-net/simconnect` SDK dependency from v0.6.1 to v0.15.0
- Upgraded `github.com/mrlm-net/simconnect` to v0.16.0 (airborne ATC). The traffic tools fly better as they are:
  - turns are the airframe's standard turn instead of MSFS AI's late hard turns at waypoints;
  - no height step when the injected final takes over;
  - an injected take-off builds up as the engines spool;
  - pushbacks give way to each other;
  - taxi clearances leave out short stub taxiways.
- Minimum Go version raised to 1.27.1 (was 1.25 with a `go1.25.8` toolchain); Docker builder image moved to `golang:1.27-alpine`

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

### Changed

- Website: simplified the SDK CTA and sponsor sections on the homepage

## [0.5.6] - 2026-03-14

### Added

- Website: SDK CTA section on the homepage — introduces the underlying `github.com/mrlm-net/simconnect` Go SDK with a copyable `go get` command, "Get Started" and "View Documentation" links to `simconnect.mrlm.net`
- Website: "Support the project" sponsor section on the homepage — "Sponsor via Revolut" link for funding infrastructure, development time, and MSFS licence costs

## [0.5.5] - 2026-03-14

### Fixed

- Website: 404 pages on `/docs/*` URLs crashed with `Cannot read properties of undefined (reading 'title')` because the `(docs)` group layout reads `data.siteConfig` from the layout server load, which is unavailable when GitHub Pages serves the static `404.html` fallback; both group layouts (`docs`, `marketing`) now fall back to a directly-imported `siteConfig` constant when `data.siteConfig` is undefined
- Website: added `(docs)/+error.svelte` so error pages inside the docs group render inside the docs chrome (header, sidebar, footer) rather than rendering a second standalone header/footer from the root error page

## [0.5.4] - 2026-03-14

### Fixed

- Website: "Back to home" and "Getting started" links on the 404 page caused a client-side navigation crash (`Cannot read properties of undefined (reading 'title')`); links now use `data-sveltekit-reload` to force a full page load, bypassing SvelteKit's client-side router which cannot resolve layout data from the GitHub Pages 404 fallback context

### Changed

- README: live SimConnect tools table expanded from 4 tools to all 19 current tools, organised into four groups (simulation variables, traffic, airports, navigation facilities)
- README: completed milestone sections removed; contributing section simplified

## [0.5.3] - 2026-03-14

### Added

- Website: custom 404 error page with aviation-themed random messages (TERRAIN PULL UP, SQUAWK 7700, GO AROUND, etc.), full site header/footer, and links back to home and getting started

### Fixed

- Website: 404 page crashed with `Cannot read properties of undefined (reading 'title')` on GitHub Pages because `page.data` is empty for unmatched static routes; `siteConfig` is now imported directly from `$lib/config/site.js` instead of relying on layout server data

## [0.5.2] - 2026-03-14

### Added

- `list_simvar_categories` — new MCP tool that returns a sorted, deduplicated list of all SimVar category strings present in the loaded corpus; use the returned values as the exact `category` filter argument for `list_simvars`
- HTTP 404 responses now pick from a pool of aviation-themed messages at random (e.g. "TERRAIN PULL UP — no route found at this altitude", "Squawk 7700 — this endpoint is a navigation emergency")

### Fixed

- `get_traffic_with_phase` — AI aircraft that SimConnect reports as airborne at very low altitude (< 100 ft MSL) with near-zero ground speed (< 2 kts) and near-zero vertical speed (< 100 fpm) are now classified as `PARKED` instead of `LEVEL`; this corrects a common SimConnect issue where a parked AI aircraft's model reference point is several feet above the gear contact point, causing `on_ground` to read `false`
- `search_docs` — query matching is now token-based (all whitespace-separated words must appear somewhere in the name or description, order-independent) instead of a consecutive substring match; queries like `"parking brake"` now correctly find `BRAKE PARKING POSITION`

## [0.5.1] - 2026-03-13

### Added

- `Dockerfile` and `.dockerignore` for docs mode — multi-stage build using `golang:1.25-alpine` builder and `gcr.io/distroless/static-debian12:nonroot` runtime; set `MCP_MODE=docs` and `GIN_MODE=release` by default; exposes port 8080; accepts `VERSION` build arg for version embedding

### Fixed

- Closed stale packaging backlog issues (#60–#80) that were already implemented — version embedding, release workflow (GoReleaser), CHANGELOG.md, README installation docs

## [0.5.0] - 2026-03-13

### Added

- `get_airport_taxiways` — return the taxiway network graph for a specific airport by ICAO code; response contains three correlated arrays: `names` (taxiway letter strings), `paths` (directed edges referencing start/end node indices and a name index), and `points` (graph nodes including hold-short positions); uses `RequestFacilityData` with TAXI_NAME, TAXI_PATH, and TAXI_POINT sub-requests
- `get_airport_parkings` — return all parking stands, gates, and ramps at a specific airport by ICAO code; each entry includes type, name, suffix, number, heading (degrees true), radius (metres), and position offsets (`bias_x_m`, `bias_z_m`) from the airport reference point; returns the full TAXI_PARKING record with more fields than the stands array in `get_airport_details`

### Fixed

- `get_airport_details`, `get_airport_taxiways`, and `get_airport_parkings` now validate `icao` (1–9 uppercase alphanumeric) and `region` (0–4 uppercase alphanumeric) with the same regex guards used by the navaid tools; previously raw user input was passed directly as C-string arguments to the SimConnect SDK

### Changed

- Upgraded `golang.org/x/net` v0.51.0 → v0.52.0 (pulls in `x/crypto`, `x/sys`, `x/text` patch releases)

## [0.4.0] - 2026-03-11

### Added

- `get_vors_in_range` — list VOR navigation stations in the simulator's reality bubble sorted by distance from the player aircraft; configurable radius (default 200 km, max 500 km); each entry includes ICAO, region, lat/lon, altitude (metres MSL), frequency (Hz), magnetic variation, and distance (km)
- `get_vor_details` — query detailed VOR data by ICAO code; returns position, frequency (Hz and MHz), magnetic variation, nav range (NM), and capability flags (`is_nav`, `is_dme`, `is_tacan`, `has_glide_slope`, `has_back_course`)
- `get_ndbs_in_range` — list NDB navigation stations in the simulator's reality bubble sorted by distance; configurable radius (default 200 km, max 500 km); each entry includes ICAO, region, lat/lon, altitude (metres MSL), frequency (Hz), magnetic variation, and distance (km)
- `get_ndb_details` — query detailed NDB data by ICAO code; returns position, frequency (Hz and kHz), type, range, magnetic variation, name, and terminal flag
- `get_waypoints_in_range` — list waypoints in the simulator's reality bubble sorted by distance; configurable radius (default 100 km, max 500 km) and count limit (default 200, max 1000); each entry includes ICAO, region, lat/lon, altitude (metres MSL), magnetic variation, and distance (km)
- `get_waypoint_details` — query detailed waypoint data by ICAO code; returns position, type, magnetic variation, number of airways routes, and terminal flag

## [0.3.20] - 2026-03-08

### Fixed

- `get_simvar_values` now registers all requested variables on a **single data definition** with sequential datum indices and issues one `RequestDataOnSimObject` call, receiving all values in one `SIMOBJECT_DATA` packet — the previous implementation spawned one goroutine per variable each creating its own definition, which caused SimConnect to reject the second concurrent `AddToDataDefinition` call with `E_FAIL (0x80004005)`; the single-definition approach matches the pattern already used by `get_nearby_traffic` and `get_traffic_with_phase`

## [0.3.19] - 2026-03-08

### Fixed

- `get_simvar_values` batch now executes sequentially instead of in parallel — SimConnect does not support concurrent `AddToDataDefinition` / `RequestDataOnSimObject` calls from multiple goroutines; parallel execution caused requests to be silently dropped or rejected with `E_FAIL (0x80004005)`, resulting in timeouts for all variables in the batch including ones that work correctly when fetched individually

## [0.3.18] - 2026-03-08

### Fixed

- `GetSimVar`, `SetSimVar`, `GetTraffic`, and `GetEnrichedTraffic` now return `BRIDGE_DISCONNECTED` instead of a raw SimConnect `HRESULT 0x80004005` error when the connection drops between the initial state check and the `AddToDataDefinition` call; the state is re-checked immediately after any `AddToDataDefinition` failure so transient disconnections surface as a clean reconnectable error rather than an opaque internal failure

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
