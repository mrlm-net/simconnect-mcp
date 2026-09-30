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

Claude calls `start_schedule`. The server builds the day's schedule for LKPR and, on its own, keeps it running:

- departures appear on a free stand 10 minutes before their STD and push back at it;
- arrivals appear in the air on their flight plan and are handed to the arrival controller at their STAR entry;
- overflights cross the area at cruise level;
- the tower clears them to cross, line up and take off, and the landing sequence spaces the arrivals.

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
  "running": true,
  "airports": ["LKPR"],
  "active": 3,
  "max_aircraft": 3,
  "boards": [
    {
      "icao": "LKPR",
      "departures": [
        { "callsign": "WZZ794", "type": "A321", "from": "LKPR", "to": "LHBP", "scheduled": "09:30Z", "status": "taxiing", "stand": "A1", "runway": "06" },
        { "callsign": "SWR643", "type": "A20N", "from": "LKPR", "to": "LSGG", "scheduled": "09:40Z", "status": "boarding", "stand": "A1", "runway": "06" }
      ],
      "arrivals": [
        { "callsign": "AFR1094", "type": "A321", "from": "LFPG", "to": "LKPR", "scheduled": "09:55Z", "status": "approaching", "stand": "N50", "runway": "06", "note": "landing sequence" }
      ]
    }
  ]
}
```

**`get_atc_log` (messages)**

```text
WZZ794, cross runway 12/30
AFR1094, number 2, delay 34s: 210 kt
SWR643, cross runway 12/30
WZZ794, runway 06, line up and wait
WZZ794, runway 06, cleared for take-off
```

**Claude's reply**

> "The LKPR schedule is running. Wizz Air 794 to Budapest is taxiing to runway 06 and has just been cleared for take-off. Swiss 643 to Geneva is taxiing behind it. Air France 1094 from Paris is number 2 for runway 06 and has been slowed to 210 knots to absorb a 34-second delay."

Taken from a live run in MSFS 2024. `stop_schedule` with `remove: true` takes the schedule's aircraft out of the simulator.
