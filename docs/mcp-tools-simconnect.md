---
title: "MCP Tools — SimConnect Mode"
description: Reference for the 59 live-data, user aircraft, AI traffic, airborne ATC and scheduled traffic MCP tools in SimConnect mode (MCP_MODE=simconnect, Windows only).
order: 2
section: reference
---

All 59 MCP tools listed here are available when the server runs with `MCP_MODE=simconnect` (and, on Windows, with `MCP_MODE=both`, alongside the 15 docs tools — 74 in all). This mode provides live simulator data via the SimConnect SDK, and AI traffic under our control.

**Both mode**: with `MCP_MODE=both` on Windows, the server registers these 59 tools alongside the 15 [docs-mode tools](/docs/mcp-tools-docs) — 74 tools in total — provided SimConnect opens at startup (10-second timeout). If the simulator cannot be reached, or on non-Windows platforms, both mode serves the 15 docs tools only; `simconnect_ready` in the `/health` response reports which case applies.

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
| [`get_fuel_state`](#get_fuel_state) | The user aircraft's fuel: total, weight and the main tanks |
| [`get_aircraft_systems`](#get_aircraft_systems) | The user aircraft's systems through its profile: power, radios, lights, doors, transponder, ground equipment |
| [`set_aircraft_control`](#set_aircraft_control) | Operate a door, chocks, GPU, parking brake, cabin signs, external power or the cabin call |
| [`request_ground_service`](#request_ground_service) | Ask for the sim's jetway, stairs, baggage, catering, ground power, fuel truck or pushback |
| [`set_radio`](#set_radio) | Set a COM frequency, swap a COM, or set the squawk |
| [`set_atc_callsign`](#set_atc_callsign) | Set the call sign the sim's ATC uses for the user aircraft |
| [`list_addons`](#list_addons) | The installed MSFS packages: Community, Official and streamed |
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
| [`get_landing_sequence`](#get_landing_sequence) | The landing sequence of each runway end with our arrivals, and who uses each runway |
| [`approach_instruction`](#approach_instruction) | An approach instruction to one of our arrivals: up, down, slow, hold, release, direct, goaround |
| [`get_atc_log`](#get_atc_log) | The latest instructions of the runtime's tower and approach to our traffic |
| [`get_conflicts`](#get_conflicts) | Predict airborne conflicts, with the least disturbing resolution for ours as advice |
| [`separation_minima`](#separation_minima) | Wake categories, spacing on final, departure interval and runway occupancy for a pair of types |
| [`start_schedule`](#start_schedule) | Run a realistic airline schedule at airports: departures and arrivals appear and go by themselves |
| [`stop_schedule`](#stop_schedule) | Stop the schedule (and remove its aircraft) |
| [`get_schedule`](#get_schedule) | The running schedule's departure and arrival boards |
| [`add_flights`](#add_flights) | Add flights to the schedule at chosen times |
| [`set_real_traffic`](#set_real_traffic) | Fly real-world traffic at an airport instead of the timetable |
| [`observe_traffic`](#observe_traffic) | Feed real-world sightings (ADS-B) to the engine, or drop them |
| [`set_traffic_corridor`](#set_traffic_corridor) | Keep airliners around the user's flight in cruise |
| [`set_player_clearance`](#set_player_clearance) | Tell the engine what the user's ATC cleared, so our traffic keeps off the runway |
| [`get_traffic_status`](#get_traffic_status) | The traffic engine's state and settings |
| [`get_traffic_airport_info`](#get_traffic_airport_info) | An airport as the engine works it: runways in use, ATIS, ILS |
| [`get_tcas`](#get_tcas) | TCAS advisories (TA, RA) of our airborne traffic |

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

## get_fuel_state

The user aircraft's fuel: total quantity and capacity, percent full and weight, with the center, left main and right main tanks. Tanks the aircraft does not have (zero capacity) are left out. It reads all nine SimVars in one request.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**: none.

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `total_quantity_gal` | number | Fuel on board, US gallons |
| `total_capacity_gal` | number | Usable capacity, US gallons |
| `total_percent_full` | number | Quantity as a percentage of capacity (0 when capacity is 0) |
| `total_weight_lbs` | number | Fuel weight, pounds |
| `total_weight_kg` | number | Fuel weight, kilograms |
| `tanks` | array | `name` (Center, Left Main, Right Main), `quantity_gal`, `capacity_gal`, `percent_full` |

**Example response** (an A320 with 50 % of its fuel, none in the center tank)

```json
{
  "total_quantity_gal": 3200, "total_capacity_gal": 6400, "total_percent_full": 50,
  "total_weight_lbs": 21440, "total_weight_kg": 9725,
  "tanks": [
    { "name": "Center", "quantity_gal": 0, "capacity_gal": 2200, "percent_full": 0 },
    { "name": "Left Main", "quantity_gal": 1600, "capacity_gal": 1800, "percent_full": 88.9 },
    { "name": "Right Main", "quantity_gal": 1600, "capacity_gal": 1800, "percent_full": 88.9 }
  ]
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `INTERNAL_ERROR`: The simulator refused a fuel SimVar.

---

## get_aircraft_systems

The user aircraft's systems, read through its systems profile (the library's `pkg/systems`). The profile is the standard SimVars, the library's shipped profile for the model on top (the Fenix A320 family reads power, radios, chocks and GPU from its own L:vars and tablet), then the local override files from `SIMCONNECT_AIRCRAFT_PROFILES` on top of that. The model is matched by its package (found from the aircraft the sim loaded, with `pkg/addons`), its title or its ATC type.

**Requirements**: Windows + MSFS 2020 or 2024 running with SimConnect enabled.

**Parameters**: none.

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `aircraft` | object | `title`, `atc_type` (empty when the aircraft gives an untranslated key), `path` (the aircraft.cfg loaded), `package` and `package_source` when found |
| `profile` | object | `name`, `measured` (how the model's profile was measured), `local_overrides` and `local_override_errors` |
| `power` | object | `battery`, `powered`, `bus_volts`, `avionics`, `external_available`, `external_on` |
| `radios` | object | `com1`, `com2`: `working`, `active_mhz`, `standby_mhz` (0 when the radio gives none, e.g. dark) |
| `transponder` | object | `state` (`off`, `standby`, `test`, `on`, `alt`) and `squawk` |
| `engines` | array | `engine`, `running`, `starter` per engine |
| `parking_brake`, `gear_down` | boolean | |
| `flaps_pct` | number | Flaps handle, percent |
| `lights` | object | `beacon`, `nav`, `strobe`, `landing`, `taxi` |
| `doors` | array | Every door the profile names: `control` (`door0`…, for `set_aircraft_control`), `name` (`L1`, `FWD cargo`), `open` |
| `ground` | object | `chocks`, `gpu` and whether the model has them (`has_chocks`, `has_gpu`); the sim's pushback: `pushback_attached`, `pushback_available`, `pushback_wait` |
| `seatbelts` | boolean | When the profile gives the sign |
| `no_smoking` | number | 0 off, 1 auto, 2 on, when the profile gives the sign |
| `values` | object | Every value the profile resolved, by name |
| `can`, `cannot` | array | Controls and ground services the profile can and cannot operate |

**Example response** (a cold and dark Fenix A319, shortened)

```json
{
  "aircraft": { "title": "FenixA319 CFM WF HD", "atc_type": "", "package": "fnx-aircraft-319-321", "package_source": "Community" },
  "profile": { "name": "Fenix A320 family" },
  "power": { "battery": false, "powered": false, "bus_volts": 27.5, "external_available": true, "external_on": false },
  "radios": { "com1": { "working": false, "active_mhz": 0, "standby_mhz": 0 } },
  "transponder": { "state": "off", "squawk": "2000" },
  "doors": [ { "control": "door0", "name": "L1", "open": false }, { "control": "door4", "name": "FWD cargo", "open": false } ],
  "ground": { "chocks": true, "has_chocks": true, "gpu": true, "has_gpu": true, "pushback_available": true },
  "can": ["baggage", "cabinCall", "chocks", "door0", "gpu", "jetway", "parkingBrake", "pushback", "seatbelts"]
}
```

**Error codes**

- `BRIDGE_DISCONNECTED`: Not connected to the simulator.
- `TIMEOUT`: The simulator did not answer in 5 s.

---

## set_aircraft_control

Operate one of the user aircraft's controls the way its profile says: the standard key events by default, the model's own variables or tablet where it has them (the Fenix's chocks and GPU go through its EFB). A toggling event is sent only when the state differs from the one asked for.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `control` | string | Yes | | A door by name (`L1`, `Door 2`) or `door0`…; `chocks`, `gpu`, `parking_brake`, `seatbelts`, `ext_power`, `no_smoking`, `cabin_call` |
| `state` | string | No | `on` | `on`/`open` or `off`/`closed`; `auto` for `no_smoking`. Ignored for `cabin_call`, which is a press |

**Returns** `control`, `requested` (1 on, 0 off; `no_smoking` 0–2) and `state_now`, the value read right after the command (doors and tablet controls take a few seconds); `cabin_call` returns `pressed`.

**Error codes**

- `INVALID_ARGUMENT`: Unknown control or state; the message lists the aircraft's doors.
- `NOT_APPLICABLE`: The aircraft's profile gives no way to operate it (`get_aircraft_systems` lists `can`).
- `BRIDGE_DISCONNECTED`, `TIMEOUT`.

---

## request_ground_service

Ask for one of the simulator's own ground services for the user aircraft, by the MSFS key events (`TOGGLE_JETWAY`, `TOGGLE_RAMPTRUCK`, `REQUEST_LUGGAGE`, `REQUEST_CATERING`, `REQUEST_POWER_SUPPLY`, `REQUEST_FUEL_KEY`, `TOGGLE_PUSHBACK`) or the model's own way where its profile gives one. The jetway, stairs and pushback toggle: asking again sends them away. The simulator decides whether the service can come where the aircraft is.

**Parameters**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `service` | string | Yes | `jetway`, `stairs`, `baggage`, `catering`, `powerSupply`, `fuelTruck` or `pushback` (`fuel_truck` style works too) |

**Returns** `service` and `requested: true`.

**Error codes**: `INVALID_ARGUMENT`, `NOT_APPLICABLE`, `BRIDGE_DISCONNECTED`, `TIMEOUT`.

---

## set_radio

Set the user aircraft's radios with the library's `pkg/avionics`. The aircraft must be powered: a dark radio ignores it. On the Fenix the swap presses its RMP transfer key (`L:S_PED_RMP1_XFER`), because the stock swap event does not reach its RMP. `get_aircraft_systems` reads the result back.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `action` | string | Yes | | `com_active`, `com_standby`, `com_swap` or `squawk` |
| `com` | number | No | `1` | COM radio 1, 2 or 3 |
| `frequency_mhz` | number | For `com_active`, `com_standby` | | 118.000–136.990 MHz, 8.33 kHz channels included |
| `squawk` | string | For `squawk` | | Four octal digits, e.g. `"4521"` |

**Returns** the request and `sent: true`.

**Error codes**

- `INVALID_ARGUMENT`: A radio other than 1–3, a frequency out of the band, or a squawk that is not four digits 0–7. Nothing is sent.
- `BRIDGE_DISCONNECTED`, `TIMEOUT`.

---

## set_atc_callsign

Set the call sign the simulator's ATC uses for the user aircraft: `ATC AIRLINE` (the airline's call sign as said) and `ATC FLIGHT NUMBER`. Either may be left out to keep it. The registration (`ATC ID`) is not changed.

**Parameters**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `airline` | string | No | The spoken call sign, e.g. `"Speedbird"`, up to 63 characters |
| `flight_number` | string | No | The flight number, e.g. `"123"`, up to 7 characters |

At least one is required. **Returns** `set: true` and the values set.

**Error codes**: `INVALID_ARGUMENT`, `BRIDGE_DISCONNECTED`.

---

## list_addons

What is installed in Microsoft Flight Simulator on this machine, with the library's `pkg/addons`. It reads files only: the `UserCfg.opt` of MSFS 2024 or 2020 (Steam or Microsoft Store) gives the packages folder, then each package's `manifest.json`. Streamed packages have no manifest; when a folder name reads as an airport (`fs20-orbx-airport-lkpr-prague`), its ICAO and publisher are given. A streamed folder shows that the sim knows the package, not that it is owned. The scan is cached for 5 minutes.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `source` | string | No | `Community` | `Community` (both Community folders), `Community2024`, `Official`, `Streamed` or `all` |
| `search` | string | No | | Only packages whose folder, title, creator or ICAO contains this text |
| `refresh` | boolean | No | `false` | Scan the disk again |
| `limit` | number | No | `200` | At most this many packages |

**Returns**

| Field | Type | Description |
|-------|------|-------------|
| `install` | object | `sim` (`2024`, `2020`), `store` (`steam`, `store`), `packages_path` |
| `fingerprint` | string | Changes whenever the set of packages changes |
| `by_source` | object | Package count per source |
| `matched` | number | Packages matching the filters (the list stops at `limit`) |
| `packages` | array | `source`, `folder`, `title`, `creator`, `content_type` (as written, not reliable), `version`; streamed airports `icao`, `publisher`, `cached_archives` |

**Error codes**

- `NOT_FOUND`: No MSFS 2024 or 2020 installation found.
- `SIM_ERROR`: The packages folder could not be read.

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

Work out the runways in use at an airport as a tower would: the preferential runway if the wind allows, else the one with the most headwind, within the airport's tailwind and crosswind limits. Returns the departure and arrival runway, wind components, whether an ILS or visual approach is expected and the best published approach, the transition altitude and level. Once a runway is in use it stays in use while the wind allows it (up to 5 kt tailwind), as at a real airport; a calm or variable wind doesn't swap it. The spawn tools and the schedule use the same runways. Uses the weather at the user aircraft, so it is right for the airport the aircraft is at or near.

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

The AI traffic and ATC tools run on the library's traffic engine, `pkg/traffic/world` (the engine of the library's airport map), on the server's own SimConnect connection. The engine starts with the first tool that needs it, and runs again after the simulator reconnects, with its schedule, real traffic and corridor set again. What it does for each aircraft of ours:

- **Departures:** stand services from the airport's fleet (a fuel truck; boarding stairs and a GPU at a remote stand), a pushback with a tug, the taxi route to the runway, line-up, take-off and the SID. The crew talks to delivery, ground and tower.
- **Arrivals:** en route or on a STAR, sequenced by the approach controller (speed, vectors, holds), the approach, landing, vacating and taxi to a stand. Turnarounds stay on their stand and depart again.
- **ATC:** a tower per runway, landing sequences, airborne separation with conflict resolutions, and the radio (`get_atc_log`).

**Manual or automatic.** An aircraft the engine clears by itself is *automatic*. One spawned with `hold_for_clearances` / `hold_for_clearance` (the default), or given any `atc_clearance` other than `remove`, is *manual*: it waits for your clearances. `atc_clearance` with `action: "manual", on: false` hands it back to the engine.

**Shutdown.** When the server stops it turns the schedule and corridor off and removes our aircraft from the simulator. En route aircraft the schedule created are not on the engine's list; they go when the connection closes.

**IDs.** The engine's library helpers sit at IDBase 950,000,000; its fixed IDs (2000–2021, 8200+, 10010, 20000–31279) clash with none of the server's.

---

## list_aircraft_models

The installed models the engine can spawn: titles, with the livery after the separator.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `search` | string | No | | Only titles containing this text |
| `limit` | number | No | `100` | At most this many |

Returns `count` (all matching) and `models`.

---

## spawn_departure

Puts an AI departure of ours on a stand. The stand, the runway in use, the SID and the model are chosen when not given.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | | The airport; the simulator must have it loaded around the user aircraft |
| `callsign` | string | Yes | | 2–8 letters and digits, not already ours |
| `stand` | string | No | free stand that fits | Stand label, e.g. `"C22"` |
| `runway` | string | No | in use | Departure runway |
| `entry` | string | No | full length | Runway entry taxiway for an intersection departure |
| `sid` | string | No | `auto` | A SID name, `auto` or `none` |
| `model` | string | No | | A title from `list_aircraft_models` |
| `aircraft_type` | string | No | | ICAO type to pick a model by when no model is given |
| `via` | string | No | | Taxiways in order, e.g. `"F, L"` |
| `squawk` | string | No | assigned | SSR code |
| `push_in_min` | number | No | when ready | Push this many minutes from now, giving the stand services their time |
| `stand_use` | string | No | airliner | `ga` or `cargo` when no stand is given |
| `hold_for_clearances` | boolean | No | `true` | Wait for `atc_clearance` at every step |
| `tug` | boolean | No | `true` | A tug pushes it |
| `fuel` | boolean | No | `true` | A fuel truck comes before the push |

Returns the aircraft as `list_our_traffic` shows it.

**Error codes**: `INVALID_ARGUMENT`, `NOT_FOUND` (stand), `NOT_APPLICABLE` (the engine refused the spawn, e.g. no flight plan), `BRIDGE_DISCONNECTED`, `TIMEOUT`.

---

## spawn_arrival

Puts an AI arrival of ours into the simulator, on its STAR (or straight in with `star: "none"`). It is sequenced by the approach controller, lands, vacates and taxis to a stand.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | Yes | | The airport |
| `callsign` | string | Yes | | Call sign |
| `runway` | string | No | in use | Landing runway |
| `stand` | string | No | free stand that fits | Stand label |
| `star` | string | No | `auto` | A STAR name, `auto` or `none` |
| `model`, `aircraft_type`, `via` | | No | | As for `spawn_departure` |
| `hold_for_clearance` | boolean | No | `true` | Wait for the taxi clearance after vacating |
| `turnaround` | boolean | No | `false` | Stay on the stand and depart again |
| `dwell_min` | number | No | the engine's | With `turnaround`: minutes on the stand |

---

## list_our_traffic

Our aircraft, spawned or scheduled. Each one has:

- **Identity:** `id`, `callsign`, `kind` (`departure`, `arrival`), `rules`, `icao`, `model`, `squawk`.
- **Plan:** `stand`, `runway`, `procedure`.
- **State:** `state`, `atc` and `frequency` (the position working it), `onGround`, `position`, `heading`, `groundSpeed`, `holdingShortOf`, `taxiRemainingM`, `entry`, `deicing`, `pushbackHeld`, `error`.
- **Control:** `manual` and `actions` (what `atc_clearance` takes now).
- **Ground vehicles:** `vehicles`, each with `kind`, `title`, `state` (`waiting`, `inbound`, `attached`, `fuelling`, `outbound`, `removed`), `position`, `heading` and route.
- **Real traffic:** `real`, `observedId`, `registration`.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | No | | Only this airport's aircraft |
| `detail` | boolean | No | `false` | Include routes, taxi nodes and air fixes |

Before the engine has started it returns no aircraft and `engine: "not started"`.

---

## atc_clearance

A clearance or instruction to one of ours. Any action but `remove` puts the aircraft under manual control.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `callsign` | string | Yes | One of ours (or its engine `id`) |
| `action` | string | Yes | See below |
| `stand` | string | For `standto` | The stand label |
| `entry` | string | For `entry` | The entry taxiway (`""` full length) |
| `node` | number | For `upto` | The taxi node to stop at |
| `facing` | string | No | `pushback`: the direction to face after the push, e.g. `"east"` |
| `behind` | string | For `follow` | The call sign of the one of ours to follow, at the same airport |
| `startup` | boolean | No | `pushback`: start engines during the push |
| `on` | boolean | No | `manual`, `rush`: on (default) or off |

| Action | For | What it does |
|--------|-----|--------------|
| `pushback`, `pushstart`, `startup` | departures | Push back (and start); start engines |
| `taxi`, `upto`, `cross`, `hold` | both | Taxi on, taxi up to a node, cross a runway, hold position |
| `lineup`, `lineupbehind`, `takeoff`, `abort` | departures | Line up (behind the one landing), take off, reject the take-off |
| `entry`, `rush` | departures | Change the runway entry; expedite |
| `land`, `goaround` | arrivals | Cleared to land (spoken); go around |
| `standto` | arrivals | Another stand |
| `depart` | turnarounds | Depart now, skipping the dwell |
| `follow` | both, on the ground | Stays behind another of ours (`behind`) wherever their ways meet, told the way ground says it |
| `manual` | both | `on: true` the caller clears it; `on: false` the engine does |
| `remove` | both | Takes it out of the simulator and frees its stand and vehicles |

Returns the aircraft after the action; `removed: true` after `remove`.

**Error codes**: `NOT_FOUND` (not one of ours), `NOT_APPLICABLE` (the action does not fit now: see `actions`), `INVALID_ARGUMENT`.

---

## get_traffic_picture

Every aircraft the simulator has around the user aircraft, or around an airport: call sign, title, phase (`parked`, `taxiing`, `runway`, `departing`, `enroute`, `arriving`, `holding`), the airport it belongs to, position, altitude, AGL, ground speed, heading and vertical speed. The user's and ours are marked. It reads the engine's picture as it is (the engine's area is not moved) and keeps the aircraft within `radius_nm`; the first call can take a few seconds.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `centre` | string | No | the user aircraft | Airport ICAO code to centre on |
| `radius_nm` | number | No | `40` | Radius, 1–40 NM |

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

The engine runs, at each airport it works, a tower per runway, a landing sequence per runway end (spacing by wake, at least the minimum, more in low visibility), airborne separation with conflict resolutions for ours, and the radio. Manual aircraft are counted but never cleared or instructed by it.

---

## get_landing_sequence

The landing sequences at an airport the engine works, or at all of them: per runway its approach `conditions`, `lvp`, and `sequence`. Each entry has `callsign`, `number`, `leader`, `wake`, `spacingNM`, `spacingWhy`, `minimumNM`, `distanceToGoNM`, `eta`, `landing`, `delay` (nanoseconds) and `fixed` (established).

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | No | every airport the engine works | The airport |

---

## approach_instruction

An approach controller's instruction to one of our arrivals.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `callsign` | string | Yes | Our arrival |
| `instruction` | string | Yes | `up`, `down`, `slow`, `speed`, `hold`, `release`, `direct`, `joinfinal`, `holdat`, `goaround` |
| `kts` | number | For `speed` | The speed; 0 resumes normal speed |
| `lat`, `lon` | number | For `joinfinal`, `holdat`; optional for `direct` | The point |

`up` / `down` move it a place in the landing order, `slow` loses another minute, `hold` holds at its STAR's hold fix (`release` ends it), `direct` goes to the point (else to the final), `joinfinal` joins the final at the point, `holdat` holds at the point.

**Error codes**: `NOT_FOUND` (not in a sequence), `NOT_APPLICABLE` (not one of our arrivals, established), `INVALID_ARGUMENT`.

---

## get_atc_log

The radio: the engine's latest transmissions, oldest first, each with `at`, `airport`, `frequency`, `position` (delivery, ground, tower, approach), `controller` or `pilot`, `callsign`, `intent`, `params` and `text`.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `icao` | string | No | all | Only this airport's |
| `limit` | number | No | `30` | At most this many |

---

## get_conflicts

Airborne separation as the engine sees it: `minNM`, the `closest` pairs, `open` and past `losses` of separation, predicted `conflicts`, and the `resolutions` given to ours with what was said.

---

## get_traffic_status

The engine's state: `running`, `connected`, `aircraft` (ours), `dropped_messages`, the `schedule`, `real_traffic` and `corridor` settings, `player` (the user's place in a landing sequence) and the last `error`.

---

## get_traffic_airport_info

An airport as the engine works it: `runways`, `use` (the runways in use with head- and crosswind, within limits), `atis` (letter, text, spoken), `ils`, `limits` and `weather`. While our traffic runs there, this is what its ATC uses; `get_atis` and `get_active_runway` compute their own.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `icao` | string | Yes | The airport |

---

## get_tcas

TCAS II for our airborne traffic. Each of ours sees every aircraft around it (ours, the user's and other traffic) within 12 NM, as TCAS II does. It gets traffic advisories (TA) and resolution advisories (RA), flies an RA after the crew's reaction time and reports it on the frequency ("TCAS RA"), then "clear of conflict, returning to assigned altitude". Meanwhile approach gives it no instruction. Between two of ours the RAs are coordinated. Each aircraft's advisory now is also on `list_our_traffic` as `tcas` (`advisory`, `intruder`, `aural`, `sense`).

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `limit` | number | No | `30` | At most this many latest events |

Returns `ta` and `ra` (how many advisories now) and `events`, each with `at`, `callsign`, `intruder`, `advisory` (`TA`, `RA` or `clear`), `aural`, `rangeNM` and `dzFt`.

---

## set_player_clearance

Tells the engine what the user's own ATC cleared the user aircraft to do. The engine never controls or calls the user aircraft. While the user lines up, takes off or lands on a runway, none of our traffic is cleared onto it. Landing, the user is in that runway's landing sequence and our traffic fits around it. `vacated` ends it.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `icao` | string | Yes | The airport |
| `runway` | string | Yes | The runway end, e.g. `"24"` |
| `phase` | string | Yes | `pushback`, `taxi`, `holding_short`, `lineup`, `takeoff`, `landing` or `vacated` |
| `callsign` | string | No | The user's call sign as said (default `Player`) |
| `model` | string | No | The user's model, for its wake and speed |

Returns the `clearance` and, landing, the user's `place` (number, leader, spacing, distances to go).

---

## Scheduled traffic tools

`start_schedule` runs the engine's scheduled traffic at airports: an airline timetable (the same schedule as `generate_schedule`) and light aircraft in the circuit by day.

- **Departures** appear on their stand before their STD, get their stand services and push on time.
- **Arrivals** come in en route and join their STAR.
- **Turnarounds and overflights** are flown too.
- **ATC:** the engine's tower, sequences and separation clear them.
- **Removal:** departed and parked aircraft are removed.

`add_flights` adds flights at chosen times, `set_real_traffic` and `observe_traffic` fly real-world aircraft instead of the timetable, and `set_traffic_corridor` keeps airliners around the user's flight in cruise.

## start_schedule

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `airports` | string | Yes | | Airports, e.g. `"LKPR"` or `"LKPR, LKTB"`; loaded around the user aircraft |
| `density` | number | No | `1` | Traffic density (1: the timetable as it is) |
| `max_aircraft` | number | No | the engine's | Most of the schedule's aircraft at once |
| `seed` | number | No | | Random seed |
| `ifr` | boolean | No | `true` | Airline flights |
| `vfr` | boolean | No | `true` | Light aircraft in the circuit |
| `generator` | boolean | No | `true` | The generated timetable; `false`: only flights from `add_flights` |
| `offset_min` | number | No | `0` | Fly the timetable this many minutes later now (600 puts a morning wave into an evening) |
| `others` | string | No | `respect` | `respect` or `ignore` the traffic that is not ours |

Calling it again changes the settings. Returns the schedule's settings, `now` (traffic time), `active` and `flight_count`.

## stop_schedule

No more aircraft appear; those flying finish their flights. With `remove: true` every aircraft of ours on the engine's list is taken out now. Returns `running` and `removed`.

## get_schedule

The schedule's settings and `now`, and per airport (`icao`, default every scheduled airport) its `boards`: `departures` and `arrivals`. Each flight has `callsign`, `airline`, `type`, `origin`, `destination`, `std`, `sta`, `status`, `stand`, `runway`, `estimated`, `note` and, for a real one, `observed`.

## add_flights

Adds flights to the running schedule at chosen times: an arrival just before the user's ETA, say, or a departure just after their off-block. `start_schedule` with `generator: false` runs only these.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `flights` | array | Yes | Flight objects: `callsign`, `origin`, `destination` (one of them a scheduled airport), `type` (default A320), `airline`, and `std_in_min` / `sta_in_min` (minutes from traffic time now) or `std` / `sta` (RFC 3339, traffic time) |

A departure needs an STD, an arrival an STA. An STA too soon (inside 15 minutes with the defaults) is refused with `NOT_APPLICABLE`. Returns `now` and the flights `added`.

## set_real_traffic

Flies real-world traffic at an airport instead of the generated timetable. Generated flights not yet in the simulator go at once, those flying finish. `on: false` brings the timetable back.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `on` | boolean | No | `true` | On or off |
| `icao` | string | When on | | The airport |

## observe_traffic

Feeds real-world sightings to the engine, from a feed such as ADS-B. What happens to each:

- **IDs:** each is identified by its ICAO 24-bit address and spawned once; later sightings only refresh its registration, origin and destination.
- **Parked and departing** aircraft go on a stand within 80 m of where they are seen. A parked one waits, with no push, until its departure is seen.
- **Arrivals** appear on their projected track and join a STAR of the runway in use.
- **Overflights** are not flown.
- **Unknown** origins and destinations stay unknown.

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sightings` | array | No | Objects: `id`, `callsign`, `registration`, `type`, `lat`, `lon`, `altFt`, `groundKts`, `trackDeg`, `vsFpm`, `onGround`, `seenAt` (default now), and optional `kind`, `origin`, `destination`, `departAt` |
| `drop` | string | No | IDs the feed no longer sees, e.g. `"49d2a1, 4ca7b2"`; one in progress finishes first |

Returns `results`, one per sighting: `id`, `kind`, `callsign`, `airport`, `status` (`added`, `updated`, `retimed`, `turnaround`, `ignored`) and `reason`. It also returns `dropped`.

## set_traffic_corridor

Keeps a few airliners around the user's flight in cruise, flown by MSFS AI:

- **same:** ahead on the route, 25–45 NM, same direction, 2000 ft above or below.
- **opposite:** 70–100 NM ahead, coming the other way, 1000 ft above or below.
- **crossing:** 45–70 NM ahead, across the route at 60–120°, 1000 or 2000 ft above or below.

One more than `despawn_nm` from the user aircraft and moving away is replaced.

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `enabled` | boolean | No | `true` | On, or off (takes them all out) |
| `route` | string | When on | | The user's route ahead, in its direction: `"lat,lon; lat,lon; …"`, at least two points |
| `level_ft` | number | When on | | The user's cruise level, feet |
| `kts` | number | No | | The user's cruise speed |
| `same`, `opposite`, `crossing` | number | No | `1` each | How many of each |
| `despawn_nm` | number | No | `80` | Replace one this far away |

Returns the corridor and its `aircraft`.

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

---

