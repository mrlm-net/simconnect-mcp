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

github.com/mrlm-net/simconnect Go library guides (v0.15.0):

- list_library_guides: List the library guides and their chapters, optionally filtered by section
- get_library_guide: Read a library guide, or a single chapter of it, as Markdown
- search_library_docs: Search the library guides chapter by chapter

## MCP Tools — SimConnect Mode (MCP_MODE=simconnect, Windows only, 19 tools)

- get_simvar_value: Read a single live simulation variable from the running simulator
- get_simvar_values: Read up to 20 simulation variables in a single call
- set_simvar_value: Write a numeric simulation variable to the user aircraft
- transmit_event: Transmit a named SimConnect client event to the simulator
- get_sim_state: Return a snapshot of current simulator connection state and flight status
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

## MCP Tools — Both Mode (MCP_MODE=both)

Docs-mode tools always; on Windows, also the SimConnect-mode tools (34 in total) when the simulator is reachable at startup.

## Documentation

${docList || '(documentation coming soon)'}
`;

    return new Response(content, {
        headers: { 'Content-Type': 'text/plain; charset=utf-8' }
    });
};
