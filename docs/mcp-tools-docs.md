---
title: "MCP Tools — Docs Mode"
description: Reference for all 15 MCP tools available in docs mode (MCP_MODE=docs).
order: 1
section: reference
---

All 15 MCP tools listed here are available when the server runs with `MCP_MODE=docs` (and also with `MCP_MODE=both`). Twelve of them provide read-only access to the scraped SimConnect SDK documentation corpus: simulation variables, client events, API functions, data structures, and exception/error codes. The other three serve the guides of the [`github.com/mrlm-net/simconnect`](https://github.com/mrlm-net/simconnect) Go library (currently v0.23.1), embedded at the library version the server is built against. The server is cross-platform in this mode — no simulator installation is required.

Tools are called over the Model Context Protocol using JSON-RPC 2.0 with the `tools/call` method. Error responses are returned as text content (not JSON-RPC errors) with a prefix token followed by a colon and a human-readable message.

## Tool Overview

| Tool | Category | Description |
|------|----------|-------------|
| [`list_simvar_categories`](#list_simvar_categories) | SimVars | List all SimVar category filter values for `list_simvars` |
| [`list_simvars`](#list_simvars) | SimVars | List simulation variables, optionally filtered by category |
| [`get_simvar`](#get_simvar) | SimVars | Fetch a single simulation variable by name |
| [`list_events`](#list_events) | Events | List client input events |
| [`get_event`](#get_event) | Events | Fetch a single client event by name |
| [`list_functions`](#list_functions) | Functions | List SDK C API functions |
| [`get_function`](#get_function) | Functions | Fetch a single SDK function by name |
| [`list_structures`](#list_structures) | Structures | List SDK C data structures |
| [`get_structure`](#get_structure) | Structures | Fetch a single data structure by name |
| [`list_error_codes`](#list_error_codes) | Error Codes | List `SIMCONNECT_EXCEPTION` enum values |
| [`get_error_code`](#get_error_code) | Error Codes | Fetch an error code by name or integer value |
| [`search_docs`](#search_docs) | Search | Full-text search across all corpus types |
| [`list_library_guides`](#list_library_guides) | Library Guides | List the `mrlm-net/simconnect` Go library guides and their chapters |
| [`get_library_guide`](#get_library_guide) | Library Guides | Read a library guide, or one chapter of it, as Markdown |
| [`search_library_docs`](#search_library_docs) | Library Guides | Search the library guides chapter by chapter |

---

## Pagination

All paginated `list_*` tools (every `list_*` tool except `list_simvar_categories` and `list_library_guides`, which return their full result in one response) return a `Page[T]` envelope with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | Records for the current page |
| `page` | integer | Current page number (1-indexed) |
| `page_size` | integer | Maximum items per page as requested |
| `total_items` | integer | Total record count across all pages |
| `total_pages` | integer | Total page count (`ceil(total_items / page_size)`); `0` when `total_items` is `0` |

---

## SimVars

### list_simvar_categories

List all SimVar category filter values. Use these exact strings as the `category` parameter of [`list_simvars`](#list_simvars). Note that some categories contain `/` without spaces (e.g. `"AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES"`) while others use ` / ` with spaces (e.g. `"AIRCRAFT BRAKE / LANDING GEAR VARIABLES"`) — copy them verbatim.

**Parameters**

None.

**Returns**

An object with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `total` | integer | Number of categories |
| `categories` | string[] | Category names, as accepted by `list_simvars` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "method": "tools/call",
  "params": {
    "name": "list_simvar_categories",
    "arguments": {}
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"total\":2,\"categories\":[\"AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES\",\"AIRCRAFT BRAKE / LANDING GEAR VARIABLES\"]}"
      }
    ]
  }
}
```

**Error codes**

- `INTERNAL_ERROR`: Unexpected store failure.

---

### list_simvars

List SimConnect simulation variables, optionally filtered by category, with pagination.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `category` | string | No | — | Filter results to a specific SDK documentation category (e.g., `"Aircraft Position and Speed"`). Use [`list_simvar_categories`](#list_simvar_categories) for the exact values. Omit to return all categories. |
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**

A `Page[SimVar]` object. Each `SimVar` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Canonical SimVar name (e.g., `"PLANE ALTITUDE"`) |
| `description` | string | Full SDK description |
| `units` | string[] | Valid unit strings (e.g., `["feet", "meters"]`) |
| `settable` | boolean | Whether `SimConnect_SetDataOnSimObject` can write this variable |
| `category` | string | SDK documentation grouping |
| `versions` | string[] | Simulator versions that define this variable (`"2020"`, `"2024"`) |
| `indexed_by` | string | Describes the index suffix when the variable is indexed (e.g., `"1-indexed engine number"`); omitted when not indexed |
| `deprecated` | boolean | `true` when the SDK marks this variable deprecated; omitted when `false` |
| `deprecated_reason` | string | Reason or replacement variable when deprecated; omitted otherwise |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "list_simvars",
    "arguments": {
      "category": "Aircraft Position and Speed",
      "page": 1,
      "page_size": 5
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"items\":[{\"name\":\"PLANE ALTITUDE\",\"description\":\"The altitude of the aircraft above mean sea level.\",\"units\":[\"feet\",\"meters\"],\"settable\":false,\"category\":\"Aircraft Position and Speed\",\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm\"},{\"name\":\"GROUND ALTITUDE\",\"description\":\"Altitude of the ground directly below the aircraft.\",\"units\":[\"feet\",\"meters\"],\"settable\":false,\"category\":\"Aircraft Position and Speed\",\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm\"}],\"page\":1,\"page_size\":5,\"total_items\":2,\"total_pages\":1}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_simvar

Fetch a single SimConnect simulation variable by name (case-insensitive).

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | SimVar name to look up. Case-insensitive. Example: `"PLANE ALTITUDE"`. |

**Returns**

A single `SimVar` object. See [list_simvars](#list_simvars) for field descriptions.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_simvar",
    "arguments": {
      "name": "PLANE ALTITUDE"
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"name\":\"PLANE ALTITUDE\",\"description\":\"The altitude of the aircraft above mean sea level.\",\"units\":[\"feet\",\"meters\"],\"settable\":false,\"category\":\"Aircraft Position and Speed\",\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm\"}"
      }
    ]
  }
}
```

**Error codes**

- `NOT_FOUND`: No SimVar with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Events

### list_events

List SimConnect client input events (Key Event IDs) with pagination.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**

A `Page[Event]` object. Each `Event` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Canonical event name (e.g., `"BRAKES"`) |
| `description` | string | Full SDK description |
| `parameters` | EventParam[] | Positional parameters accepted by this event; omitted when the event takes no parameters |
| `versions` | string[] | Simulator versions that define this event (`"2020"`, `"2024"`) |
| `deprecated` | boolean | `true` when the SDK marks this event superseded; omitted when `false` |
| `deprecated_reason` | string | Reason or replacement event when deprecated; omitted otherwise |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

Each `EventParam` has:

| Field | Type | Description |
|-------|------|-------------|
| `index` | integer | 0-based positional index |
| `name` | string | Parameter name |
| `description` | string | Full parameter description |
| `type` | string | C data type (e.g., `"DWORD"`, `"float"`) |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "list_events",
    "arguments": {
      "page": 1,
      "page_size": 20
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"items\":[{\"name\":\"BRAKES\",\"description\":\"Apply or release the aircraft brakes.\",\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Misc_Events.htm\"},{\"name\":\"THROTTLE_SET\",\"description\":\"Set throttle to a specific position.\",\"parameters\":[{\"index\":0,\"name\":\"Value\",\"description\":\"Throttle position (0\u201316383).\",\"type\":\"DWORD\"}],\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Engine_Events.htm\"}],\"page\":1,\"page_size\":20,\"total_items\":2,\"total_pages\":1}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_event

Fetch a single SimConnect client input event by name (case-insensitive).

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Event name to look up. Case-insensitive. Example: `"BRAKES"`. |

**Returns**

A single `Event` object. See [list_events](#list_events) for field descriptions.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "get_event",
    "arguments": {
      "name": "THROTTLE_SET"
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"name\":\"THROTTLE_SET\",\"description\":\"Set throttle to a specific position.\",\"parameters\":[{\"index\":0,\"name\":\"Value\",\"description\":\"Throttle position (0\u201316383).\",\"type\":\"DWORD\"}],\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Engine_Events.htm\"}"
      }
    ]
  }
}
```

**Error codes**

- `NOT_FOUND`: No event with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Functions

### list_functions

List SimConnect C API functions with pagination.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**

A `Page[Function]` object. Each `Function` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | C function name (e.g., `"SimConnect_Open"`) |
| `description` | string | Full SDK description |
| `signature` | string | Complete C function signature string |
| `parameters` | FunctionParam[] | Ordered parameter list (always present, may be empty) |
| `return_type` | string | C return type (e.g., `"HRESULT"`) |
| `remarks` | string | Additional SDK usage notes; omitted when empty |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

Each `FunctionParam` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | C parameter name |
| `type` | string | C data type (e.g., `"HANDLE *"`, `"DWORD"`, `"const char *"`) |
| `direction` | string | Data flow relative to caller: `"in"`, `"out"`, or `"in/out"` |
| `description` | string | Full parameter description |
| `optional` | boolean | `true` when the SDK documents this as an optional trailing parameter; omitted when `false` |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "list_functions",
    "arguments": {
      "page": 1,
      "page_size": 20
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
        "text": "{\"items\":[{\"name\":\"SimConnect_Open\",\"description\":\"Opens a new connection with the SimConnect server.\",\"signature\":\"SIMCONNECTAPI SimConnect_Open(HANDLE * phSimConnect, LPCSTR szName, HWND hWnd, DWORD UserEventWin32, HANDLE hEventHandle, DWORD ConfigIndex)\",\"parameters\":[{\"name\":\"phSimConnect\",\"type\":\"HANDLE *\",\"direction\":\"out\",\"description\":\"Pointer to a handle to receive the new connection.\"},{\"name\":\"szName\",\"type\":\"LPCSTR\",\"direction\":\"in\",\"description\":\"Application name string.\"}],\"return_type\":\"HRESULT\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/General/SimConnect_Open.htm\"},{\"name\":\"SimConnect_Close\",\"description\":\"Closes the connection to the SimConnect server.\",\"signature\":\"SIMCONNECTAPI SimConnect_Close(HANDLE hSimConnect)\",\"parameters\":[{\"name\":\"hSimConnect\",\"type\":\"HANDLE\",\"direction\":\"in\",\"description\":\"The handle returned by SimConnect_Open.\"}],\"return_type\":\"HRESULT\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/General/SimConnect_Close.htm\"}],\"page\":1,\"page_size\":20,\"total_items\":2,\"total_pages\":1}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_function

Fetch a single SimConnect API function by name (case-insensitive).

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Function name to look up. Case-insensitive. Example: `"SimConnect_Open"`. |

**Returns**

A single `Function` object. See [list_functions](#list_functions) for field descriptions.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "get_function",
    "arguments": {
      "name": "SimConnect_Open"
    }
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
        "text": "{\"name\":\"SimConnect_Open\",\"description\":\"Opens a new connection with the SimConnect server.\",\"signature\":\"SIMCONNECTAPI SimConnect_Open(HANDLE * phSimConnect, LPCSTR szName, HWND hWnd, DWORD UserEventWin32, HANDLE hEventHandle, DWORD ConfigIndex)\",\"parameters\":[{\"name\":\"phSimConnect\",\"type\":\"HANDLE *\",\"direction\":\"out\",\"description\":\"Pointer to a handle to receive the new connection.\"},{\"name\":\"szName\",\"type\":\"LPCSTR\",\"direction\":\"in\",\"description\":\"Application name string.\"}],\"return_type\":\"HRESULT\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/General/SimConnect_Open.htm\"}"
      }
    ]
  }
}
```

**Error codes**

- `NOT_FOUND`: No function with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Structures

### list_structures

List SimConnect C data structures with pagination.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**

A `Page[Structure]` object. Each `Structure` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | C struct or union name (e.g., `"SIMCONNECT_RECV"`) |
| `description` | string | Full SDK description |
| `fields` | StructField[] | Member fields (always present, may be empty) |
| `remarks` | string | Additional SDK usage notes; omitted when empty |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

Each `StructField` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | C field name |
| `type` | string | C data type (e.g., `"DWORD"`, `"double"`, `"char[256]"`) |
| `description` | string | Full field description |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "list_structures",
    "arguments": {
      "page": 1,
      "page_size": 20
    }
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
        "text": "{\"items\":[{\"name\":\"SIMCONNECT_RECV\",\"description\":\"Base structure for all data received from SimConnect.\",\"fields\":[{\"name\":\"dwSize\",\"type\":\"DWORD\",\"description\":\"Size of the structure in bytes.\"},{\"name\":\"dwVersion\",\"type\":\"DWORD\",\"description\":\"SimConnect version number.\"},{\"name\":\"dwID\",\"type\":\"DWORD\",\"description\":\"ID identifying the type of the received data packet.\"}],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV.htm\"},{\"name\":\"SIMCONNECT_DATA_INITPOSITION\",\"description\":\"Defines an aircraft initial position and state.\",\"fields\":[{\"name\":\"Latitude\",\"type\":\"double\",\"description\":\"Latitude in degrees.\"},{\"name\":\"Longitude\",\"type\":\"double\",\"description\":\"Longitude in degrees.\"},{\"name\":\"Altitude\",\"type\":\"double\",\"description\":\"Altitude in feet.\"}],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_INITPOSITION.htm\"}],\"page\":1,\"page_size\":20,\"total_items\":2,\"total_pages\":1}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_structure

Fetch a single SimConnect data structure by name (case-insensitive).

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Structure name to look up. Case-insensitive. Example: `"SIMCONNECT_DATA_INITPOSITION"`. |

**Returns**

A single `Structure` object. See [list_structures](#list_structures) for field descriptions.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "tools/call",
  "params": {
    "name": "get_structure",
    "arguments": {
      "name": "SIMCONNECT_DATA_INITPOSITION"
    }
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
        "text": "{\"name\":\"SIMCONNECT_DATA_INITPOSITION\",\"description\":\"Defines an aircraft initial position and state.\",\"fields\":[{\"name\":\"Latitude\",\"type\":\"double\",\"description\":\"Latitude in degrees.\"},{\"name\":\"Longitude\",\"type\":\"double\",\"description\":\"Longitude in degrees.\"},{\"name\":\"Altitude\",\"type\":\"double\",\"description\":\"Altitude in feet.\"}],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_INITPOSITION.htm\"}"
      }
    ]
  }
}
```

**Error codes**

- `NOT_FOUND`: No structure with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Error Codes

### list_error_codes

List `SIMCONNECT_EXCEPTION` enumeration values with pagination.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**

A `Page[ErrorCode]` object. Each `ErrorCode` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Enum member name (e.g., `"SIMCONNECT_EXCEPTION_NONE"`) |
| `value` | integer | Integer ordinal of the enum member |
| `description` | string | Full SDK description |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "method": "tools/call",
  "params": {
    "name": "list_error_codes",
    "arguments": {
      "page": 1,
      "page_size": 20
    }
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
        "text": "{\"items\":[{\"name\":\"SIMCONNECT_EXCEPTION_NONE\",\"value\":0,\"description\":\"No error.\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_EXCEPTION.htm\"},{\"name\":\"SIMCONNECT_EXCEPTION_ERROR\",\"value\":1,\"description\":\"Unspecified error.\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_EXCEPTION.htm\"}],\"page\":1,\"page_size\":20,\"total_items\":2,\"total_pages\":1}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_error_code

Fetch a single `SIMCONNECT_EXCEPTION` enum value by name or by integer value. At least one of `name` or `value` must be provided.

When both `name` and `value` are provided, name lookup is attempted first. If name lookup fails, the tool falls back to scanning by integer value.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Conditional | — | Enum member name (e.g., `"SIMCONNECT_EXCEPTION_NONE"`). Required when `value` is not provided. |
| `value` | integer | Conditional | — | Integer ordinal of the enum member (e.g., `0`). Required when `name` is not provided. |

> At least one of `name` or `value` must be supplied. Providing neither returns an `INVALID_ARGUMENT` error. Providing both causes a name lookup first, with value as fallback.

**Returns**

A single `ErrorCode` object. See [list_error_codes](#list_error_codes) for field descriptions.

**Example request (by name)**

```json
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "get_error_code",
    "arguments": {
      "name": "SIMCONNECT_EXCEPTION_NONE"
    }
  }
}
```

**Example request (by value)**

```json
{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "get_error_code",
    "arguments": {
      "value": 1
    }
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
        "text": "{\"name\":\"SIMCONNECT_EXCEPTION_NONE\",\"value\":0,\"description\":\"No error.\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_EXCEPTION.htm\"}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_ARGUMENT`: Neither `name` nor `value` was provided.
- `NOT_FOUND`: No error code matching the given name or value exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Search

### search_docs

Full-text search across the SimConnect documentation corpus. Returns matching SimVars, events, functions, structures, and error codes with ranked excerpts.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | Yes | — | Search query string. Must be non-empty after trimming whitespace. |
| `type` | string | No | `"all"` | Corpus type to search. One of: `simvar`, `event`, `function`, `structure`, `error_code`, `all`. |
| `limit` | integer | No | `20` | Maximum number of results to return. Minimum `1`, maximum `100`. |

**Returns**

A `SearchResults` object with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | The search string as provided by the caller |
| `total` | integer | Number of results in this response (equal to `len(results)`; not a global match count) |
| `results` | SearchResult[] | Ordered search hits, most relevant first |

Each `SearchResult` has:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Corpus category of this hit: `simvar`, `event`, `function`, `structure`, or `errorcode` |
| `name` | string | Canonical SDK name of the matched item |
| `excerpt` | string | Short snippet from the item's description providing query context (typically 120–160 characters, trimmed at a word boundary) |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL for this item |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "search_docs",
    "arguments": {
      "query": "altitude",
      "type": "simvar",
      "limit": 5
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 12,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"query\":\"altitude\",\"total\":2,\"results\":[{\"type\":\"simvar\",\"name\":\"PLANE ALTITUDE\",\"excerpt\":\"The altitude of the aircraft above mean sea level.\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm\"},{\"type\":\"simvar\",\"name\":\"GROUND ALTITUDE\",\"excerpt\":\"Altitude of the ground directly below the aircraft.\",\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm\"}]}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_ARGUMENT`: `query` is empty or whitespace-only, `type` is not one of the allowed values, or `limit` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.


---

## Library Guides

These three tools serve the guides of the [`github.com/mrlm-net/simconnect`](https://github.com/mrlm-net/simconnect) Go library — the SimConnect client and manager, facilities, input events, client data areas, `pkg/airport` (ground layouts, taxi routing, SID/STAR/approach procedures), `pkg/nav` (airways, weather, active runway, ATIS, flight plans), `pkg/traffic` (AI traffic, departures, arrivals, schedules, sequencing, separation, radio), `pkg/camera` (add-on camera), `pkg/systems` (aircraft systems profiles), `pkg/avionics` (radios and transponder) and `pkg/addons` (installed add-ons). The 45 guides are embedded in the server binary at the library version `go.mod` requires (currently v0.23.1), so they always match the library the server is built with. In docs and both modes the `/health` endpoint reports that version as `library_version`.

Each guide is split into chapters at its `##` headings. The usual flow is `search_library_docs` or `list_library_guides` to find a guide, then `get_library_guide` with a `chapter` to read only the relevant part — whole guides can be long.

### list_library_guides

List the library guides, optionally filtered by section, with the chapter headings of each guide.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `section` | string | No | — | Filter to one section: `airport`, `client`, `datasets`, `events`, `internals`, `manager`, `nav`, `packages`, `traffic` (case-insensitive). Omit to list all guides. |

**Returns**

An object with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `library` | string | Always `"github.com/mrlm-net/simconnect"` |
| `version` | string | Library version the guides were taken from (e.g., `"v0.18.4"`) |
| `total` | integer | Number of guides returned |
| `guides` | Guide[] | Guides, ordered by section and position within the section |

Each `Guide` has:

| Field | Type | Description |
|-------|------|-------------|
| `slug` | string | Guide identifier, passed to `get_library_guide` (e.g., `"airport-layout"`) |
| `title` | string | Guide title |
| `description` | string | One-line summary; omitted when the guide has none |
| `section` | string | Section the guide belongs to |
| `chapters` | string[] | The guide's `##` chapter headings, in order |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 13,
  "method": "tools/call",
  "params": {
    "name": "list_library_guides",
    "arguments": {
      "section": "airport"
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 13,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"library\":\"github.com/mrlm-net/simconnect\",\"version\":\"v0.18.4\",\"total\":1,\"guides\":[{\"slug\":\"airport-layout\",\"title\":\"Airport Layout & Taxi Routing\",\"description\":\"Load an airport's ground layout with pkg/airport and compute taxi routes between stands and runways.\",\"section\":\"airport\",\"chapters\":[\"Loading a layout\",\"The Layout model\",\"Taxi graph and routes\",\"GeoJSON\",\"Procedures: SIDs, STARs, approaches\",\"Airport limits\",\"Seeing it on a map\"]}]}"
      }
    ]
  }
}
```

**Error codes**

- `NOT_FOUND`: No guides in the given `section`. The message lists the valid sections.

---

### get_library_guide

Read a library guide as Markdown — the whole guide, or a single `##` chapter of it.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `slug` | string | Yes | — | Guide slug from `list_library_guides` or `search_library_docs` (e.g., `"airport-layout"`). Case-insensitive; a trailing `.md` is accepted. |
| `chapter` | string | No | — | Chapter heading to read instead of the whole guide (e.g., `"Taxi graph and routes"`). Case-insensitive; a prefix that matches exactly one chapter heading also works (e.g., `"Taxi graph"`). |

**Returns**

Plain Markdown text (not JSON). Without `chapter`: the guide title as a `#` heading followed by the full guide. With `chapter`: the guide title, the chapter's `##` heading, and that chapter's text.

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 14,
  "method": "tools/call",
  "params": {
    "name": "get_library_guide",
    "arguments": {
      "slug": "airport-layout",
      "chapter": "Taxi graph"
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 14,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "# Airport Layout & Taxi Routing\n\n## Taxi graph and routes\n\n..."
      }
    ]
  }
}
```

**Error codes**

- `NOT_FOUND`: No guide with the given `slug`, or the guide has no chapter matching `chapter` (no exact match and not a unique prefix). The chapter error message lists the guide's chapter headings.

---

### search_library_docs

Search the library guides chapter by chapter. Every word of the query must appear in the chapter text, the chapter heading, or the guide's title or description — order-independent and case-insensitive. Results are ranked best first: words found in a chapter heading weigh most, then words in the guide title, then occurrences in the text. Go identifiers and concepts work well as queries (e.g., `"RouteToRunway"`, `"holding pattern"`, `"wake separation"`, `"weather reader"`).

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | Yes | — | Search words. Must be non-empty after trimming whitespace. |
| `limit` | integer | No | `10` | Maximum number of results to return. Minimum `1`, maximum `50`. |

**Returns**

An object with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | The search string as provided by the caller |
| `total` | integer | Number of results in this response (equal to `len(results)`; not a global match count) |
| `results` | LibraryHit[] | Matching chapters, most relevant first |

Each `LibraryHit` has:

| Field | Type | Description |
|-------|------|-------------|
| `slug` | string | Slug of the guide containing the chapter |
| `title` | string | Guide title |
| `heading` | string | Chapter heading — pass it as `chapter` to `get_library_guide`; omitted for text before the first `##` heading |
| `excerpt` | string | About 240 characters of the chapter text around the first query word |

**Example request**

```json
{
  "jsonrpc": "2.0",
  "id": 15,
  "method": "tools/call",
  "params": {
    "name": "search_library_docs",
    "arguments": {
      "query": "taxi route",
      "limit": 3
    }
  }
}
```

**Example response**

```json
{
  "jsonrpc": "2.0",
  "id": 15,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"query\":\"taxi route\",\"total\":1,\"results\":[{\"slug\":\"airport-layout\",\"title\":\"Airport Layout & Taxi Routing\",\"heading\":\"Taxi graph and routes\",\"excerpt\":\"…\"}]}"
      }
    ]
  }
}
```

**Error codes**

- `INVALID_ARGUMENT`: `query` is empty or whitespace-only, or `limit` is less than `1` or greater than `50`.
