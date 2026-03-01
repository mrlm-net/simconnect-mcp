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
