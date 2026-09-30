---
title: "MCP Tools — SimConnect Mode"
description: Reference for the 41 live-data, AI traffic and airborne ATC MCP tools in SimConnect mode (MCP_MODE=simconnect, Windows only).
order: 2
section: reference
---

All 41 MCP tools listed here are available when the server runs with `MCP_MODE=simconnect` (and, on Windows, with `MCP_MODE=both`, alongside the 15 docs tools — 56 in all). This mode provides live simulator data via the SimConnect SDK, and AI traffic under our control.

**Both mode**: with `MCP_MODE=both` on Windows, the server registers these 41 tools alongside the 15 [docs-mode tools](/docs/mcp-tools-docs) — 56 tools in total — provided SimConnect opens at startup (10-second timeout). If the simulator cannot be reached, or on non-Windows platforms, both mode serves the 15 docs tools only; `simconnect_ready` in the `/health` response reports which case applies.

**Requirements**: Windows only. Microsoft Flight Simulator 2020 or 2024 must be running with SimConnect enabled before issuing any read or transmit calls. The `get_sim_state` tool is safe to call at any time regardless of connection state.

Tools are called over the Model Context Protocol using JSON-RPC 2.0 with the `tools/call` method. Error responses are returned as text content (not JSON-RPC errors) with a prefix token followed by a colon and a human-readable message.

## Tool Overview

