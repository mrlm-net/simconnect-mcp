---
title: "MCP Tools — SimConnect Mode"
description: Reference for the 10 live-data MCP tools in SimConnect mode (MCP_MODE=simconnect, Windows only).
order: 2
section: reference
---

All 10 MCP tools listed here are available when the server runs with `MCP_MODE=simconnect`. This mode provides live simulator data via the SimConnect SDK.

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
- `INVALID_ARGUMENT`: `icao` was not provided or is empty.
- `AIRPORT_NOT_FOUND`: No airport matching the given ICAO code was found.
- `AIRPORT_DETAILS_ERROR`: SimConnect returned an error while fetching facility data.
