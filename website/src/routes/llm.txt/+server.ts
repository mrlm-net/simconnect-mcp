import type { RequestHandler } from './$types.js';
import { loadDocIndex } from '$lib/content/pipeline.server.js';
import { siteConfig } from '$lib/config/site.js';

export const prerender = true;

export const GET: RequestHandler = () => {
    const baseUrl = `${siteConfig.url}${siteConfig.basePath}`;
    const docs = loadDocIndex();
    docs.sort((a, b) => a.order - b.order);

    const docList = docs
        .map((doc) => `- [${doc.title}](${baseUrl}/docs/${doc.slug}): ${doc.description}`)
        .join('\n');

    const content = `# ${siteConfig.title}

> ${siteConfig.description}

## Key URLs

- Website: ${baseUrl}/
- Documentation: ${baseUrl}/docs/
- Repository: ${siteConfig.repoUrl}

## MCP Tools — Docs Mode (MCP_MODE=docs, 15 tools)

SimConnect SDK reference:

- list_simvar_categories: List all SimVar category filter values for list_simvars
- list_simvars: List SimConnect simulation variables with pagination and category filter
- get_simvar: Fetch a single simulation variable by name
- list_events: List SimConnect client input events (Key Event IDs)
- get_event: Fetch a single client event by name
- list_functions: List SimConnect C API functions
- get_function: Fetch a single SDK function by name
- list_structures: List SimConnect C data structures
- get_structure: Fetch a single data structure by name
- list_error_codes: List SIMCONNECT_EXCEPTION enum values
- get_error_code: Fetch an error code by name or integer value
- search_docs: Full-text search across all corpus types

github.com/mrlm-net/simconnect Go library guides (v0.24.0):

- list_library_guides: List the library guides and their chapters, optionally filtered by section
- get_library_guide: Read a library guide, or a single chapter of it, as Markdown
- search_library_docs: Search the library guides chapter by chapter

## MCP Tools — SimConnect Mode (MCP_MODE=simconnect, Windows only, 59 tools)

- get_simvar_value: Read a single live simulation variable from the running simulator
- get_simvar_values: Read up to 20 simulation variables in a single call
- set_simvar_value: Write a numeric simulation variable to the user aircraft
- transmit_event: Transmit a named SimConnect client event to the simulator
- get_sim_state: Return a snapshot of current simulator connection state and flight status
- get_fuel_state: The user aircraft's fuel: total quantity, capacity, weight and the main tanks
- get_nearby_traffic: List AI and player aircraft within a radius of the user aircraft
- get_traffic_with_phase: Nearby traffic with enriched telemetry and inferred flight phase
- get_airports_in_range: List airports in the loaded scenery area sorted by distance
- get_nearest_airport: Return the single closest airport to the player aircraft
- get_airport_details: Return detailed facility data for an airport by ICAO code
- get_airport_taxiways: Return the taxiway network graph for an airport
- get_taxiway_names: Return only the taxiway letter/name strings for an airport
- get_airport_parkings: Return all parking stands, gates, and ramps at an airport
- get_vors_in_range: List VOR stations sorted by distance from the player aircraft
- get_vor_details: Return detailed data for a VOR by ICAO code
- get_ndbs_in_range: List NDB stations sorted by distance from the player aircraft
- get_ndb_details: Return detailed data for an NDB by ICAO code
- get_waypoints_in_range: List waypoints sorted by distance from the player aircraft
- get_waypoint_details: Return detailed data for a waypoint by ICAO code
- get_airport_procedures: List an airport's SIDs, STARs and approaches, or resolve one into its points
- plan_taxi_route: ATC-style taxi route between a parking stand and a runway (departure or arrival)
- get_runway_entries_exits: Taxiways onto a runway end and the exits from it
- find_stands: Parking stands that fit an aircraft, by wing span, airline and gate
- get_weather: Weather at the user aircraft (no gusts, ceiling or dewpoint)
- get_active_runway: Runways in use, wind components, expected approach, transition altitude and level
- get_atis: ATIS broadcast composed from the simulator's weather, as text and as spoken
- get_fix: Waypoint, VOR or NDB with its position, frequency and airways
- find_airway_route: Airway route between two enroute fixes, e.g. VOZ M725 OKF
- plan_flight: IFR flight plan between two airports (SID, airways, STAR, approach, profile, fuel); optionally loads it into the simulator

User aircraft (its systems through the library's aircraft profiles):

- get_aircraft_systems: Power, radios, engines, lights, named doors, transponder, chocks, GPU, cabin signs and pushback state, through the aircraft's systems profile
- set_aircraft_control: Open or close a door, set chocks, GPU, parking brake, cabin signs or external power, or call the cabin
- request_ground_service: The sim's jetway, stairs, baggage, catering, ground power, fuel truck or pushback
- set_radio: A COM active or standby frequency, a COM swap, or the squawk
- set_atc_callsign: The call sign the sim's ATC uses (ATC AIRLINE, ATC FLIGHT NUMBER)
- list_addons: The installed MSFS packages, Community, Official and streamed

AI traffic (the library's traffic engine, pkg/traffic/world; adds and removes AI aircraft in the simulator, at airports loaded around the user aircraft):

- list_aircraft_models: Installed aircraft models the engine can spawn, filtered by text
- spawn_departure: Add an AI departure on a stand: stand services, pushback with a tug, taxi, line-up, take-off and SID, on your clearances or the engine's
- spawn_arrival: Add an AI arrival on its STAR: sequenced, approach, landing, vacating and taxi to a stand; optionally a turnaround
- list_our_traffic: Our AI aircraft with state, position, frequency, ground vehicles and the clearances they take now
- atc_clearance: Clear one of ours: pushback, taxi, upto, cross, lineup, takeoff, hold, abort, goaround, standto, follow, manual, remove and more
- get_traffic_picture: Every aircraft around the user aircraft or an airport, with phase and airport; the user's and ours marked
- generate_schedule: Realistic airline schedule for airports (call signs, types, routes, STD/STA); nothing is spawned

Airborne ATC (the engine's tower, landing sequences, separation and radio):

- get_landing_sequence: Each runway end's landing sequence: number, spacing and why, distance to go, delay
- approach_instruction: An approach instruction to one of our arrivals: up, down, slow, speed, hold, release, direct, joinfinal, holdat or goaround
- get_atc_log: The radio: the engine's latest transmissions by position and frequency
- get_conflicts: Closest pairs, losses of separation, predicted conflicts and the resolutions given to ours
- separation_minima: Wake categories, spacing on final, departure interval and runway occupancy for a pair of aircraft types
- set_player_clearance: Tell the engine what the user's ATC cleared, so our traffic keeps off that runway and fits around the user's landing
- get_traffic_status: The engine's state, settings and last error
- get_traffic_airport_info: An airport as the engine works it: runways in use, ATIS, ILS, weather
- get_tcas: TCAS II for our airborne traffic: traffic and resolution advisories now and the latest events

Scheduled and real traffic (adds and removes AI aircraft in the simulator):

- start_schedule: Run scheduled traffic at airports: an airline timetable and light aircraft, by themselves under the engine's ATC
- stop_schedule: Stop the schedule; its aircraft finish their flights, or are removed at once
- get_schedule: The departure and arrival boards with status, stand, runway and estimates
- add_flights: Add flights to the schedule at chosen times
- set_real_traffic: Fly real-world traffic at an airport instead of the timetable
- observe_traffic: Feed real-world sightings (ADS-B) to the engine, or drop them
- set_traffic_corridor: Keep airliners around the user's flight in cruise

## MCP Tools — Both Mode (MCP_MODE=both)

Docs-mode tools always; on Windows, also the SimConnect-mode tools (74 in total) when the simulator is reachable at startup.

## Documentation

${docList || '(documentation coming soon)'}
`;

    return new Response(content, {
        headers: { 'Content-Type': 'text/plain; charset=utf-8' }
    });
};
