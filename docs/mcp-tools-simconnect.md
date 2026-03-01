---
title: "MCP Tools — SimConnect Mode"
description: Reference for the 4 live-data MCP tools in SimConnect mode (MCP_MODE=simconnect, Windows only).
order: 2
section: reference
---

All 4 MCP tools listed here are available when the server runs with `MCP_MODE=simconnect`. This mode provides live simulator data via the SimConnect SDK.

**Requirements**: Windows only. Microsoft Flight Simulator 2020 or 2024 must be running with SimConnect enabled before issuing any read or transmit calls. The `get_sim_state` tool is safe to call at any time regardless of connection state.

Tools are called over the Model Context Protocol using JSON-RPC 2.0 with the `tools/call` method. Error responses are returned as text content (not JSON-RPC errors) with a prefix token followed by a colon and a human-readable message.

## Tool Overview

| Tool | Description |
|------|-------------|
| [`get_simvar_value`](#get_simvar_value) | Read a single simulation variable value from the running simulator |
| [`get_simvar_values`](#get_simvar_values) | Read up to 20 simulation variables in one call |
| [`transmit_event`](#transmit_event) | Transmit a named SimConnect client event to the simulator |
| [`get_sim_state`](#get_sim_state) | Return a snapshot of current simulator connection state and flight status |

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