| Tool | Description |
|------|-------------|
| [`get_simvar_value`](#get_simvar_value) | Read a single simulation variable value from the running simulator |
| [`get_simvar_values`](#get_simvar_values) | Read up to 20 simulation variables in one call |
| [`set_simvar_value`](#set_simvar_value) | Write a numeric simulation variable to the user aircraft |
| [`transmit_event`](#transmit_event) | Transmit a named SimConnect client event to the simulator |
| [`get_sim_state`](#get_sim_state) | Return a snapshot of current simulator connection state and flight status |
| [`get_nearby_traffic`](#get_nearby_traffic) | List AI and player aircraft within a radius of the user aircraft |
| [`get_traffic_with_phase`](#get_traffic_with_phase) | Like `get_nearby_traffic` with enriched telemetry and inferred flight phase |
| [`get_airports_in_range`](#get_airports_in_range) | List airports in the simulator's loaded scenery area sorted by distance |
| [`get_nearest_airport`](#get_nearest_airport) | Return the single closest airport to the player aircraft |
| [`get_airport_details`](#get_airport_details) | Return detailed facility data for a specific airport by ICAO code |
| [`get_vors_in_range`](#get_vors_in_range) | List VOR navigation stations sorted by distance from the player aircraft |
| [`get_vor_details`](#get_vor_details) | Return detailed data for a specific VOR by ICAO code |
| [`get_ndbs_in_range`](#get_ndbs_in_range) | List NDB navigation stations sorted by distance from the player aircraft |
| [`get_ndb_details`](#get_ndb_details) | Return detailed data for a specific NDB by ICAO code |
| [`get_waypoints_in_range`](#get_waypoints_in_range) | List waypoints sorted by distance from the player aircraft |
| [`get_waypoint_details`](#get_waypoint_details) | Return detailed data for a specific waypoint by ICAO code |
| [`get_airport_taxiways`](#get_airport_taxiways) | Return the taxiway network graph for a specific airport by ICAO code |
| [`get_taxiway_names`](#get_taxiway_names) | Return only the taxiway letter/name strings for an airport (lightweight alternative) |
| [`get_airport_parkings`](#get_airport_parkings) | Return all parking stands, gates, and ramps at a specific airport by ICAO code |
| [`get_airport_procedures`](#get_airport_procedures) | List an airport's SIDs, STARs and approaches, or resolve one into its points |
| [`plan_taxi_route`](#plan_taxi_route) | Plan an ATC-style taxi route from a stand to a runway holding point, or from a runway exit to a stand |
| [`get_runway_entries_exits`](#get_runway_entries_exits) | List the taxiways onto a runway end and the exits from it |
| [`find_stands`](#find_stands) | Find parking stands that fit an aircraft, by wing span, airline and gate |
| [`get_weather`](#get_weather) | Return the weather at the user aircraft |
| [`get_active_runway`](#get_active_runway) | Work out the departure and arrival runways in use and the expected approach |
| [`get_atis`](#get_atis) | Compose an airport's ATIS broadcast, as text and as spoken |
| [`get_fix`](#get_fix) | Look up a waypoint, VOR or NDB with the airways through it |
| [`find_airway_route`](#find_airway_route) | Find the airway route between two enroute fixes |
| [`plan_flight`](#plan_flight) | Plan an IFR flight between two airports: runways, SID, airways, STAR, approach, profile, time and fuel |
| [`list_aircraft_models`](#list_aircraft_models) | List the installed aircraft titles (and liveries) AI traffic can use |
| [`spawn_departure`](#spawn_departure) | Add an AI departure of ours on a stand: pushback, taxi, line-up, take-off and SID, on clearance |
| [`spawn_arrival`](#spawn_arrival) | Add an AI arrival of ours on a STAR or final: approach, landing, vacating and taxi to a stand |
| [`list_our_traffic`](#list_our_traffic) | List our AI aircraft: state, position, speed, taxiway, holding point and the clearances they take now |
| [`atc_clearance`](#atc_clearance) | Give one of our AI aircraft a clearance: pushback, taxi, cross, lineup, takeoff, hold, abort, goaround or remove |
| [`get_traffic_picture`](#get_traffic_picture) | Every aircraft around the user aircraft or an airport, with phase and airport; the user's and ours marked |
| [`generate_schedule`](#generate_schedule) | Generate a realistic airline schedule for airports (pure computation) |

---

## get_simvar_value

Read a single live simulation variable from the running simulator.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | SimVar name, e.g. `"PLANE ALTITUDE"` |
| `unit` | string | Yes | — | SimConnect unit string, e.g. `"feet"` |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | SimVar name as supplied by the caller |
| `value` | number | Current value of the simulation variable |
| `unit` | string | Unit string as supplied by the caller |
| `sim_time` | number | Simulator absolute time in seconds at the moment of the read |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_simvar_value",
    "arguments": {
      "name": "PLANE ALTITUDE",
      "unit": "feet"
    }
  }
}
```

**Example response (connected)**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"name\":\"PLANE ALTITUDE\",\"value\":5280.0,\"unit\":\"feet\",\"sim_time\":3742.5}"
      }
    ]
  }
}
```

**Example response (disconnected)**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "BRIDGE_DISCONNECTED: no active simulator connection"
      }
    ],
    "isError": true
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: The SimConnect bridge is not connected to the simulator. Start the simulator and ensure SimConnect is enabled.
- `INVALID_ARGUMENT`: `name` or `unit` was not provided or is empty.
- `UNKNOWN_VARIABLE`: The variable name or unit is not recognised by SimConnect. Check the name against the SimConnect SDK reference.
- `INTERNAL_ERROR`: Unexpected bridge or SimConnect failure.

---

## get_simvar_values

Read up to 20 live simulation variables in a single call. Per-item failures are embedded in the response array and do not abort the batch.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `vars` | array | Yes | — | List of SimVar name/unit pairs to read. Maximum 20 items. Each item must be an object with `name` (string) and `unit` (string) fields. |

**Returns**

An array of result objects. Each object has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | SimVar name as supplied in the request |
| `value` | number | Current value of the simulation variable; omitted when `error` is present |
| `unit` | string | Unit string as supplied in the request |
| `sim_time` | number | Simulator absolute time in seconds at the moment of the read; omitted when `error` is present |
| `error` | string | Per-item error message; present only when this specific variable could not be read |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_simvar_values",
    "arguments": {
      "vars": [
        { "name": "PLANE ALTITUDE", "unit": "feet" },
        { "name": "AIRSPEED INDICATED", "unit": "knots" }
      ]
    }
  }
}
```

**Example response (connected)**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\"name\":\"PLANE ALTITUDE\",\"value\":5280.0,\"unit\":\"feet\",\"sim_time\":3742.5},{\"name\":\"AIRSPEED INDICATED\",\"value\":142.3,\"unit\":\"knots\",\"sim_time\":3742.5}]"
      }
    ]
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: The SimConnect bridge is not connected to the simulator.
- `INVALID_ARGUMENT`: `vars` was not provided, is empty, exceeds 20 items, or contains an item missing `name` or `unit`.
- `UNKNOWN_VARIABLE`: One or more variable names or units in the batch are not recognised by SimConnect.
- `INTERNAL_ERROR`: Unexpected bridge or SimConnect failure.

---

## transmit_event

Transmit a named SimConnect client event to the running simulator.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | SimConnect event name, e.g. `"LANDING_LIGHTS_TOGGLE"` |
| `value` | number | No | `0` | Optional uint32 event parameter. Must be an integer in the range `0–4294967295`. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Always `true` on success |
| `event` | string | Event name as supplied by the caller |
| `value` | number | The uint32 value transmitted with the event |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "transmit_event",
    "arguments": {
      "name": "LANDING_LIGHTS_TOGGLE"
    }
  }
}
```

**Example response (connected)**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"event\":\"LANDING_LIGHTS_TOGGLE\",\"success\":true,\"value\":0}"
      }
    ]
  }
}
```

**Example response (disconnected)**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "BRIDGE_DISCONNECTED: no active simulator connection"
      }
    ],
    "isError": true
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: The SimConnect bridge is not connected to the simulator.
- `INVALID_ARGUMENT`: `name` was not provided or is empty, or `value` is not an integer in the range `[0, 4294967295]`.
- `INTERNAL_ERROR`: Unexpected bridge or SimConnect failure.

---

## get_sim_state

Return a snapshot of current simulator connection state and flight status. This tool never returns an error — it returns `{"connected": false}` when the bridge is not connected, allowing clients to safely poll at any connection state.

**Requirements**: Windows. The simulator does not need to be running.

**Parameters**: None.

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `connected` | boolean | `true` when the bridge has an active SimConnect session |
| `paused` | boolean | `true` when the simulator is paused; only present when `connected` is `true` |
| `current_flight` | string | Path or name of the currently loaded flight file; only present when `connected` is `true` |
| `sim_time` | number | Simulator absolute time in seconds; only present when `connected` is `true` |
| `simulator_version` | string | Simulator version string reported by SimConnect; only present when `connected` is `true` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "get_sim_state",
    "arguments": {}
  }
}
```

**Example response (connected)**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"connected\":true,\"current_flight\":\"LAST.FLT\",\"paused\":false,\"sim_time\":3742.5,\"simulator_version\":\"11.0.282174.0\"}"
      }
    ]
  }
}
```

**Example response (disconnected)**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"connected\":false}"
      }
    ],
    "isError": false
  }
}
```

**Error codes**: None. This tool always succeeds.

---

## set_simvar_value

Write a numeric simulation variable to the user aircraft. Only writable SimVars will take effect. Use `get_simvar` (docs mode) to check whether a variable is settable before calling this tool.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | SimVar name, e.g. `"AUTOPILOT ALTITUDE LOCK VAR"` |
| `unit` | string | Yes | — | SimConnect unit string, e.g. `"feet"` |
| `value` | number | Yes | — | Numeric value to write |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Always `true` on success |
| `name` | string | SimVar name as supplied by the caller |
| `unit` | string | Unit string as supplied by the caller |
| `value` | number | The value written to the simulator |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "set_simvar_value",
    "arguments": {
      "name": "AUTOPILOT ALTITUDE LOCK VAR",
      "unit": "feet",
      "value": 10000
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"name\":\"AUTOPILOT ALTITUDE LOCK VAR\",\"success\":true,\"unit\":\"feet\",\"value\":10000}"
      }
    ]
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: The SimConnect bridge is not connected to the simulator.
- `INVALID_ARGUMENT`: `name`, `unit`, or `value` was not provided or has the wrong type.
- `INTERNAL_ERROR`: Unexpected bridge or SimConnect failure.

---

## get_nearby_traffic

Return a list of AI and player aircraft within a given radius of the user aircraft. The player aircraft is always included. Returns position, heading, speed, and on-ground status for each aircraft.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `radius_meters` | number | No | `25000` | Search radius in metres. Maximum `200000`. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `sim_time` | number | Simulator Zulu time in seconds since midnight at the moment of the scan |
| `radius_meters` | number | The radius used for the scan |
| `count` | number | Number of aircraft returned |
| `traffic` | array | Array of traffic entries (see below) |
| `error` | string | Present only when the bridge is not connected or the scan failed |

Each entry in `traffic`:

| Field | Type | Description |
|-------|------|-------------|
| `object_id` | number | SimConnect object ID |
| `title` | string | Aircraft title (livery name) |
| `atc_id` | string | ATC callsign / tail number |
| `atc_airline` | string | ATC airline name |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_ft` | number | Altitude in feet MSL |
| `true_heading_deg` | number | Magnetic heading in degrees |
| `ground_speed_kts` | number | Ground speed in knots |
| `on_ground` | boolean | `true` when the aircraft is on the ground |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "get_nearby_traffic",
    "arguments": { "radius_meters": 50000 }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"count\":2,\"radius_meters\":50000,\"traffic\":[{\"object_id\":1,\"title\":\"Airbus A320 Neo\",\"atc_id\":\"TAP123\",\"atc_airline\":\"TAP Air Portugal\",\"latitude\":38.77,\"longitude\":-9.13,\"altitude_ft\":8200,\"true_heading_deg\":274.0,\"ground_speed_kts\":210.5,\"on_ground\":false}]}"
      }
    ]
  }
}
```

**Error codes**: None. Errors (not connected, scan failure) are embedded as an `error` field in the response body rather than returned as MCP errors.

---

## get_traffic_with_phase

Return nearby aircraft with enriched telemetry: vertical speed, actual ground track, inferred flight phase, parking state, runway occupancy, and aircraft category. Uses a single SimConnect round-trip with the same latency as `get_nearby_traffic`.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `radius_meters` | number | No | `25000` | Search radius in metres. Maximum `200000`. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `sim_time` | number | Simulator Zulu time in seconds since midnight at the moment of the scan |
| `radius_meters` | number | The radius used for the scan |
| `count` | number | Number of aircraft returned |
| `traffic` | array | Array of enriched traffic entries (see below) |
| `error` | string | Present only when the bridge is not connected or the scan failed |

Each entry in `traffic` (all fields from `get_nearby_traffic` plus):

| Field | Type | Description |
|-------|------|-------------|
| `category` | string | SimConnect aircraft category (e.g. `"Jet"`, `"Prop"`, `"HelicopterAircraft"`) |
| `track_deg` | number | Actual ground track derived from velocity vectors, in degrees |
| `vertical_speed_fpm` | number | Vertical speed in feet per minute (positive = climbing) |
| `in_parking_state` | boolean | `true` when the aircraft is in a parking / pushback state |
| `on_any_runway` | boolean | `true` when the aircraft is occupying a runway |
| `flight_phase` | string | Inferred phase: `PARKED`, `TAXI`, `CLIMB`, `CLIMB SHALLOW`, `LEVEL`, `DESCENT`, `APPROACH`, or `FINAL` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "get_traffic_with_phase",
    "arguments": {}
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"count\":1,\"radius_meters\":25000,\"traffic\":[{\"object_id\":1,\"title\":\"Cessna 172\",\"atc_id\":\"N12345\",\"atc_airline\":\"\",\"category\":\"Prop\",\"latitude\":38.77,\"longitude\":-9.13,\"altitude_ft\":4500,\"true_heading_deg\":090.0,\"track_deg\":088.5,\"ground_speed_kts\":110.0,\"vertical_speed_fpm\":-200,\"on_ground\":false,\"in_parking_state\":false,\"on_any_runway\":false,\"flight_phase\":\"DESCENT\"}]}"
      }
    ]
  }
}
```

**Error codes**: None. Errors are embedded in the response body.

---

## get_airports_in_range

Return a list of airports in the simulator's loaded scenery area (reality bubble), sorted by distance from the player aircraft. By default only standard ICAO-format airports are returned.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `radius_km` | number | No | `50` | Maximum distance from the player aircraft in kilometres. Maximum `500`. |
| `expanded` | boolean | No | `false` | When `true`, include non-standard identifiers (private fields, military strips, simulator-only codes). |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `radius_km` | number | The radius used for filtering |
| `expanded` | boolean | Whether non-standard airports were included |
| `count` | number | Number of airports returned |
| `airports` | array | Array of airport entries (see below) |
| `error` | string | Present only when not connected or the scan failed |

Each entry in `airports`:

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO airport code |
| `region` | string | ICAO region prefix |
| `latitude` | number | Airport reference latitude in decimal degrees |
| `longitude` | number | Airport reference longitude in decimal degrees |
| `altitude_m` | number | Airport elevation in metres MSL |
| `distance_km` | number | Haversine distance from the player aircraft in kilometres |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "tools/call",
  "params": {
    "name": "get_airports_in_range",
    "arguments": { "radius_km": 100 }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"airports\":[{\"icao\":\"LPMA\",\"region\":\"LP\",\"latitude\":32.697,\"longitude\":-16.778,\"altitude_m\":58.0,\"distance_km\":0.3},{\"icao\":\"LPCC\",\"region\":\"LP\",\"latitude\":32.699,\"longitude\":-16.645,\"altitude_m\":98.0,\"distance_km\":14.9}],\"count\":2,\"expanded\":false,\"radius_km\":100}"
      }
    ]
  }
}
```

**Error codes**: None. Errors are embedded in the response body.

---

## get_nearest_airport

Return the single closest airport to the player aircraft. Note: SimConnect's facility list APIs may exclude the airport the player is currently on; use `get_airports_in_range` with `expanded=true` if you need to detect the current airport.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**: None.

**Returns**

An airport entry object with the same fields as entries returned by `get_airports_in_range`, or an object with an `error` field when not connected, no airports are found, or the scan fails.

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO airport code |
| `region` | string | ICAO region prefix |
| `latitude` | number | Airport reference latitude in decimal degrees |
| `longitude` | number | Airport reference longitude in decimal degrees |
| `altitude_m` | number | Airport elevation in metres MSL |
| `distance_km` | number | Distance from the player aircraft in kilometres |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "method": "tools/call",
  "params": {
    "name": "get_nearest_airport",
    "arguments": {}
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"icao\":\"LPCC\",\"region\":\"LP\",\"latitude\":32.699,\"longitude\":-16.645,\"altitude_m\":98.0,\"distance_km\":14.9}"
      }
    ]
  }
}
```

**Error codes**: None. Errors are embedded in the response body.

---

## get_airport_details

Return detailed facility data for a specific airport by ICAO code. The default response includes name, coordinates, elevation, magnetic variation, closed status, runways, and ATC frequencies. Set `expanded=true` to also receive parking stands, helipads, instrument approaches, SIDs, and STARs.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO airport code, e.g. `"LPMA"` or `"EDDM"`. |
| `region` | string | No | `""` | ICAO region code, e.g. `"LP"`. Leave empty for best results — SimConnect's region filter is strict. |
| `expanded` | boolean | No | `false` | When `true`, also return stands, helipads, approaches, SIDs, and STARs. |

**Returns (default)**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO code |
| `region` | string | ICAO region prefix |
| `name` | string | Short airport name |
| `name64` | string | Full airport name (up to 64 characters) |
| `latitude` | number | Reference latitude in decimal degrees |
| `longitude` | number | Reference longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `magvar_deg` | number | Magnetic variation in degrees (positive = east) |
| `is_closed` | boolean | `true` when the airport is marked closed |
| `runway_count` | number | Number of runways |
| `runways` | array | Runway entries: `name`, `heading_deg`, `length_m`, `width_m`, `surface` |
| `frequencies` | array | ATC frequency entries: `type`, `freq_mhz`, `name` |

**Additional fields when `expanded=true`**

| Field | Type | Description |
|-------|------|-------------|
| `stand_count` | number | Number of parking stands |
| `stands` | array | Stand entries: `number`, `type`, `heading_deg` |
| `helipad_count` | number | Number of helipads |
| `helipads` | array | Helipad entries: `latitude`, `longitude`, `altitude_m`, `heading_deg`, `length_m`, `width_m`, `surface`, `type` |
| `approach_count` | number | Number of instrument approaches |
| `approaches` | array | Approach entries: `type`, `runway`, `has_lnav`, `has_lnavvnav`, `has_lp`, `has_lpv` |
| `departure_count` | number | Number of departure procedures (SIDs) |
| `departures` | array | SID entries: `name`, `runway_transitions`, `enroute_transitions` |
| `arrival_count` | number | Number of arrival procedures (STARs) |
| `arrivals` | array | STAR entries: `name`, `runway_transitions`, `enroute_transitions` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "get_airport_details",
    "arguments": { "icao": "LPMA" }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"icao\":\"LPMA\",\"region\":\"LP\",\"name\":\"Madeira\",\"name64\":\"Cristiano Ronaldo International Airport\",\"latitude\":32.697,\"longitude\":-16.778,\"altitude_m\":58.0,\"magvar_deg\":-5.2,\"is_closed\":false,\"runway_count\":1,\"runways\":[{\"name\":\"05/23\",\"heading_deg\":54.0,\"length_m\":2781,\"width_m\":45,\"surface\":\"Asphalt\"}],\"frequencies\":[{\"type\":\"ATIS\",\"freq_mhz\":126.9,\"name\":\"Madeira ATIS\"}]}"
      }
    ]
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` must be 1–9 uppercase alphanumeric characters; `region` must be 0–4 uppercase alphanumeric characters.
- `AIRPORT_NOT_FOUND`: No airport matching the given ICAO code was found.
- `AIRPORT_DETAILS_ERROR`: SimConnect returned an error while fetching facility data.

---

## get_vors_in_range

Return a list of VOR navigation stations in the simulator's reality bubble, sorted by distance from the player aircraft.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `radius_km` | number | No | `200` | Maximum distance from the player aircraft in kilometres. Maximum `500`. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `radius_km` | number | The radius used for filtering |
| `count` | number | Number of VORs returned |
| `vors` | array | Array of VOR entries (see below) |

Each entry in `vors`:

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO identifier |
| `region` | string | ICAO region prefix |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `frequency_hz` | number | VOR frequency in Hz |
| `magvar_deg` | number | Magnetic variation in degrees |
| `distance_km` | number | Haversine distance from the player aircraft in kilometres |

**Error codes**: None. Errors (not connected) are embedded as an `error` field in the response body.

---

## get_vor_details

Return detailed data for a specific VOR navigation station by ICAO code.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO identifier of the VOR, e.g. `"OPO"`. |
| `region` | string | No | `""` | ICAO region code. Leave empty for best results. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO identifier |
| `region` | string | ICAO region prefix |
| `name` | string | Station name |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `frequency_hz` | number | Frequency in Hz |
| `frequency_mhz` | number | Frequency in MHz |
| `magvar_deg` | number | Magnetic variation in degrees |
| `nav_range_nm` | number | Navigational range in nautical miles |
| `is_nav` | boolean | `true` when the station transmits a VOR signal |
| `is_dme` | boolean | `true` when the station has DME |
| `is_tacan` | boolean | `true` when the station is TACAN |
| `has_glide_slope` | boolean | `true` when the station has a glide slope (ILS) |
| `has_back_course` | boolean | `true` when the station has a back course |

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` was not provided or is empty.
- `VOR_NOT_FOUND`: No VOR matching the given ICAO code was found.

---

## get_ndbs_in_range

Return a list of NDB navigation stations in the simulator's reality bubble, sorted by distance from the player aircraft.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `radius_km` | number | No | `200` | Maximum distance from the player aircraft in kilometres. Maximum `500`. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `radius_km` | number | The radius used for filtering |
| `count` | number | Number of NDBs returned |
| `ndbs` | array | Array of NDB entries (see below) |

Each entry in `ndbs`:

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO identifier |
| `region` | string | ICAO region prefix |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `frequency_hz` | number | NDB frequency in Hz |
| `magvar_deg` | number | Magnetic variation in degrees |
| `distance_km` | number | Haversine distance from the player aircraft in kilometres |

**Error codes**: None. Errors are embedded as an `error` field in the response body.

---

## get_ndb_details

Return detailed data for a specific NDB navigation station by ICAO code.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO identifier of the NDB, e.g. `"LIS"`. |
| `region` | string | No | `""` | ICAO region code. Leave empty for best results. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO identifier |
| `region` | string | ICAO region prefix |
| `name` | string | Station name |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `frequency_hz` | number | Frequency in Hz |
| `frequency_khz` | number | Frequency in kHz |
| `type` | number | NDB type code from SimConnect |
| `range_nm` | number | Navigational range in nautical miles |
| `magvar_deg` | number | Magnetic variation in degrees |
| `is_terminal` | boolean | `true` when this is a terminal NDB |

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` was not provided or is empty.
- `NDB_NOT_FOUND`: No NDB matching the given ICAO code was found.

---

## get_waypoints_in_range

Return a list of waypoints in the simulator's reality bubble, sorted by distance from the player aircraft.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `radius_km` | number | No | `100` | Maximum distance from the player aircraft in kilometres. Maximum `500`. |
| `limit` | number | No | `200` | Maximum number of waypoints to return. Maximum `1000`. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `radius_km` | number | The radius used for filtering |
| `limit` | number | The count cap applied |
| `count` | number | Number of waypoints returned |
| `waypoints` | array | Array of waypoint entries (see below) |

Each entry in `waypoints`:

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO identifier |
| `region` | string | ICAO region prefix |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `magvar_deg` | number | Magnetic variation in degrees |
| `distance_km` | number | Haversine distance from the player aircraft in kilometres |

**Error codes**: None. Errors are embedded as an `error` field in the response body.

---

## get_waypoint_details

Return detailed data for a specific waypoint by ICAO code.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO identifier of the waypoint, e.g. `"ABRIX"`. |
| `region` | string | No | `""` | ICAO region code. Leave empty for best results. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | ICAO identifier |
| `region` | string | ICAO region prefix |
| `latitude` | number | Latitude in decimal degrees |
| `longitude` | number | Longitude in decimal degrees |
| `altitude_m` | number | Elevation in metres MSL |
| `type` | number | Waypoint type code from SimConnect |
| `magvar_deg` | number | Magnetic variation in degrees |
| `n_routes` | number | Number of airways routes passing through this waypoint |
| `is_terminal` | boolean | `true` when this is a terminal waypoint |

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` must be 1–9 uppercase alphanumeric characters; `region` must be 0–4 uppercase alphanumeric characters.
- `WAYPOINT_NOT_FOUND`: No waypoint matching the given ICAO code was found.

---

## get_airport_taxiways

Return the taxiway network graph for a specific airport by ICAO code. The response contains three correlated arrays: `names` (taxiway letter strings), `paths` (directed edges referencing start/end node indices and a name index), and `points` (graph nodes including hold-short positions). Leave `region` empty for best results.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO airport code, e.g. `"EDDM"` or `"KLAX"`. Must be 1–9 uppercase alphanumeric characters. |
| `region` | string | No | `""` | ICAO region code, e.g. `"ED"` or `"K6"`. Leave empty for best results. |
| `max_paths` | number | No | `500` | Maximum number of path entries to return. Range `1–2000`. Large airports (EDDM, KJFK) can exceed 1000 paths; use this to keep the response within MCP client size limits. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `name_count` | number | Total number of taxiway names |
| `point_count` | number | Total number of graph nodes |
| `path_count` | number | Total number of paths before truncation |
| `names` | array | List of taxiway name strings (e.g. `["A", "B", "C"]`) |
| `paths` | array | Directed edges, capped at `max_paths` (see below) |
| `points` | array | Graph nodes (see below) |
| `truncated` | boolean | Present and `true` when paths were capped by `max_paths` |
| `truncated_to` | number | Present when `truncated=true`; the number of paths actually returned |

Each entry in `paths`:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Path type (e.g. `"TAXIWAY"`, `"RUNWAY"`, `"VEHICLE"`) |
| `width_m` | number | Path width in metres |
| `left_half_width_m` | number | Left half-width in metres |
| `right_half_width_m` | number | Right half-width in metres |
| `weight` | number | Path weight / priority value |
| `runway_number` | number | Associated runway number (0 if not runway-related) |
| `runway_designator` | number | Associated runway designator code |
| `left_edge` | string | Left edge lighting type |
| `right_edge` | string | Right edge lighting type |
| `center_line` | string | Center line type |
| `start_node` | number | Index into `points` array for the path start |
| `end_node` | number | Index into `points` array for the path end |
| `name_index` | number | Index into `names` array identifying the taxiway letter |

Each entry in `points`:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Node type (e.g. `"NORMAL"`, `"HOLD_SHORT"`, `"ILS_HOLD_SHORT"`) |
| `orientation` | string | Orientation descriptor |
| `bias_x_m` | number | X offset from the airport reference point in metres |
| `bias_z_m` | number | Z offset from the airport reference point in metres |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 17,
  "method": "tools/call",
  "params": {
    "name": "get_airport_taxiways",
    "arguments": { "icao": "EDDM" }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 17,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"names\":[\"A\",\"B\",\"N\"],\"paths\":[{\"type\":\"TAXIWAY\",\"width_m\":23.0,\"start_node\":0,\"end_node\":1,\"name_index\":0,...}],\"points\":[{\"type\":\"NORMAL\",\"orientation\":\"NONE\",\"bias_x_m\":-823.5,\"bias_z_m\":441.2}]}"
      }
    ]
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` must be 1–9 uppercase alphanumeric characters; `region` must be 0–4 uppercase alphanumeric characters.
- `TAXIWAY_NOT_FOUND`: No taxiway data was found for the given ICAO code.
- `TAXIWAY_ERROR`: SimConnect returned an error while fetching taxiway data.

---

## get_taxiway_names

Return only the taxiway letter/name strings for an airport by ICAO code. Lightweight alternative to `get_airport_taxiways` when only the taxiway label list is needed — no paths or points are returned.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO airport code, e.g. `"EDDM"` or `"KLAX"`. Must be 1–9 uppercase alphanumeric characters. |
| `region` | string | No | `""` | ICAO region code, e.g. `"ED"` or `"K6"`. Leave empty for best results. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | Airport ICAO code |
| `name_count` | number | Number of taxiway names |
| `names` | array | List of taxiway name strings (e.g. `["A", "B", "C", "N"]`) |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 19,
  "method": "tools/call",
  "params": {
    "name": "get_taxiway_names",
    "arguments": { "icao": "EDDM" }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 19,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"icao\":\"EDDM\",\"name_count\":12,\"names\":[\"A\",\"B\",\"C\",\"D\",\"E\",\"F\",\"G\",\"H\",\"J\",\"K\",\"M\",\"N\"]}"
      }
    ]
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` must be 1–9 uppercase alphanumeric characters; `region` must be 0–4 uppercase alphanumeric characters.
- `TAXIWAY_NOT_FOUND`: No taxiway data was found for the given ICAO code.
- `TAXIWAY_ERROR`: SimConnect returned an error while fetching taxiway data.

---

## get_airport_parkings

Return all parking stands, gates, and ramps at a specific airport by ICAO code. Each entry includes type, name, suffix, number, heading, radius, and position offsets from the airport reference point. Returns the full `TAXI_PARKING` record — more fields than the abbreviated stands array in `get_airport_details`. Leave `region` empty for best results.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | ICAO airport code, e.g. `"EDDM"` or `"KLAX"`. Must be 1–9 uppercase alphanumeric characters. |
| `region` | string | No | `""` | ICAO region code, e.g. `"ED"` or `"K6"`. Leave empty for best results. |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `count` | number | Number of parking stands returned |
| `parkings` | array | Array of parking entries (see below) |

Each entry in `parkings`:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Parking type (e.g. `"GATE"`, `"RAMP_GA"`, `"RAMP_CARGO"`, `"FUEL"`, `"VEHICLE"`) |
| `name` | string | Parking name label (derived from type) |
| `suffix` | string | Parking suffix letter (e.g. `"A"`, `"L"`, `"R"`) |
| `number` | number | Parking stand number |
| `orientation` | string | Push-back orientation descriptor |
| `heading_deg` | number | Parking heading in degrees true |
| `radius_m` | number | Parking radius in metres (aircraft size constraint) |
| `bias_x_m` | number | X offset from the airport reference point in metres |
| `bias_z_m` | number | Z offset from the airport reference point in metres |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 18,
  "method": "tools/call",
  "params": {
    "name": "get_airport_parkings",
    "arguments": { "icao": "EDDM" }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 18,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"count\":2,\"parkings\":[{\"type\":\"GATE\",\"name\":\"Gate\",\"suffix\":\"A\",\"number\":1,\"orientation\":\"NONE\",\"heading_deg\":180.0,\"radius_m\":40.0,\"bias_x_m\":120.5,\"bias_z_m\":-310.2},{\"type\":\"RAMP_GA\",\"name\":\"Ramp GA\",\"suffix\":\"\",\"number\":2,\"orientation\":\"NONE\",\"heading_deg\":90.0,\"radius_m\":12.0,\"bias_x_m\":-50.0,\"bias_z_m\":80.0}]}"
      }
    ]
  }
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INVALID_ARGUMENT`: `icao` must be 1–9 uppercase alphanumeric characters; `region` must be 0–4 uppercase alphanumeric characters.
- `PARKING_NOT_FOUND`: No parking data was found for the given ICAO code.
- `PARKING_ERROR`: SimConnect returned an error while fetching parking data.

---

## Airport, weather and navigation tools

The ten tools below are built on the [mrlm-net/simconnect](https://github.com/mrlm-net/simconnect) Go library (`pkg/airport` and `pkg/nav`), which the server runs on the bridge's SimConnect connection. They are registered only with the real SimConnect bridge — a server started with the mock bridge does not list them.

Shared behaviour:

- **Airport data** (layouts, taxi network, stands, procedures) is loaded from the simulator on first use and cached. SimConnect does not answer for an unknown airport, so an unknown ICAO code ends in `NOT_FOUND` after about 20 s (procedures: about 30 s).
- **Weather** comes from the simulator's ambient weather **at the user aircraft** only. SimConnect has no gust, ceiling or dewpoint variables. `get_active_runway`, `get_atis` and `plan_flight` (departure runway) use it, so they are right for the airport the aircraft is at or near.
- **Airways** are crawled from the simulator's navdata on demand around the route. The first crawl of an area takes a few seconds and is cached; the crawl radius is capped at 400 NM from the route midpoint, beyond which the route is flown direct.
- ICAO airport codes are 3–8 letters or digits (case-insensitive).

Examples show the tool result's `text` content, formatted; the JSON-RPC envelope is the same as for the tools above. Long arrays are abridged (`…`).

**Error codes** used by these tools:

- `INVALID_ARGUMENT`: A parameter is missing or malformed, or names something the airport does not have (procedure, runway, stand).
- `NOT_FOUND`: The airport or fix is not in the simulator's data.
- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `TIMEOUT`: The simulator did not answer in time.
- `SIM_ERROR`: Any other simulator or loader failure.
- `NO_ROUTE`: No taxi route or airway route between the given points.
- `PLAN_ERROR`: `plan_flight` could not build the plan or its `.pln` file.

---

## Airport procedures & ground

## get_airport_procedures

List an airport's SIDs, STARs and instrument approaches from the simulator's navdata, or resolve one into the points it flies. Without `name`: every procedure (optionally only those of `runway`) with its runways and transitions. With `name`: the SID, STAR or approach as points in order — ident, position, ARINC 424 leg type, true course, altitude window (ft) and speed limit, IAF/FAF/MAP — for the runway and transition given. Also returns the magnetic variation.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code, e.g. `"LKPR"` |
| `runway` | string | No | — | Runway end, e.g. `"24"` or `"06L"`. Filters the list; required to resolve a SID or STAR serving several runways. |
| `name` | string | No | — | Procedure to resolve: a SID or STAR name (`"VOZ5M"`) or an approach name (`"ILS 24"`, `"RNAV 06 Z"`) |
| `transition` | string | No | — | Enroute transition of a SID/STAR, or the approach transition (IAF) of an approach |

**Returns (list, without `name`)**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | Airport ICAO code |
| `runway` | string | Runway filter (empty when not given) |
| `magvar` | number | Magnetic variation in degrees |
| `sids`, `stars` | array | `{name, runways, enroute_transitions}` per procedure |
| `approaches` | array | `{name, runway, transitions}` per approach |

**Returns (resolved, with `name`)**

| Field | Type | Description |
|-------|------|-------------|
| `icao`, `name`, `runway`, `transition` | string | As requested |
| `kind` | string | `"SID"`, `"STAR"` or `"APPROACH"` |
| `magvar` | number | Magnetic variation in degrees |
| `points` | array | Points in order (see below) |
| `missed_approach` | array | Approach only: the missed approach points |

Each point: `ident`, `kind`, `lat`, `lon`, `leg` (ARINC 424 leg type, e.g. `"TF"`), `course_true`, `alt_min_ft`, `alt_max_ft`, `speed_max_kts`, and the flags `fly_over`, `iaf`, `faf`, `map`, `vectors` (zero and false values are omitted).

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 20,
  "method": "tools/call",
  "params": {
    "name": "get_airport_procedures",
    "arguments": { "icao": "LKPR", "name": "ILS 24" }
  }
}
```

**Example response (abridged)**

```json
{
  "icao": "LKPR", "kind": "APPROACH", "name": "ILS 24", "runway": "", "transition": "", "magvar": …,
  "points": [
    { "ident": "…", "lat": …, "lon": …, "leg": "IF", "alt_min_ft": …, "iaf": true },
    …,
    { "ident": "RW24", "lat": …, "lon": …, "leg": "…", "course_true": …, "map": true }
  ],
  "missed_approach": [ … ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `icao` is not an airport ICAO code; `name` is not a SID, STAR or approach of the airport; or the SID/STAR serves several runways and `runway` was not given.
- `NOT_FOUND` / `TIMEOUT`: Unknown airport or no procedures loaded (after about 30 s).
- `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## plan_taxi_route

Plan a taxi route on the simulator's taxi network the way ATC would give it: fewer turns, no needless runway crossings, taxiways the aircraft fits. `direction=departure`: from the parking stand to the holding point of the runway (full length, or at `entry`). `direction=arrival`: from a runway exit (`exit`, or the one reached after `rollout_m`) to the stand. Use `get_runway_entries_exits` and `find_stands` for names.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |
| `parking` | string | Yes | — | Stand label, e.g. `"C22"`, `"S22A"` |
| `runway` | string | Yes | — | Runway end, e.g. `"24"` |
| `direction` | string | No | `"departure"` | `"departure"` or `"arrival"` |
| `entry` | string | No | — | Departure: taxiway to enter the runway by (intersection departure), e.g. `"B"` |
| `exit` | string | No | — | Arrival: taxiway to vacate the runway by, e.g. `"D"` |
| `rollout_m` | number | No | `1500` | Arrival without `exit`: landing roll in metres before vacating |
| `via` | string | No | — | Taxiways to follow in order, e.g. `"F, L"` |
| `wingspan_m` | number | No | — | Aircraft wing span in metres; keeps to taxiways it fits (e.g. `35.8` A320, `64.8` 777-300ER) |
| `include_points` | boolean | No | `false` | Include the route's points (lat/lon) |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao`, `direction`, `parking`, `runway` | string | As planned |
| `entry` / `exit` | string | Runway entry (departure) or exit (arrival) used |
| `instruction` | string | ATC-style taxi instruction |
| `length_m` | number | Route length in metres |
| `taxiways` | array | Taxiways in order |
| `runway_crossings` | array | Runways crossed (empty array when none) |
| `hold_short` | string | Holding point runway; `" (ILS hold)"` appended for an ILS holding point |
| `tight` | boolean | Present when the route uses a taxiway tight for the wing span |
| `points` | array | `{lat, lon}` points, with `include_points=true` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 21,
  "method": "tools/call",
  "params": {
    "name": "plan_taxi_route",
    "arguments": { "icao": "LKPR", "parking": "C22", "runway": "24" }
  }
}
```

**Example response** (MSFS 2024)

```json
{
  "icao": "LKPR", "direction": "departure", "parking": "C22", "runway": "24",
  "instruction": "taxi to holding point runway 24 via H1 H A",
  "length_m": 1595, "taxiways": ["H1", "H", "A"], "runway_crossings": [], "hold_short": "06/24"
}
```

**Error codes**

- `INVALID_ARGUMENT`: `icao`, `parking` or `runway` missing; `direction` not `departure`/`arrival`; unknown stand (use `find_stands`).
- `NO_ROUTE`: No route found, or the runway or `exit` does not exist (the message lists the runway's exits).
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## get_runway_entries_exits

List the taxiways onto a runway end for departures (nearest the threshold first, with the runway length remaining ahead) and the exits for landings on it (distance from the threshold, angle, high-speed, side). Names feed `plan_taxi_route`'s `entry` and `exit`.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |
| `runway` | string | Yes | — | Runway end, e.g. `"24"` |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao`, `runway` | string | As requested |
| `entries` | array | `{taxiway, from_threshold_m, remaining_m, angle}`, nearest the threshold first |
| `exits` | array | `{taxiway, from_threshold_m, angle, high_speed, side}`; `side` is `"left"` or `"right"`, `high_speed` present when true |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 22,
  "method": "tools/call",
  "params": {
    "name": "get_runway_entries_exits",
    "arguments": { "icao": "LKPR", "runway": "24" }
  }
}
```

**Example response (abridged)**

```json
{
  "icao": "LKPR", "runway": "24",
  "entries": [
    { "taxiway": "A", "from_threshold_m": …, "remaining_m": …, "angle": … },
    …
  ],
  "exits": [
    { "taxiway": "D", "from_threshold_m": …, "angle": …, "side": "…" },
    …
  ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `icao` or `runway` missing, or the airport has no such runway end.
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## find_stands

Find parking stands at an airport that fit an aircraft: stand label, type, size class (small/medium/heavy), radius, heading, the airlines the scenery assigns, and the stands it overlaps (split or alternate stands). Filter by wing span, airline (stands without airlines serve any) and gates only.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |
| `wingspan_m` | number | No | `0` | Aircraft wing span in metres; stands need a radius of at least half the span + 1 m (`0`: any stand) |
| `airline` | string | No | — | Airline ICAO code the stand must serve, e.g. `"DLH"` |
| `gates_only` | boolean | No | `false` | Only gates (no ramps) |
| `limit` | number | No | `100` | Maximum stands returned, 1–500 |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | Airport ICAO code |
| `total` | number | Stands matching the filters |
| `count` | number | Stands returned (at most `limit`) |
| `stands` | array | `{label, type, size, radius_m, heading_true, airlines, overlaps, lat, lon}`; `type` is the SimConnect parking type (e.g. `"GATE_HEAVY"`, `"RAMP_GA_SMALL"`) |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 23,
  "method": "tools/call",
  "params": {
    "name": "find_stands",
    "arguments": { "icao": "LKPR", "wingspan_m": 64.8 }
  }
}
```

**Example response (abridged)**

```json
{
  "icao": "LKPR", "total": 4, "count": 4,
  "stands": [
    { "label": "…", "type": "…", "size": "heavy", "radius_m": …, "heading_true": …, "overlaps": ["…"], "lat": …, "lon": … },
    …
  ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `icao` is not an airport ICAO code, or `limit` is outside 1–500.
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## Weather & runway in use

## get_weather

Return the weather at the user aircraft: wind (degrees true, knots), visibility, temperature, QNH (hPa and inHg), precipitation, whether the aircraft is in cloud, and icing conditions (visible moisture at or below +10 °C). SimConnect has no gust, ceiling or dewpoint variables.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

None.

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `wind_dir_true` | number | Wind direction, degrees true |
| `wind_kts` | number | Wind speed, knots |
| `calm` | boolean | Wind is calm |
| `visibility_m` | number | Visibility in metres |
| `temp_c` | number | Ambient temperature, °C |
| `qnh_hpa`, `qnh_inhg` | number | Sea-level pressure in hPa and inHg |
| `precip` | string | `"none"`, `"rain"` or `"snow"` (omitted when unknown) |
| `in_cloud` | boolean | The aircraft is in cloud |
| `icing_conditions` | boolean | Visible moisture at or below +10 °C |
| `note` | string | Reminder that the weather is measured at the user aircraft |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 24,
  "method": "tools/call",
  "params": { "name": "get_weather", "arguments": {} }
}
```

**Example response** (illustrative)

```json
{
  "wind_dir_true": 100, "wind_kts": 8, "calm": false, "visibility_m": 20000, "temp_c": 12,
  "qnh_hpa": 1025, "qnh_inhg": 30.27, "precip": "none", "in_cloud": false, "icing_conditions": false,
  "note": "Measured at the user aircraft: the simulator gives the ambient weather there only. Gusts, ceiling and dewpoint are not available from SimConnect."
}
```

**Error codes**

- `TIMEOUT`: No weather from the simulator within 10 s.
- `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## get_active_runway

Work out the runways in use at an airport as a tower would: the preferential runway if the wind allows, else the one with the most headwind, within the airport's tailwind and crosswind limits. Returns the departure and arrival runway, wind components, whether an ILS or visual approach is expected and the best published approach, the transition altitude and level. Uses the weather at the user aircraft, so it is right for the airport the aircraft is at or near.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | Airport ICAO code |
| `departure_runway`, `arrival_runway` | string | Runways in use |
| `headwind_kts`, `crosswind_kts` | number | Mean wind components on the arrival runway (negative headwind: tailwind) |
| `within_wind_limits` | boolean | `false` when no runway meets the limits and the one with the most headwind was taken anyway |
| `approach_kind` | string | `"ILS"` when visibility is below 5000 m, else `"visual/RNAV"` |
| `approach` | string | Best published approach to the arrival runway (omitted without procedures) |
| `transition_altitude_ft` | number | Transition altitude |
| `transition_level` | number | Transition level (flight level) for the current QNH |
| `preferred_runways` | array | The airport's preferential runways, if known |
| `weather` | object | The weather used, as returned by `get_weather` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 25,
  "method": "tools/call",
  "params": {
    "name": "get_active_runway",
    "arguments": { "icao": "LKPR" }
  }
}
```

**Example response** (MSFS 2024, weather abridged)

```json
{
  "icao": "LKPR", "departure_runway": "06", "arrival_runway": "06",
  "headwind_kts": 7, "crosswind_kts": 5, "within_wind_limits": true,
  "approach_kind": "visual/RNAV", "approach": "ILS 06",
  "transition_altitude_ft": 5000, "transition_level": 60, "preferred_runways": ["24", "06"],
  "weather": { "wind_dir_true": 100, "wind_kts": 8, "qnh_hpa": 1025, … }
}
```

**Error codes**

- `INVALID_ARGUMENT`: `icao` is not an airport ICAO code.
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## get_atis

Compose the ATIS broadcast of an airport from the simulator's weather and the runway in use: information letter, time, runways, approach, wind, visibility, temperature, QNH, transition level. Returns the text as written and as spoken (phonetic, for text-to-speech). Uses the weather at the user aircraft.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |
| `letter` | string | No | `"A"` | Information letter A–Z (case-insensitive) |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `icao` | string | Airport ICAO code |
| `letter` | string | Information letter |
| `text` | string | ATIS as written |
| `spoken` | string | ATIS as spoken (phonetic numbers and letters) |
| `note` | string | Reminder that the weather is measured at the user aircraft |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 26,
  "method": "tools/call",
  "params": {
    "name": "get_atis",
    "arguments": { "icao": "LKPR", "letter": "B" }
  }
}
```

**Example response** (MSFS 2024, `spoken` and `note` abridged)

```json
{
  "icao": "LKPR", "letter": "B",
  "text": "Ruzyne information Bravo, time 2330, runway in use 06, wind 100 degrees 8 knots, visibility 10 kilometers or more, temperature 12, QNH 1025, transition level 60, advise on initial contact you have information Bravo.",
  "spoken": "Ruzyne information Bravo, time two three three zero, runway in use zero six, …",
  "note": "Measured at the user aircraft: …"
}
```

**Error codes**

- `INVALID_ARGUMENT`: `icao` is not an airport ICAO code, or `letter` is not a single letter A–Z.
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## Navigation & flight planning

## get_fix

Look up an enroute fix in the simulator's navdata: a waypoint, VOR or NDB with its position, frequency (MHz for a VOR, kHz for an NDB), name, and the airways through it (previous fix, airway, next fix). Identifiers repeat between waypoints, VORs and NDBs; without `kind` all three are tried.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `ident` | string | Yes | — | Fix identifier, e.g. `"VOZ"`, `"GOLOP"` (1–9 letters or digits) |
| `region` | string | No | — | ICAO region, e.g. `"LK"` (recommended: identifiers repeat worldwide) |
| `kind` | string | No | all | `W` (waypoint), `V` (VOR) or `N` (NDB) |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `count` | number | Number of fixes found |
| `fixes` | array | Fixes (see below) |

Each fix: `key` (`IDENT.REGION.KIND`, usable as `find_airway_route`'s `from`/`to`), `ident`, `region`, `kind` (`"waypoint"`, `"VOR"` or `"NDB"`), `name`, `freq`, `lat`, `lon`, `terminal` (present when true), `airways` (`"PREV AIRWAY NEXT"` per airway through the fix).

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 27,
  "method": "tools/call",
  "params": {
    "name": "get_fix",
    "arguments": { "ident": "VOZ", "region": "LK" }
  }
}
```

**Example response** (MSFS 2024)

```json
{
  "count": 1,
  "fixes": [
    {
      "key": "VOZ.LK.V", "ident": "VOZ", "region": "LK", "kind": "VOR", "name": "VOZICE", "freq": 116.95,
      "lat": …, "lon": …,
      "airways": ["M725 TABEM", "T709 USUPA", "Z21 NELPA", "VAKLA Z30"]
    }
  ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `ident` must be 1–9 and `region` 0–4 uppercase letters or digits; `kind` must be `W`, `V` or `N`.
- `NOT_FOUND`: No such fix.
- `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## find_airway_route

Find the airway route between two enroute fixes over the simulator's airway network (A*, with a penalty for every change of airway). The network is crawled from the simulator on demand around the two fixes (a few seconds the first time; cached). When the fixes are not connected, or the airways are over `max_stretch` times the direct distance, the route is direct (`DCT`).

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `from` | string | Yes | — | Start fix as `IDENT`, `IDENT.REGION` or `IDENT.REGION.KIND`, e.g. `"VOZ.LK.V"` (kind defaults to waypoint) |
| `to` | string | Yes | — | End fix, same format |
| `max_stretch` | number | No | `1.5` | Fly direct when the airways are longer than this times the direct distance; `0` = always airways |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `route` | string | ICAO route string, e.g. `"VOZ M725 OKF"` |
| `distance_nm` | number | Route distance, NM |
| `direct_nm` | number | Direct (great-circle) distance, NM |
| `steps` | array | `{airway, fix, lat, lon, distance_nm}` per fix, with the leg distance |
| `graph` | object | `fixes_crawled_within_nm` (crawl radius) and `segments` (airway segments loaded) |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 28,
  "method": "tools/call",
  "params": {
    "name": "find_airway_route",
    "arguments": { "from": "VOZ.LK.V", "to": "OKF.LK.V" }
  }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "route": "VOZ M725 OKF", "distance_nm": 42.8, "direct_nm": …,
  "steps": [
    { "fix": "VOZ.LK.V", "lat": …, "lon": …, "distance_nm": 0 },
    …,
    { "airway": "M725", "fix": "OKF.LK.V", "lat": …, "lon": …, "distance_nm": … }
  ],
  "graph": { "fixes_crawled_within_nm": …, "segments": … }
}
```

**Error codes**

- `INVALID_ARGUMENT`: `from` or `to` is not `IDENT[.REGION[.KIND]]`, or the kind is not `W`, `V` or `N`.
- `NOT_FOUND`: A fix does not exist.
- `NO_ROUTE`: No route between the fixes.
- `TIMEOUT`: The crawl did not finish within 90 s (it completes in the background for the next call).
- `BRIDGE_DISCONNECTED`, `SIM_ERROR`: See above.

---

## plan_flight

Plan an IFR flight between two airports from the simulator's navdata: runways in use (weather at the user aircraft for the departure), SID, airways (crawled on demand; direct where they detour), STAR and best approach, semicircular cruise level, vertical profile with TOC/TOD, distance, time and fuel for the aircraft type. Returns the ICAO route and the waypoints; optionally the MSFS `.pln` file. Takes up to a minute the first time (airport data and the airway crawl).

> **`load_into_sim=true` changes the simulator's flight plan.** The plan is written as a `.pln` file to `%TEMP%\simconnect-mcp` and loaded with `SimConnect_FlightPlanLoad` as the user aircraft's flight plan.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `departure` | string | Yes | — | Departure airport ICAO |
| `arrival` | string | Yes | — | Arrival airport ICAO |
| `aircraft_type` | string | No | A320 | ICAO type designator, e.g. `"A20N"`, `"B738"`, `"B77W"` (planning speeds, fuel flow) |
| `cruise_fl` | number | No | auto | Cruise flight level, e.g. `340` (default: chosen by direction and distance) |
| `departure_runway` | string | No | in use | Departure runway (default: the runway in use for the weather) |
| `arrival_runway` | string | No | auto | Arrival runway (default: in use for calm wind, the longest) |
| `airways` | boolean | No | `true` | Route over airways; `false` plans direct between SID and STAR |
| `alternate_fuel_kg` | number | No | `0` | Alternate fuel to add, kg |
| `include_pln` | boolean | No | `false` | Include the `.pln` file text in the result |
| `load_into_sim` | boolean | No | `false` | Load the plan into the simulator as the user aircraft's flight plan |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `departure`, `arrival` | string | Airport ICAO codes |
| `aircraft_type` | string | Type planned (empty for an unknown type, planned as a generic medium jet) |
| `departure_runway`, `arrival_runway` | string | Runways |
| `sid`, `sid_transition`, `star`, `star_transition` | string | Procedures chosen |
| `approach`, `approach_transition` | string | Approach chosen |
| `route` | string | ICAO route string |
| `cruise_fl` | number | Cruise flight level |
| `magnetic_track` | number | Overall magnetic track (for the semicircular rule) |
| `distance_nm`, `toc_nm`, `tod_nm` | number | Total distance; distance to top of climb and to top of descent |
| `ete_minutes` | number | Estimated time enroute |
| `fuel_kg` | object | `taxiKg`, `tripKg`, `contingencyKg`, `alternateKg`, `reserveKg`, `totalKg` (block fuel) |
| `waypoints` | array | `{ident, via, phase, alt_ft, dist_nm, constraint, lat, lon}`; `phase` is `SID`, `ENROUTE`, `STAR` or `APPROACH` |
| `pln` | string | `.pln` file text, with `include_pln=true` |
| `loaded_into_sim` | boolean | `true` when loaded with `load_into_sim=true` |
| `warnings` | array | Fallbacks taken, e.g. no procedures, no weather, airways unavailable or cut off beyond 400 NM |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 29,
  "method": "tools/call",
  "params": {
    "name": "plan_flight",
    "arguments": { "departure": "LKPR", "arrival": "LOWW", "aircraft_type": "A20N" }
  }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "departure": "LKPR", "arrival": "LOWW", "aircraft_type": "A20N",
  "departure_runway": "06", "arrival_runway": "16",
  "sid": "VOZ5D", "star": "LANU7W", "approach": "ILS 16",
  "route": "VOZ5D VOZ M725 LANUX LANU7W",
  "cruise_fl": 290, "distance_nm": 166, "ete_minutes": 32,
  "fuel_kg": { "totalKg": 2724, … },
  "waypoints": [ { "ident": "VOZ", "phase": "SID", "alt_ft": …, "dist_nm": …, "lat": …, "lon": … }, … ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `departure` or `arrival` is not an airport ICAO code.
- `NOT_FOUND`: Unknown airport (after about 20 s).
- `PLAN_ERROR`: The plan or its `.pln` file could not be built.
- `TIMEOUT`: The plan did not finish within 90 s.
- `BRIDGE_DISCONNECTED`, `SIM_ERROR`: Not connected, or loading the flight plan into the simulator failed.

---

## AI traffic tools

The seven tools below put AI aircraft of our own into the simulator and fly them under ATC-style control: departures push back, taxi, line up, take off and fly the SID; arrivals fly the STAR and approach, land, vacate and taxi to a stand. They are built on the [mrlm-net/simconnect](https://github.com/mrlm-net/simconnect) Go library's `pkg/traffic` and, like the tools above, are registered only with the real SimConnect bridge.

> **These tools change the simulator.** `spawn_departure` and `spawn_arrival` add an aircraft to the sim; `atc_clearance` with `remove` takes it out again. `list_aircraft_models`, `list_our_traffic` and `get_traffic_picture` only read, and `generate_schedule` does not touch the simulator at all.

Shared behaviour:

- **The airport must be loaded around the user aircraft.** Spawns use the airport's taxi graph, stands and procedures from the simulator (see [Airport, weather and navigation tools](#airport-weather-and-navigation-tools)).
- **Motion is injected**: the library moves each aircraft of ours along its taxi route, runway and approach (speed, heading, lights) instead of leaving it to MSFS AI.
- **Chosen when not given**: the runway is the one in use for the weather at the user aircraft (as `get_active_runway`); the SID or STAR is the first one for the runway; the stand is a free one that fits the wing span and the call sign's airline (for arrivals, near the runway), assigned by the library's stand allocator, which also reserves each flight's stand and taxi route; the model is an installed aircraft of `aircraft_type` (default A320) in the livery of the call sign's airline — its first three letters, e.g. `CSA` in `CSA123`. `traffic.ModelsFor` ranks the type in the airline's livery first, then a type of the same size in the airline's livery, then the type in any livery.
- **At most 32 aircraft of ours** at once. Call signs are 2–8 letters or digits and unique among ours.
- **Clearances**: each flight lists `actions`, the `atc_clearance` actions that fit its state now (see [Clearance flow](#clearance-flow)).
- **The traffic picture** comes from a scan of every aircraft within 80 km (about 43 NM) of the user aircraft, repeated every second once `get_traffic_picture` or a spawn has started it.
- **Airborne ATC**: the runtime's tower and landing sequences clear and sequence the flights not held for clearances (see [Airborne ATC tools](#airborne-atc-tools)).

Examples show the tool result's `text` content, formatted, from MSFS 2024 at LKPR. Long arrays are abridged (`…`).

**Error codes** used by these tools:

- `TRAFFIC_ERROR`: The library refused or failed: the call sign is already ours, 32 aircraft already, no free stand that fits, no installed aircraft of the type, no taxi route, or an action that does not fit the flight (the message lists the ones that do).
- `NOT_FOUND`: The call sign is not one of ours (see `list_our_traffic`), or the airport is not in the simulator's data.
- `INVALID_ARGUMENT`: A parameter is missing or malformed, or names something the airport does not have (runway, SID, STAR).
- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `TIMEOUT`: The simulator did not answer in time (spawns: 60 s).

### Clearance flow

`atc_clearance` takes any action of the flight's kind; the flight's `actions` list the ones that fit now. A clearance given early means no stop at that point — `takeoff` while taxiing gives a rolling take-off. `remove` fits every state.

**Departures** (`hold_for_clearances=true`)

| Action | When (state) | Effect |
|--------|--------------|--------|
| `pushback` | `awaiting pushback` | Pushes back off the stand (`pushback`, about 3 kt, nav and beacon lights on), then `awaiting taxi` |
| `taxi` | `awaiting pushback`, `pushback`, `awaiting taxi`, `taxiing`; `holding short` of a runway on the way | Taxis the planned route to the holding point (`taxiing`, up to about 15 kt); after `hold`, taxis on |
| `hold` | `taxiing` | Stops where it is (0 kt) until `taxi` |
| `cross` | `holding short` of a runway on the way | Crosses it and taxis on |
| `lineup` | `holding short` of the departure runway | Lines up and waits (`lining up`, `lined up`) |
| `takeoff` | `pushback`, `awaiting taxi`, `taxiing`, `holding short` of the departure runway, `lining up`, `lined up` | Takes off (`departing`) and flies the SID, then `complete` |
| `abort` | `lining up`, `lined up`, `departing` | Rejects the take-off before V1: stops, vacates and taxis back to the holding point; past V1 it is refused and the take-off continues |
| `remove` | any | Takes the aircraft out of the simulator |

**Arrivals** (`hold_for_clearance=true`)

| Action | When (state) | Effect |
|--------|--------------|--------|
| `goaround` | `approaching`, `landing` | Goes around before touchdown and comes back to the approach; on the runway it is refused |
| `taxi` | `approaching`, `landing`, `rollout`, `vacating`, `awaiting taxi`, `taxiing`, `holding short` | Taxis to the stand once clear of the runway (given early: no stop); after `hold`, taxis on |
| `hold` | `taxiing` | Stops where it is until `taxi` |
| `cross` | `holding short` of a runway on the way | Crosses it and taxis on |
| `remove` | any | Takes the aircraft out of the simulator |

States: departures go `spawning`, `awaiting pushback`, `pushback`, `awaiting taxi`, `taxiing`, `holding short`, `lining up`, `lined up`, `departing`, `complete`; arrivals go `spawning`, `approaching`, `landing`, `rollout`, `vacating`, `awaiting taxi`, `taxiing`, `holding short`, `parking`, `parked`. Either may end `cancelled` or `failed` (see `error`). Without holding for clearances, the ground steps clear themselves after a short, varied wait, and the runway steps (line-up, take-off, crossings) come from the tower when the runway allows. Arrivals are sequenced, lose their delays, and go around by themselves when the runway is not free.

---

## list_aircraft_models

List the aircraft installed in the simulator that AI traffic can use, as `"title"` or `"title|livery"` — the form `spawn_departure` and `spawn_arrival` take in `model`. Without a `model`, the spawn tools pick one of `aircraft_type` in the call sign's airline livery. The simulator enumerates its aircraft once, on the first call.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `filter` | string | No | — | Words that must all appear in the title (case-insensitive), e.g. `"A320 Lufthansa"` |
| `limit` | number | No | `100` | Maximum titles returned, 1–500 |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `total` | number | Titles matching the filter |
| `count` | number | Titles returned (at most `limit`) |
| `models` | array | Titles, sorted; a title with a livery is the title, a vertical bar and the livery |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 30,
  "method": "tools/call",
  "params": {
    "name": "list_aircraft_models",
    "arguments": { "filter": "a320", "limit": 8 }
  }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "total": 517, "count": 8,
  "models": [ "A320neo V2 VIP|Air Busan", … ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `limit` is outside 1–500.
- `TIMEOUT`: The simulator did not list its aircraft within 20 s.
- `BRIDGE_DISCONNECTED`, `TRAFFIC_ERROR`: See above.

---

## spawn_departure

Put an AI departure under our control on a stand at an airport. It pushes back, taxis the planned route to the runway, lines up and takes off, then flies the SID. With `hold_for_clearances` (default `true`) it waits at every step for `atc_clearance` — `pushback`, `taxi`, (`cross`), `lineup`, `takeoff`; otherwise it goes by itself. Stand, runway, SID and model are chosen when not given. Follow it with `list_our_traffic`.

> **Adds an aircraft to the simulator.** Take it out with `atc_clearance` `remove`.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled, and the airport loaded around the user aircraft.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |
| `callsign` | string | Yes | — | Call sign, 2–8 letters or digits, e.g. `"CSA123"`; the first three letters pick the airline |
| `stand` | string | No | free stand that fits | Stand label, e.g. `"C22"` |
| `runway` | string | No | in use | Departure runway |
| `entry` | string | No | full length | Runway entry taxiway for an intersection departure, e.g. `"B"` |
| `sid` | string | No | `"auto"` | SID name, `"auto"` (the first SID for the runway) or `"none"` (climb straight ahead) |
| `model` | string | No | chosen | Aircraft title from `list_aircraft_models`, as listed (with its livery, if any) |
| `aircraft_type` | string | No | A320 | ICAO type to pick a model by, e.g. `"A20N"`, `"B738"` |
| `via` | string | No | — | Taxiways to follow in order, e.g. `"F, L"` |
| `hold_for_clearances` | boolean | No | `true` | Wait at every step for `atc_clearance` |

**Returns**

The flight, as in [`list_our_traffic`](#list_our_traffic): `kind` `"departure"`, airport, model, stand, runway, `entry`, `procedure` (the SID), `taxi_route`, `state` (`"spawning"` at first) and `actions`.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 31,
  "method": "tools/call",
  "params": {
    "name": "spawn_departure",
    "arguments": { "icao": "LKPR", "callsign": "CSA123", "aircraft_type": "A320" }
  }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "callsign": "CSA123", "kind": "departure", "icao": "LKPR",
  "model": "FSLTL_FAIB_A320_SmartWings_CzechAirlinesLivery",
  "stand": "A1", "runway": "06", "procedure": "ARTU5E",
  "taxi_route": ["A1", "Z", "H", "F"],
  "state": "spawning", "position": { "lat": …, "lon": … }, …,
  "actions": ["remove"], "done": false
}
```

A few seconds later `list_our_traffic` shows it `"awaiting pushback"` with actions `["pushback", "taxi", "remove"]`.

**Error codes**

- `INVALID_ARGUMENT`: `icao` or `callsign` missing or malformed; the airport has no such runway or SID (the message lists the runway's SIDs).
- `TRAFFIC_ERROR`: The call sign is already ours, 32 aircraft already, no free stand that fits or an unknown or taken `stand`, no installed aircraft of the type (give `model`), or no taxi route.
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`: See above. Without `runway`, the weather is read to choose one.

---

## spawn_arrival

Put an AI arrival under our control into the simulator: at the STAR's first fix (or `spawn_nm` out on final with `star="none"`), flying the STAR and the best approach, landing, vacating and taxiing to a stand. With `hold_for_clearance` (default `true`) it waits clear of the runway for `atc_clearance` `taxi` and before runway crossings; `goaround` sends it around on final. Runway, stand, STAR and model are chosen when not given. Follow it with `list_our_traffic`.

> **Adds an aircraft to the simulator.** Take it out with `atc_clearance` `remove`.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled, and the airport loaded around the user aircraft.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | — | Airport ICAO code |
| `callsign` | string | Yes | — | Call sign, 2–8 letters or digits, e.g. `"DLH4AB"`; the first three letters pick the airline |
| `runway` | string | No | in use | Landing runway |
| `stand` | string | No | free stand that fits | Stand label; by default one near the runway |
| `star` | string | No | `"auto"` | STAR name, `"auto"` (the first STAR for the runway) or `"none"` (straight in on final) |
| `spawn_nm` | number | No | `5` | Straight in (`star="none"`, or no STAR for the runway): distance out on final to start, NM |
| `model` | string | No | chosen | Aircraft title from `list_aircraft_models` |
| `aircraft_type` | string | No | A320 | ICAO type to pick a model by |
| `via` | string | No | — | Taxiways to follow to the stand, in order |
| `hold_for_clearance` | boolean | No | `true` | Wait for the taxi clearance and at runway crossings |

Note the singular `hold_for_clearance` here and the plural `hold_for_clearances` of `spawn_departure`.

**Returns**

The flight, as in [`list_our_traffic`](#list_our_traffic): `kind` `"arrival"`, model, stand, runway, `procedure` (`"STAR → approach"`, omitted when straight in), `taxi_route` from the planned runway exit, `state` and `actions`.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 32,
  "method": "tools/call",
  "params": {
    "name": "spawn_arrival",
    "arguments": { "icao": "LKPR", "callsign": "DLH4AB", "star": "none", "spawn_nm": 4 }
  }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "callsign": "DLH4AB", "kind": "arrival", "icao": "LKPR",
  "model": "FSLTL A320 DLH Lufthansa",
  "stand": "N50", "runway": "06", "taxi_route": ["B", "G"],
  "state": "spawning", …, "actions": ["remove"], "done": false
}
```

It then goes `"approaching"` (e.g. 827 ft AGL at 143 kt, actions `["goaround", "taxi", "remove"]`), `"landing"`, `"rollout"`, `"vacating"` by B, and waits `"awaiting taxi"`.

**Error codes**

- `INVALID_ARGUMENT`: `icao` or `callsign` missing or malformed; the airport has no such runway or STAR (the message lists the runway's STARs).
- `TRAFFIC_ERROR`, `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`: As for `spawn_departure`.

---

## list_our_traffic

List the AI aircraft under our control (from `spawn_departure` and `spawn_arrival`): state, position, speed, current taxiway, what it is holding short of, errors, and `actions` — the clearances `atc_clearance` takes now. Poll it to follow the flights. A flight that has ended stays listed (`done: true`) until `atc_clearance` `remove`.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**: None.

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `count` | number | Flights of ours |
| `flights` | array | Flights, sorted by call sign (see below) |

Each flight:

| Field | Type | Description |
|-------|------|-------------|
| `callsign` | string | Call sign |
| `kind` | string | `"departure"` or `"arrival"` |
| `icao` | string | Airport |
| `model` | string | Aircraft title, followed by a vertical bar and the livery when there is one |
| `stand`, `runway` | string | Stand and runway |
| `entry` | string | Departure runway entry, when given |
| `procedure` | string | SID, or `"STAR → approach"` |
| `taxi_route` | array | Planned taxiways |
| `state` | string | See [Clearance flow](#clearance-flow) |
| `taxiway` | string | Taxiway it is on |
| `holding_short_of` | string | Runway it is holding short of |
| `remaining_m` | number | Metres to the hold-short point (departure) or the stand (arrival) |
| `position` | object | `{lat, lon}` |
| `heading` | number | True heading, degrees |
| `ground_speed_kts` | number | Ground speed, knots |
| `agl_ft` | number | Height above ground, feet |
| `on_ground` | boolean | `true` on the ground |
| `lights` | string | `NBSTLOW` — nav, beacon, strobe, taxi, landing, logo, wing — with a dot for each light off, e.g. `"NB...O."` |
| `error` | string | Why it failed, when it did |
| `actions` | array | `atc_clearance` actions that fit now |
| `done` | boolean | `true` once the flight has ended (`complete`, `parked`, `cancelled`, `failed`); only `remove` is left |

Empty strings and zero `remaining_m` / `agl_ft` are omitted.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 33,
  "method": "tools/call",
  "params": { "name": "list_our_traffic", "arguments": {} }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "count": 1,
  "flights": [
    {
      "callsign": "CSA123", "kind": "departure", "icao": "LKPR", "stand": "A1", "runway": "06",
      "procedure": "ARTU5E", "taxi_route": ["A1", "Z", "H", "F"],
      "state": "pushback", "position": { "lat": …, "lon": … }, "heading": …, "ground_speed_kts": 3,
      "on_ground": true, "lights": "NB...O.", "actions": ["taxi", "takeoff", "remove"], "done": false
    }
  ]
}
```

**Error codes**: None — with no aircraft of ours it returns `{"count": 0, "flights": []}`.

---

## atc_clearance

Give one of our AI aircraft a clearance or instruction. Departures: `pushback`, `taxi` (to the holding point), `cross` (a runway on the way), `lineup` (line up and wait), `takeoff`, `hold` (hold position), `abort` (reject the take-off before V1). Arrivals: `goaround` (on final), `taxi` (to the stand), `cross`, `hold`. Both: `remove` (take it out of the simulator). A clearance given early means no stop there. See [Clearance flow](#clearance-flow).

> **`remove` takes the aircraft out of the simulator**, frees its stand and forgets the flight.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `callsign` | string | Yes | — | Call sign of one of ours (case-insensitive) |
| `action` | string | Yes | — | `pushback`, `taxi`, `cross`, `lineup`, `takeoff`, `hold`, `abort`, `goaround` or `remove` |

**Returns**

The flight as the clearance finds it, as in [`list_our_traffic`](#list_our_traffic); the state changes as the aircraft reacts, so follow it with `list_our_traffic`.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 34,
  "method": "tools/call",
  "params": {
    "name": "atc_clearance",
    "arguments": { "callsign": "CSA123", "action": "pushback" }
  }
}
```

**Example response** (abridged)

```json
{ "callsign": "CSA123", "kind": "departure", "state": "awaiting pushback", …, "actions": ["pushback", "taxi", "remove"], "done": false }
```

Moments later the flight is `"pushback"` at about 3 kt with lights `"NB...O."`; `taxi` then has it `"taxiing"` at up to 15 kt, `hold` stops it (0 kt) and `taxi` sends it on.

An action that does not fit, e.g. `lineup` for an arrival on its landing roll:

```
TRAFFIC_ERROR: lineup DLH4AB: DLH4AB (arrival, rollout) takes taxi, remove, not "lineup"
```

**Error codes**

- `NOT_FOUND`: The call sign is not one of ours.
- `TRAFFIC_ERROR`: The action is not one of the flight's kind (the message lists the actions that fit now), or the library refused it, e.g. `abort` past V1 or `goaround` on the runway.

---

## get_traffic_picture

The traffic picture: every aircraft the simulator has around the user aircraft (or an airport) with call sign, aircraft title, position, altitude, ground speed, heading, vertical speed and phase — `parked`, `taxiing`, `runway`, `departing`, `enroute` or `arriving` — and the airport it belongs to. The user aircraft and ours are marked. The first call starts the scan and takes a few seconds.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `centre` | string | No | user aircraft | Airport ICAO code to centre on |
| `radius_nm` | number | No | `40` | Radius in NM, at most `40`. The scan reaches about 43 NM from the user aircraft, so an airport farther away shows only what is in reach |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `centre` | string | Airport centred on (empty: the user aircraft) |
| `radius_nm` | number | Radius used |
| `count` | number | Aircraft returned |
| `aircraft` | array | Aircraft (see below) |

Each aircraft: `callsign` (ATC ID), `title`, `phase`, `airport` (omitted enroute), `lat`, `lon`, `alt_ft`, `agl_ft`, `ground_speed_kts`, `heading_true`, `vs_fpm`, and `user: true` for the user aircraft or `ours: true` for one of ours.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 35,
  "method": "tools/call",
  "params": {
    "name": "get_traffic_picture",
    "arguments": { "radius_nm": 10 }
  }
}
```

**Example response** (MSFS 2024, abridged)

```json
{
  "centre": "", "radius_nm": 10, "count": 2,
  "aircraft": [
    { "callsign": "…", "title": "…", "phase": "parked", "airport": "LKPR", "lat": …, "lon": …, "alt_ft": …, "ground_speed_kts": 0, …, "user": true },
    { "callsign": "CSA123", "title": "FSLTL_FAIB_A320_SmartWings_CzechAirlinesLivery", "phase": "taxiing", "airport": "LKPR", "lat": …, "lon": …, …, "ours": true }
  ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `centre` is not an airport ICAO code, or `radius_nm` is outside 1–40.
- `NOT_FOUND`, `TIMEOUT`, `BRIDGE_DISCONNECTED`, `TRAFFIC_ERROR`: See above.

---

## generate_schedule

Generate a realistic airline schedule for airports: flights with call sign, airline, aircraft type, origin, destination, STD/STA (UTC) and distance, following time-of-day waves and each airline's bases and fleet (17 European airlines, about 80 airports). Deterministic for a seed. Pure computation — nothing is spawned; use it to pick flights for `spawn_departure` and `spawn_arrival`.

**Requirements**: Registered with the other traffic tools; it does not read the simulator.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `airports` | string | Yes | — | Airports to schedule, e.g. `"LKPR"` or `"LKPR, EDDM"` |
| `hours` | number | No | `2` | Hours from `start`, 1–24 |
| `start` | string | No | now (UTC) | Start time, RFC 3339, e.g. `"2026-10-01T06:00:00Z"` |
| `density` | number | No | `1` | Traffic density, 0.1–3 |
| `seed` | number | No | `1` | Random seed |
| `limit` | number | No | `100` | Maximum flights returned, 1–500 |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `from` | string | Start time (UTC) |
| `hours` | number | Hours scheduled |
| `total` | number | Flights generated |
| `count` | number | Flights returned (at most `limit`) |
| `flights` | array | `{callsign, airline, type, origin, destination, std, sta, distanceNM}`; `airline` and `type` are ICAO codes |
| `warning` | string | Airports not in the schedule's airport list (few or no flights) |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 36,
  "method": "tools/call",
  "params": {
    "name": "generate_schedule",
    "arguments": { "airports": "LKPR", "hours": 1, "start": "2026-10-01T06:00:00Z" }
  }
}
```

**Example response (abridged)**

```json
{
  "from": "2026-10-01T06:00:00Z", "hours": 1, "total": …, "count": …,
  "flights": [
    { "callsign": "…", "airline": "…", "type": "…", "origin": "LKPR", "destination": "…", "std": "…", "sta": "…", "distanceNM": … },
    …
  ]
}
```

**Error codes**

- `INVALID_ARGUMENT`: `airports` missing or not ICAO codes; `hours`, `density` or `limit` out of range; `start` not RFC 3339.

---

## Airborne ATC tools

The live runtime runs a controller for every airport where we have traffic, using the library's v0.16 (airborne ATC):

- **A tower per runway** clears our departures' line-up and take-off and our aircraft's runway crossings. It works in mixed mode: departures go in the gaps between arrivals, after the wake and same-SID interval. It sends an arrival around when the runway will not be free on short final (someone lined up, crossing, or still on it after landing). The go-around flies the published missed approach and is sequenced again.
- **A landing sequence per runway end**, first come first served, of our arrivals and the other traffic on the final. Spacing is the wake minimum, at least 5 NM, and more in low visibility or on a contaminated runway (the weather at the user aircraft). Our arrivals lose their delays by themselves: slower first, then a longer downwind, then a hold at the STAR fix (stacked 1000 ft apart), left once the delay is down to a minute.

Flights spawned with `hold_for_clearances` / `hold_for_clearance` are yours: the tower and the sequence count them but never clear or instruct them.

The five tools below read the sequence and the tower, give approach instructions, predict conflicts, and compute separation minima.

---

## get_landing_sequence

The landing sequence of each runway end with our arrivals, and who uses each runway now.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | No | all | Airport ICAO code |

**Returns**: `sequences` has one entry per runway end: `icao`, `runway`, `conditions` (visibility, ceiling, headwind, surface), and `arrivals`, first to land first. Each arrival has:

- `number`, `callsign`, `wake` (`M/D`: ICAO/RECAT-EU);
- `behind` and `spacing_nm`, with `spacing_why` when the conditions change it;
- `distance_to_go_nm` along what it still flies;
- `predicted_landing` and `sequenced_landing` (UTC);
- `delay_s`, and `established` inside 8 NM.

`tower` lists who uses each runway: `runway`, `callsign`, `phase` (holding short, lined up, on the runway, final), `ours`, and `waiting` (why it waits, e.g. `"CSA1 on a 3.0 NM final"`, `"1m20s behind DLH2"`).

---

## approach_instruction

Give one of our arrivals in a landing sequence an approach controller's instruction. It returns what was said (`said`).

| `instruction` | Effect |
|---|---|
| `up`, `down` | A place earlier or later in the landing order; it keeps the place |
| `slow` | Loses another minute: slower, then a longer downwind |
| `hold` | Holds at its STAR's hold fix, stacked above the others |
| `release` | Leaves the hold and continues the arrival |
| `direct` | Straight to the final, leaving out the rest of its STAR |
| `goaround` | Goes around (the published missed approach) and is sequenced again |

**Error codes**:

- `NOT_FOUND`: not in a landing sequence.
- `NOT_APPLICABLE`: established (inside 8 NM), on the final, already holding or not holding.
- `INVALID_ARGUMENT`: an unknown instruction.

---

## get_atc_log

The latest instructions of the runtime's controllers, newest last, as ATC says them. For example:

- `"CSA123, runway 24, line up, cleared for take-off"`
- `"DLH4AB, number 2, delay 1m30s: 210 kt, +4.1 NM"`
- `"KLM7, hold at PR722, direct entry, maintain 7000 ft, expect further clearance 1042Z"`
- `"CSA1, go around, I say again, go around — TVS3 on the runway"`

Parameters: `icao`, `callsign`, `limit` (1–200, default 50). Returns `messages`: `at`, `icao`, `callsign`, `text`.

---

## get_conflicts

Airborne conflicts in the traffic picture: pairs that, flying on as they are (track, ground speed, vertical speed), come closer than 5 NM (3 NM near an airport) and 1000 ft within the look-ahead.

- Departures and arrivals at the same airport low near the runway are left to the tower.
- Where one of the pair is ours, the least disturbing resolution comes as advice: a speed (±10–20 %), a level (1000 or 2000 ft, by the semicircular rule) or a heading (20–45°), whichever keeps it clear of everyone.

Parameters: `centre`, `radius_nm` (1–40), `lookahead_min` (1–10, default 5).

Returns `conflicts`. Each conflict has `pair`, `loss_in_s`, `closest_nm`, `closest_vertical_ft`, `closest_in_s` and `minimum_nm`, plus either:

- `resolution`: `callsign`, `kind` (speed, level, heading), `kts`, `altFt` or `headingDeg`, and `why`;
- `resolution_note`, when there is none.

---

## separation_minima

Wake turbulence and runway separation for a pair of aircraft types. It is a pure calculation.

Parameters:

- `leader`, `follower`: ICAO types, required;
- `scheme`: `icao` or `recat`;
- `visibility_m`;
- `surface`: `dry`, `wet` or `contaminated`;
- `same_route`: `1` for the same SID.

It returns:

- each type's `wake` (ICAO and RECAT-EU) and `landing_occupancy_s`;
- `wake_minimum_nm`, and `spacing_on_final_nm` with `spacing_why` (at least 6 NM in low visibility, +1 NM contaminated, reduced 2.5 NM only in good conditions);
- `departure_interval_s` and `follower_takeoff_occupancy_s`.
