---
title: Examples
description: Worked examples showing how to use SimConnect MCP tools in Claude Desktop.
order: 3
section: integration
---

The following scenarios demonstrate how Claude Desktop uses SimConnect MCP tools to answer natural language questions about the flight simulator. Each example shows the user prompt, the underlying MCP tool call, the raw response, and the reply Claude produces.

All examples that read or write simulator state require SimConnect mode (`MCP_MODE=simconnect`) and an active simulator session.

---

## Scenario 1: Query Current Altitude

**User prompt in Claude Desktop**

> "What is the aircraft's current altitude?"

**How it works**

Claude identifies that the question asks for a live simulator variable and calls `get_simvar_value` with the `PLANE ALTITUDE` variable in `feet`.

**MCP tool call**

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

**Response**

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

**Claude's reply**

> "The aircraft is currently at 5,280 feet MSL."

---

## Scenario 2: Toggle Landing Lights

**User prompt in Claude Desktop**

> "Turn on the landing lights."

**How it works**

Claude maps the user's intent to a simulator event and calls `transmit_event` with the `LANDING_LIGHTS_TOGGLE` event name. SimConnect toggles the landing lights and confirms the event was dispatched.

**MCP tool call**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "transmit_event",
    "arguments": {
      "name": "LANDING_LIGHTS_TOGGLE"
    }
  }
}
```

**Response**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
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

**Claude's reply**

> "Landing lights toggled."

---

## Scenario 3: Check Simulator State

**User prompt in Claude Desktop**

> "Is the simulator connected?"

**How it works**

Claude calls `get_sim_state` to retrieve the current connection status and flight session details. The tool returns different payloads depending on whether SimConnect is connected.

**MCP tool call**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_sim_state",
    "arguments": {}
  }
}
```

**Response — simulator connected**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
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

**Response — simulator not connected**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
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

**Claude's reply (connected)**

> "The simulator is connected and running flight LAST.FLT. Sim time: 3,742.5 seconds."

**Claude's reply (not connected)**

> "The simulator is not currently connected. Start Microsoft Flight Simulator and ensure SimConnect is enabled, then try again."

---

## Scenario 4: Run Scheduled Traffic at an Airport

**User prompt in Claude Desktop**

> "Bring Prague to life — run the airline schedule at LKPR, but no more than three aircraft at once."

**How it works**

Claude calls `start_schedule`. The first traffic tool starts the server's traffic engine (the library's `pkg/traffic/world`), which builds the day's schedule for LKPR and keeps it running on its own:

- departures appear on a stand before their STD, get their stand services (a fuel truck, stairs and a GPU at a remote stand) and push back behind a tug;
- arrivals come in en route and join their STAR, sequenced by the approach controller;
- the tower clears runway crossings, line-ups and take-offs, and the crews talk to delivery, ground, tower and approach.

Claude follows the flights with `get_schedule` and `get_atc_log`, and ends the run with `stop_schedule`.

**MCP tool call**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "start_schedule",
    "arguments": { "airports": "LKPR", "max_aircraft": 3 }
  }
}
```

**Later: `get_schedule` (text content, shortened)**

```json
{
  "enabled": true,
  "airports": ["LKPR"],
  "maxAircraft": 3,
  "active": 3,
  "now": "2026-10-07T09:31:12Z",
  "boards": {
    "LKPR": {
      "departures": [
        { "callsign": "WZZ794", "airline": "WZZ", "type": "A321", "origin": "LKPR", "destination": "LHBP", "std": "2026-10-07T09:30:00Z", "kind": "departure", "status": "taxiing", "stand": "A1", "runway": "24" },
        { "callsign": "SWR643", "airline": "SWR", "type": "A20N", "origin": "LKPR", "destination": "LSGG", "std": "2026-10-07T09:40:00Z", "kind": "departure", "status": "boarding", "stand": "B14", "runway": "24" }
      ],
      "arrivals": [
        { "callsign": "AFR1094", "airline": "AFR", "type": "A321", "origin": "LFPG", "destination": "LKPR", "sta": "2026-10-07T09:55:00Z", "kind": "arrival", "status": "approaching", "stand": "C18", "runway": "24" }
      ]
    }
  }
}
```

**`get_atc_log` (transmissions' `position` and `text`)**

```text
ground    Wizz Air 794, pushback approved, facing north-west
tower     Wizz Air 794, runway 24, line up and wait
tower     Wizz Air 794, runway 24, cleared for take-off, wind calm
approach  Air France 1094, reduce speed to 210 knots
ground    Swiss 643, pushback approved, facing north-west
```

**Claude's reply**

> "The LKPR schedule is running. Wizz Air 794 to Budapest has just been cleared for take-off on runway 24. Swiss 643 to Geneva is boarding on B14 and will push next. Air France 1094 from Paris is on the approach and has been slowed to 210 knots for spacing."

An illustration of the shapes (field names as the engine returns them, values shortened). `stop_schedule` with `remove: true` takes the schedule's aircraft out of the simulator.
