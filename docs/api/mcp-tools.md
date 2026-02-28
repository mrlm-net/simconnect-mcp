# MCP Tool API Reference

This document describes all 11 MCP tools exposed by the SimConnect MCP server when running in `docs` mode (`MCP_MODE=docs`). These tools provide read-only access to the scraped SimConnect SDK documentation corpus: simulation variables, client events, API functions, data structures, and exception/error codes.

All tools are called over the Model Context Protocol using JSON-RPC 2.0 with the `tools/call` method. Error responses are returned as text content (not JSON-RPC errors) with a prefix token (`NOT_FOUND`, `INVALID_ARGUMENT`, `INVALID_PAGE`, `INTERNAL_ERROR`) followed by a colon and a human-readable message.

## Tool Overview

| Tool | Category | Description |
|------|----------|-------------|
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

---

## Pagination

All `list_*` tools return a `Page[T]` envelope with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | Records for the current page |
| `page` | integer | Current page number (1-indexed) |
| `page_size` | integer | Maximum items per page as requested |
| `total_items` | integer | Total record count across all pages |
| `total_pages` | integer | Total page count (`ceil(total_items / page_size)`); `0` when `total_items` is `0` |

---

## SimVars

### list_simvars

**Description**: List SimConnect simulation variables, optionally filtered by category, with pagination.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `category` | string | No | — | Filter results to a specific SDK documentation category (e.g., `"Aircraft Position and Speed"`). Omit to return all categories. |
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**: A `Page[SimVar]` object. Each `SimVar` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Canonical SimVar name (e.g., `"PLANE ALTITUDE"`) |
| `description` | string | Full SDK description |
| `units` | string[] | Valid unit strings (e.g., `["feet", "meters"]`) |
| `settable` | boolean | Whether `SimConnect_SetDataOnSimObject` can write this variable |
| `category` | string | SDK documentation grouping |
| `versions` | string[] | Simulator versions that define this variable (`"2020"`, `"2024"`) |
| `indexed_by` | string | Describes the index suffix when variable is indexed (e.g., `"1-indexed engine number"`); omitted when not indexed |
| `deprecated` | boolean | `true` when the SDK marks this variable deprecated; omitted when `false` |
| `deprecated_reason` | string | Reason or replacement variable when deprecated; omitted otherwise |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

**Example request**:

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

**Example response**:

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

**Error codes**:

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_simvar

**Description**: Fetch a single SimConnect simulation variable by name (case-insensitive).

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | SimVar name to look up. Case-insensitive. Example: `"PLANE ALTITUDE"`. |

**Returns**: A single `SimVar` object. See [`list_simvars`](#list_simvars) for field descriptions.

**Example request**:

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

**Example response**:

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

**Error codes**:

- `NOT_FOUND`: No SimVar with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Events

### list_events

**Description**: List SimConnect client input events (Key Event IDs) with pagination.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**: A `Page[Event]` object. Each `Event` has:

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

**Example request**:

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

**Example response**:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"items\":[{\"name\":\"BRAKES\",\"description\":\"Apply or release the aircraft brakes.\",\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Misc_Events.htm\"},{\"name\":\"THROTTLE_SET\",\"description\":\"Set throttle to a specific position.\",\"parameters\":[{\"index\":0,\"name\":\"Value\",\"description\":\"Throttle position (0–16383).\",\"type\":\"DWORD\"}],\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Engine_Events.htm\"}],\"page\":1,\"page_size\":20,\"total_items\":2,\"total_pages\":1}"
      }
    ]
  }
}
```

**Error codes**:

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_event

**Description**: Fetch a single SimConnect client input event by name (case-insensitive).

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Event name to look up. Case-insensitive. Example: `"BRAKES"`. |

**Returns**: A single `Event` object. See [`list_events`](#list_events) for field descriptions.

**Example request**:

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

**Example response**:

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"name\":\"THROTTLE_SET\",\"description\":\"Set throttle to a specific position.\",\"parameters\":[{\"index\":0,\"name\":\"Value\",\"description\":\"Throttle position (0–16383).\",\"type\":\"DWORD\"}],\"versions\":[\"2020\",\"2024\"],\"source_url\":\"https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Engine_Events.htm\"}"
      }
    ]
  }
}
```

**Error codes**:

- `NOT_FOUND`: No event with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Functions

### list_functions

**Description**: List SimConnect C API functions with pagination.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**: A `Page[Function]` object. Each `Function` has:

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

**Example request**:

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

**Example response**:

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

**Error codes**:

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_function

**Description**: Fetch a single SimConnect API function by name (case-insensitive).

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Function name to look up. Case-insensitive. Example: `"SimConnect_Open"`. |

**Returns**: A single `Function` object. See [`list_functions`](#list_functions) for field descriptions.

**Example request**:

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

**Example response**:

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

**Error codes**:

- `NOT_FOUND`: No function with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Structures

### list_structures

**Description**: List SimConnect C data structures with pagination.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**: A `Page[Structure]` object. Each `Structure` has:

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

**Example request**:

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

**Example response**:

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

**Error codes**:

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_structure

**Description**: Fetch a single SimConnect data structure by name (case-insensitive).

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Yes | — | Structure name to look up. Case-insensitive. Example: `"SIMCONNECT_DATA_INITPOSITION"`. |

**Returns**: A single `Structure` object. See [`list_structures`](#list_structures) for field descriptions.

**Example request**:

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

**Example response**:

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

**Error codes**:

- `NOT_FOUND`: No structure with the given name exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Error Codes

### list_error_codes

**Description**: List `SIMCONNECT_EXCEPTION` enumeration values with pagination.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `page` | integer | No | `1` | Page number, 1-indexed. |
| `page_size` | integer | No | `20` | Results per page. Maximum `100`. |

**Returns**: A `Page[ErrorCode]` object. Each `ErrorCode` has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Enum member name (e.g., `"SIMCONNECT_EXCEPTION_NONE"`) |
| `value` | integer | Integer ordinal of the enum member |
| `description` | string | Full SDK description |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL from which this record was scraped |

**Example request**:

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

**Example response**:

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

**Error codes**:

- `INVALID_PAGE`: `page` is less than `1`, or `page_size` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.

---

### get_error_code

**Description**: Fetch a single `SIMCONNECT_EXCEPTION` enum value by name or by integer value; at least one of `name` or `value` must be provided.

When both `name` and `value` are provided, name lookup is attempted first. If name lookup fails, the tool falls back to scanning by integer value.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | string | Conditional | — | Enum member name (e.g., `"SIMCONNECT_EXCEPTION_NONE"`). Required when `value` is not provided. |
| `value` | integer | Conditional | — | Integer ordinal of the enum member (e.g., `0`). Required when `name` is not provided. |

> **Note**: At least one of `name` or `value` must be supplied. Providing neither returns an `INVALID_ARGUMENT` error. Providing both causes a name lookup first, with value as fallback.

**Returns**: A single `ErrorCode` object. See [`list_error_codes`](#list_error_codes) for field descriptions.

**Example request (by name)**:

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

**Example request (by value)**:

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

**Example response**:

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

**Error codes**:

- `INVALID_ARGUMENT`: Neither `name` nor `value` was provided.
- `NOT_FOUND`: No error code matching the given name or value exists in the corpus.
- `INTERNAL_ERROR`: Unexpected store failure.

---

## Search

### search_docs

**Description**: Full-text search across the SimConnect documentation corpus. Returns matching SimVars, events, functions, structures, and error codes with ranked excerpts.

**Arguments**:

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `query` | string | Yes | — | Search query string. Must be non-empty after trimming whitespace. |
| `type` | string | No | `"all"` | Corpus type to search. One of: `simvar`, `event`, `function`, `structure`, `error_code`, `all`. |
| `limit` | integer | No | `20` | Maximum number of results to return. Minimum `1`, maximum `100`. |

**Returns**: A `SearchResults` object with the following fields:

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
| `excerpt` | string | Short snippet from the item's description providing query context (typically 120-160 characters, trimmed at a word boundary) |
| `source_url` | string | Canonical `docs.flightsimulator.com` URL for this item |

**Example request**:

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

**Example response**:

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

**Error codes**:

- `INVALID_ARGUMENT`: `query` is empty or whitespace-only, `type` is not one of the allowed values, or `limit` is less than `1` or greater than `100`.
- `INTERNAL_ERROR`: Unexpected store failure.
