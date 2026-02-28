# Technical Architecture — Milestone 2: SimConnect MCP Live Data Mode

**Date:** 2026-02-28
**Status:** Approved
**Branch:** `milestone/2-simconnect`
**Module:** `github.com/mrlm-net/simconnect-mcp`

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [ADR-007: No-CGo — Pure Go FFI via mrlm-net/simconnect](#2-adr-007-no-cgo--pure-go-ffi-via-mrlm-netsimconnect)
3. [ADR-008: Bridge Interface and MockBridge for Cross-Platform Testing](#3-adr-008-bridge-interface-and-mockbridge-for-cross-platform-testing)
4. [ADR-009: Go 1.25 Toolchain Bump](#4-adr-009-go-125-toolchain-bump)
5. [Component Breakdown](#5-component-breakdown)
6. [Bridge Interface Definition](#6-bridge-interface-definition)
7. [MCP Tool Catalogue](#7-mcp-tool-catalogue)
8. [Configuration Contract](#8-configuration-contract)
9. [Error Handling Contract](#9-error-handling-contract)
10. [CI Matrix](#10-ci-matrix)

---

## 1. System Overview

The simconnect mode server connects to a running instance of Microsoft Flight Simulator
(MSFS 2020 or MSFS 2024), Prepar3D, or FSX via the SimConnect SDK. It exposes four MCP
tools that allow an AI agent to read live simulation variables, query top-level simulator
state, and transmit client events back to the simulator. A simulator connection is required
for three of the four tools; the fourth (`get_sim_state`) always succeeds, returning
`{connected: false}` when the simulator is unreachable.

The server runs on Windows only. The simconnect mode registration file and the real bridge
implementation carry `//go:build windows` tags. All other code — including the Bridge
interface, MockBridge, and all four MCP tool handlers — compiles cross-platform, which
enables Linux/macOS development and CI without a Windows environment.

### C4 Container Diagram

```
+-----------------------------------------------------------------------+
|  AI Agent (Claude Desktop, Cursor, custom MCP client)                 |
|  Transport: Streamable HTTP  POST /mcp  GET /mcp                      |
+-----------------------------------+-----------------------------------+
                                    |
                                    v
+-----------------------------------------------------------------------+
|  simconnect-mcp binary  (MCP_MODE=simconnect, Windows only)           |
|                                                                       |
|  +-------------------+   +---------------------------------------+    |
|  |  Gin HTTP layer   |   |  mcp-go MCPServer                     |    |
|  |  GET /health      |   |  mounted via gin.WrapH at /mcp        |    |
|  |  POST /mcp        |   |                                       |    |
|  |  GET  /mcp        |   |  get_simvar_value                     |    |
|  +-------------------+   |  get_simvar_values                    |    |
|                           |  transmit_event                       |    |
|                           |  get_sim_state                        |    |
|                           +-------------------+-------------------+    |
|                                               |                        |
|                           +------------------v------------------+      |
|                           |  Bridge interface                    |      |
|                           |  internal/bridge/bridge.go           |      |
|                           +--------+--------------------+--------+      |
|                                    |                    |               |
|            +-----------------------+     +--------------+----------+   |
|            |  simconnectBridge          |  MockBridge              |   |
|            |  (//go:build windows)      |  (no build tag)          |   |
|            |  wraps mrlm Manager        |  used in tests           |   |
|            +-----------+---------------+  --------------------------+  |
|                        |                                               |
+------------------------+-----------------------------------------------+
                          |
                          v  (Windows only)
              +-----------+------------+
              |  mrlm-net/simconnect   |
              |  Pure Go FFI layer     |
              |  (no CGo)              |
              +-----------+------------+
                          |
                          v  (DLL call via syscall)
              +-----------+------------+
              |  SimConnect.dll        |
              |  Microsoft Flight      |
              |  Simulator / P3D / FSX |
              +------------------------+
```

### Startup Sequence

```
main()
  |
  +--> Read MCP_MODE env var (= "simconnect")
  |
  +--> mode registry lookup -> simconnect.New(cfg)
  |      |
  |      +--> bridge.NewSimConnectBridge()   <- Windows-only real bridge
  |      |
  |      +--> simconnectMode{bridge, cfg}
  |
  +--> server.New()  <- Gin engine
  |
  +--> mode.Mount(router)
  |      |
  |      +--> bridge.Open(ctx, appName)      <- async, non-blocking
  |      |
  |      +--> tools.Register(mcpServer, bridge)
  |      |      |
  |      |      +--> registers 4 MCP tools
  |      |
  |      +--> GET /health -> healthHandler
  |      +--> Any /mcp   -> gin.WrapH(streamableHTTPServer)
  |      +--> GET /sse   -> gin.WrapH(sseServer)  [legacy compat]
  |      +--> POST /message -> gin.WrapH(sseServer)
  |
  +--> router.Run(cfg.ListenAddr)
```

---

## 2. ADR-007: No-CGo — Pure Go FFI via mrlm-net/simconnect

**Status:** Accepted

### Context

SimConnect.dll is a Windows COM-adjacent DLL that exposes a C-style flat API. Calling it
from Go requires crossing the language boundary. Three approaches were evaluated:

| Approach | Description | CGo required | External non-Go deps |
|---|---|---|---|
| Raw CGo with vendored stub headers | Write a `.c` adapter file and vendor `SimConnect.h`; compile with `cgo` | Yes | C compiler (MinGW or MSVC), SimConnect SDK headers |
| `mrlm-net/simconnect` pure Go FFI | Use `github.com/mrlm-net/simconnect`, which calls the DLL via Go's `syscall`/`golang.org/x/sys/windows` without CGo | No | None beyond `go get` |
| `github.com/joeshaw/go-purego` | Generic DLL loader; does not support the SimConnect calling convention | No | N/A |

### Decision

**Use `github.com/mrlm-net/simconnect` — zero CGo, zero external non-Go dependencies.**

### Rationale

The raw CGo approach requires a C compiler on every build machine (MSVC or MinGW), the
full SimConnect SDK installed at a fixed path, and a vendored copy of `SimConnect.h`. This
triples the CI setup surface: the Windows CI job must install the Windows SDK, configure
the CGo toolchain, set `CGO_ENABLED=1`, and point `CGO_CFLAGS`/`CGO_LDFLAGS` at the SDK
headers and import library. Build failures in this stack are opaque and hard to diagnose
from CI logs. The resulting binary embeds CGo runtime overhead (a separate OS thread for
the C runtime) and is harder to cross-compile.

The `mrlm-net/simconnect` library calls `SimConnect.dll` at runtime using `syscall.LoadDLL`
and `proc.Call`, the same mechanism Go's standard library uses for Windows API calls.
No C compiler is needed. The only runtime requirement is that `SimConnect.dll` is present
on the PATH or in the binary's directory — which it always is on a machine running MSFS,
P3D, or FSX. CI can test all non-bridge code on Linux with the MockBridge, and the Windows
CI job builds and tests the real bridge without any SDK installation step.

The `purego` library was evaluated but its calling convention support does not accommodate
the mixed HRESULT/HANDLE patterns in the SimConnect API surface without significant
adaptation work, providing no advantage over `mrlm-net/simconnect`'s purpose-built
implementation.

### Consequences

- Add `github.com/mrlm-net/simconnect` to `go.mod` as a direct dependency.
- `CGO_ENABLED=0` is valid for all builds, including the Windows CI job.
- `internal/bridge/simconnect_bridge.go` carries `//go:build windows` and wraps the
  `mrlm-net/simconnect` Manager type.
- No C compiler, no SimConnect SDK headers, no import library required at build time.
- The SimConnect.dll must be present on the target machine at runtime (guaranteed by MSFS
  installation).
- Go 1.25 is required (see ADR-009).

---

## 3. ADR-008: Bridge Interface and MockBridge for Cross-Platform Testing

**Status:** Accepted

### Context

The four MCP tool handlers (`get_simvar_value`, `get_simvar_values`, `transmit_event`,
`get_sim_state`) must work correctly on any platform in CI. The real SimConnect bridge
only compiles and runs on Windows with a simulator present. Without an abstraction layer,
tool handler unit tests would be Windows-only and simulator-dependent — making them
impractical in a standard GitHub Actions CI pipeline.

Three approaches were considered:

| Approach | Description | Tradeoff |
|---|---|---|
| No abstraction; test on Windows only | Tool handlers import the bridge directly | Tests require a running simulator; CI is Windows-only |
| Interface + build-tag mock | Define a `Bridge` interface; provide a `MockBridge` with no build tag | All tool handler tests run cross-platform |
| Integration tests only | Ship no unit tests; test only through HTTP | Poor test granularity; slow feedback |

### Decision

**Define a `Bridge` interface in `internal/bridge/bridge.go` (no build tag). Provide
`simconnectBridge` (Windows-only, wraps mrlm Manager) and `MockBridge` (no build tag,
used in tests). Expose `MountWithBridge` as a dependency injection hook.**

### File Layout

```
internal/bridge/
  bridge.go              Bridge interface + domain types (ConnectionState, SimVar,
                         SimVarRequest, SimVarResult, SimState, SimEvent, sentinel errors)
                         No build tag — compiles on all platforms.

  mock_bridge.go         MockBridge implementing Bridge.
                         No build tag — compiles on all platforms.
                         Used by bridge_test.go and integration tests.

  bridge_test.go         9 unit tests exercising MockBridge behaviour.
                         No build tag.

  simconnect_bridge.go   simconnectBridge implementing Bridge; wraps mrlm Manager.
                         //go:build windows — Windows-only.
```

### MountWithBridge Injection Hook

`internal/modes/simconnect/mount.go` exposes two entry points:

```go
// Mount creates a real simconnectBridge and calls MountWithBridge.
// Used in production. Only callable on Windows (//go:build windows on the
// mode registration file that calls this).
func Mount(r *gin.Engine, cfg Config) error

// MountWithBridge accepts any Bridge implementation.
// Used in tests to inject MockBridge without a simulator.
// No build tag — callable on all platforms.
func MountWithBridge(r *gin.Engine, cfg Config, b bridge.Bridge) error
```

The integration tests in `tests/integration/simconnect_mock_test.go` call
`MountWithBridge` directly, passing a `MockBridge` configured to return deterministic
data. This enables 10 end-to-end integration tests to run on the Windows CI job without
a live simulator connection.

### Rationale

Placing the `Bridge` interface and `MockBridge` in files with no build tag is the
minimum-cost cross-platform testing strategy. Tool handler code imports only
`internal/bridge` (the interface package), never the concrete bridge implementation.
The `MountWithBridge` hook follows standard Go dependency injection: the production path
wires the real bridge; the test path wires the mock. Neither path requires build tag
gymnastics in the handler code itself.

### Consequences

- Tool handlers in `internal/modes/simconnect/tools/` import `internal/bridge` only;
  they carry no build tags.
- `simconnect_bridge.go` is the sole file requiring `//go:build windows` in the bridge
  package.
- `cmd/simconnect-mcp/mode_simconnect_windows.go` (also `//go:build windows`) is the
  only file that instantiates `simconnectBridge` in the production path.
- `MockBridge` is a first-class package-level type, not a test helper buried in a
  `_test.go` file, so integration tests outside the `bridge` package can import it.

---

## 4. ADR-009: Go 1.25 Toolchain Bump

**Status:** Accepted

### Context

`github.com/mrlm-net/simconnect` v0.3.7 declares `go 1.25` in its `go.mod`. A consuming
module must declare an equal or higher Go version. At the start of Milestone 2 work, this
project's `go.mod` declared `go 1.24` and carried an explicit `toolchain go1.24.13`
directive.

### Decision

**Bump the `go` directive in `go.mod` from `1.24` to `1.25`. Remove the `toolchain
go1.24.13` line.**

### Rationale

The `toolchain` directive pins the specific Go toolchain binary. Removing it allows the
Go toolchain management introduced in Go 1.21 to select the appropriate toolchain
automatically based on the `go` directive, which is the intended usage for modules that
do not need to pin a specific patch. The bump from 1.24 to 1.25 is non-breaking: Go's
compatibility guarantee means all code valid in 1.24 is valid in 1.25.

The CI matrix already uses `go-version: 'stable'` for the Ubuntu job and installs the
latest stable Go on the Windows job, so both jobs naturally run on 1.25+.

### Consequences

- `go.mod` line 3: `go 1.24` changed to `go 1.25`.
- `go.mod`: `toolchain go1.24.13` directive removed.
- Minimum supported Go version for contributors becomes 1.25.
- No source code changes required beyond the `go.mod` edit.

---

## 5. Component Breakdown

### New Packages and Files (Milestone 2)

| File | Description |
|---|---|
| `internal/bridge/bridge.go` | `Bridge` interface and domain types: `ConnectionState`, `SimVar`, `SimVarRequest`, `SimVarResult`, `SimState`, `SimEvent`, sentinel errors (`ErrNotConnected`, `ErrUnknownVariable`, `ErrTimeout`). No build tag. |
| `internal/bridge/mock_bridge.go` | `MockBridge` — configurable in-process implementation of `Bridge` for tests. No build tag. |
| `internal/bridge/bridge_test.go` | 9 unit tests covering `MockBridge` state transitions, `GetSimVar`, `GetSimVars`, `TransmitEvent`, `GetSimState`, and `SimEvents`. No build tag. |
| `internal/bridge/simconnect_bridge.go` | `simconnectBridge` — production `Bridge` implementation wrapping the `mrlm-net/simconnect` Manager. `//go:build windows`. |
| `internal/modes/simconnect/config.go` | `Config` struct (`AppName`, `ListenAddr`) and `ConfigFromEnv()` reading `SIMCONNECT_APP_NAME` and `PORT`. |
| `internal/modes/simconnect/mode.go` | `simconnectMode` implementing the `modes.Mode` interface. Holds a `Bridge` and `Config`. |
| `internal/modes/simconnect/mount.go` | `Mount()` (production entry point, wires real bridge) and `MountWithBridge()` (injection hook for tests). No build tag on `MountWithBridge`. |
| `internal/modes/simconnect/health.go` | `HealthInfo()` returning `{"mode": "simconnect", "sim_connected": <bool>}`. |
| `internal/modes/simconnect/tools/simvar.go` | `get_simvar_value` (single variable) and `get_simvar_values` (batch, max 20 variables) tool registration and handlers. |
| `internal/modes/simconnect/tools/events.go` | `transmit_event` tool registration and handler; validates value fits uint32 range. |
| `internal/modes/simconnect/tools/state.go` | `get_sim_state` tool registration and handler; never returns `isError=true`. |
| `cmd/simconnect-mcp/mode_registry.go` | Mode factory registry (`map[string]ModeFactory`). No build tag. Provides `RegisterMode` and `NewMode`. |
| `cmd/simconnect-mcp/mode_docs.go` | Registers `"docs"` factory via `init()`. No build tag. |
| `cmd/simconnect-mcp/mode_simconnect_windows.go` | Registers `"simconnect"` factory via `init()`. `//go:build windows`. |
| `tests/integration/simconnect_mock_test.go` | 10 integration tests exercising all 4 tools through the full HTTP stack with `MockBridge`. `//go:build windows`. |

### Responsibilities Summary

| Package | Responsibility |
|---|---|
| `internal/bridge` | Bridge abstraction: interface definition, domain types, MockBridge, Windows-only real bridge |
| `internal/modes/simconnect` | Mode wiring: config, mode struct, route mounting, health info |
| `internal/modes/simconnect/tools` | One file per tool group; pure tool registration and handler logic |
| `cmd/simconnect-mcp` | Mode registry, per-mode init registrations, entry point |
| `tests/integration` | End-to-end HTTP tests using MockBridge |

---

## 6. Bridge Interface Definition

The Bridge interface is defined in `internal/bridge/bridge.go`. All Bridge implementations
must be safe to call from multiple goroutines concurrently.

```go
// Package bridge defines the Bridge interface and the value types it operates on.
// The interface is platform-agnostic; the Windows implementation lives in
// simconnect_bridge.go (//go:build windows).
package bridge

import (
    "context"
    "errors"
)

// ConnectionState represents the current state of the SimConnect connection.
type ConnectionState int

const (
    // StateDisconnected means no active handle; reconnect attempts may be pending.
    StateDisconnected ConnectionState = iota
    // StateConnecting means SimConnect_Open has been called; waiting for confirmation.
    StateConnecting
    // StateConnected means the handle is open and dispatch is running.
    StateConnected
)

// SimVar is a named simulation variable with its current value.
type SimVar struct {
    Name    string  `json:"name"`
    Value   float64 `json:"value"`
    Unit    string  `json:"unit"`
    SimTime float64 `json:"sim_time,omitempty"`
}

// SimVarRequest identifies a variable to read.
type SimVarRequest struct {
    Name string `json:"name"`
    Unit string `json:"unit"`
}

// SimVarResult is one element of a GetSimVars batch response.
// Exactly one of Value or Error will be meaningful.
type SimVarResult struct {
    Name    string  `json:"name"`
    Unit    string  `json:"unit"`
    Value   float64 `json:"value,omitempty"`
    SimTime float64 `json:"sim_time,omitempty"`
    Error   string  `json:"error,omitempty"` // non-empty signals per-item failure
}

// SimState is a snapshot of top-level simulator state.
type SimState struct {
    Connected        bool    `json:"connected"`
    Paused           bool    `json:"paused"`
    CurrentFlight    string  `json:"current_flight"`
    SimTime          float64 `json:"sim_time"`
    SimulatorVersion string  `json:"simulator_version"`
}

// SimEvent represents a simulator lifecycle notification.
type SimEvent struct {
    Name  string `json:"name"`
    Value uint32 `json:"value,omitempty"`
}

// Bridge abstracts all SimConnect SDK calls needed by the four MCP tools.
// Implementations must be safe to call from multiple goroutines concurrently.
type Bridge interface {
    // Open starts the background connection and dispatch loop.
    // It does not block; connection establishment is asynchronous.
    // Calling Open on an already-open bridge is a no-op.
    Open(ctx context.Context, appName string) error

    // Close shuts down the dispatch loop and releases the SimConnect handle.
    // Calling Close on a closed bridge is a no-op.
    Close() error

    // State returns the current connection state without blocking.
    State() ConnectionState

    // GetSimVar reads a single simulation variable by name and unit.
    // Returns ErrNotConnected if the bridge is not in StateConnected.
    GetSimVar(ctx context.Context, name, unit string) (SimVar, error)

    // GetSimVars reads up to 20 simulation variables.
    // Each entry in the returned slice corresponds positionally to the input slice.
    // Per-variable errors are embedded in SimVarResult.Error rather than
    // aborting the batch.
    GetSimVars(ctx context.Context, vars []SimVarRequest) ([]SimVarResult, error)

    // TransmitEvent sends a named client event to the simulator.
    // value is the DWORD data attached to the event (0 for events that
    // ignore data).
    TransmitEvent(ctx context.Context, name string, value uint32) error

    // GetSimState returns a snapshot of top-level simulator state.
    GetSimState(ctx context.Context) (SimState, error)

    // SimEvents returns a read-only channel that receives lifecycle events.
    SimEvents() <-chan SimEvent
}

// Sentinel errors returned by Bridge implementations.
var (
    ErrNotConnected    = errors.New("bridge: not connected to simulator")
    ErrUnknownVariable = errors.New("bridge: unknown simulation variable")
    ErrTimeout         = errors.New("bridge: request timed out")
)
```

---

## 7. MCP Tool Catalogue

Simconnect mode exposes four MCP tools. All tool names are snake_case. All tools return
structured JSON results. Three tools require `StateConnected`; `get_sim_state` always
succeeds.

### 7.1 get_simvar_value

Read a single simulation variable from the live simulator.

| Property | Detail |
|---|---|
| **Tool name** | `get_simvar_value` |
| **Arguments** | `name` (string, required) — SimConnect variable name, e.g. `"PLANE ALTITUDE"` |
| | `unit` (string, required) — unit string, e.g. `"feet"` |
| **Returns** | `{"name": string, "value": float64, "unit": string, "sim_time": float64}` |
| **Error codes** | `BRIDGE_DISCONNECTED` — bridge not in StateConnected |
| | `UNKNOWN_VARIABLE` — SimConnect rejected the name/unit combination |
| | `INTERNAL_ERROR` — unexpected bridge error |

### 7.2 get_simvar_values

Read up to 20 simulation variables in a single batch request.

| Property | Detail |
|---|---|
| **Tool name** | `get_simvar_values` |
| **Arguments** | `vars` (array, required) — array of `{"name": string, "unit": string}` objects; maximum 20 entries |
| **Returns** | Array of `{"name": string, "unit": string, "value": float64, "sim_time": float64, "error": string?}` — `error` field is non-empty for per-item failures |
| **Error codes** | `BRIDGE_DISCONNECTED` — bridge not in StateConnected (aborts entire batch) |
| | `INVALID_ARGUMENT` — `vars` array contains more than 20 entries |
| | `INTERNAL_ERROR` — unexpected bridge error |
| **Note** | Per-variable `UNKNOWN_VARIABLE` failures are embedded in the result array as `error` fields; they do not abort the batch. |

### 7.3 transmit_event

Send a named client event to the simulator.

| Property | Detail |
|---|---|
| **Tool name** | `transmit_event` |
| **Arguments** | `name` (string, required) — SimConnect event name, e.g. `"LANDING_LIGHTS_TOGGLE"` |
| | `value` (number, optional, default `0`) — DWORD value attached to the event; must be in the range `[0, 4294967295]` (uint32) |
| **Returns** | `{"success": true, "event": string}` |
| **Error codes** | `BRIDGE_DISCONNECTED` — bridge not in StateConnected |
| | `INVALID_ARGUMENT` — `value` is outside the uint32 range or is non-integer |
| | `INTERNAL_ERROR` — unexpected bridge error |

### 7.4 get_sim_state

Return a snapshot of top-level simulator state. This tool never returns `isError=true`.
When the bridge is disconnected it returns `{"connected": false}` rather than an error.

| Property | Detail |
|---|---|
| **Tool name** | `get_sim_state` |
| **Arguments** | None |
| **Returns (connected)** | `{"connected": true, "paused": bool, "current_flight": string, "sim_time": float64, "simulator_version": string}` |
| **Returns (disconnected)** | `{"connected": false}` |
| **Error codes** | None — `isError` is never set to `true` by this tool |

### Tool Summary Table

| Tool | Requires connection | Max args | Returns on disconnected |
|---|---|---|---|
| `get_simvar_value` | Yes | — | `BRIDGE_DISCONNECTED` error |
| `get_simvar_values` | Yes | 20 vars | `BRIDGE_DISCONNECTED` error |
| `transmit_event` | Yes | — | `BRIDGE_DISCONNECTED` error |
| `get_sim_state` | No | — | `{"connected": false}` |

---

## 8. Configuration Contract

All configuration is sourced from environment variables. No package below `main.go` reads
environment variables directly. `ConfigFromEnv()` in `internal/modes/simconnect/config.go`
reads and validates the simconnect-mode-specific variables; the result is passed as a
`Config` struct.

| Env Var | Default | Description |
|---|---|---|
| `MCP_MODE` | `docs` | Set to `simconnect` to activate this mode. On non-Windows platforms, the `"simconnect"` factory is not registered and this value produces an unknown-mode error. |
| `SIMCONNECT_APP_NAME` | `simconnect-mcp` | Application name registered with SimConnect when calling `SimConnect_Open`. Appears in the simulator's add-on manager. |
| `PORT` | `8080` | HTTP listen port. `ListenAddr` is derived as `":" + PORT`. |
| `GIN_MODE` | `debug` | Gin logging verbosity: `debug`, `release`, or `test`. |

**Note:** There is no `SIMCONNECT_RECONNECT_INTERVAL` variable. Reconnection after
a dropped simulator connection is handled internally by the `mrlm-net/simconnect`
Manager. The bridge's `State()` method reflects the current connection state at any
moment; callers do not configure reconnect timing.

---

## 9. Error Handling Contract

All MCP tool handler errors return a structured JSON body as the MCP tool result content
with `isError=true`, consistent with the docs mode error format. The `/health` endpoint
always returns `200 OK`.

### Error Response Shape

```json
{
  "error": {
    "code": "BRIDGE_DISCONNECTED",
    "message": "SimConnect bridge is not connected to the simulator"
  }
}
```

### Error Code Reference

| Code | Condition | Tools |
|---|---|---|
| `BRIDGE_DISCONNECTED` | Bridge is not in `StateConnected` at the time of the call | `get_simvar_value`, `get_simvar_values`, `transmit_event` |
| `INVALID_ARGUMENT` | Parameter validation failed: `vars` array has more than 20 entries, or `value` is outside the uint32 range `[0, 4294967295]` | `get_simvar_values`, `transmit_event` |
| `UNKNOWN_VARIABLE` | SimConnect rejected the variable name/unit combination | `get_simvar_value`; embedded in per-item `error` field for `get_simvar_values` |
| `INTERNAL_ERROR` | Unexpected error returned by the bridge (e.g. timeout, unexpected nil) | All tools |

### Special Case: get_sim_state

`get_sim_state` never sets `isError=true`. When the bridge is disconnected it returns a
successful MCP result with body `{"connected": false}`. All optional fields (`paused`,
`current_flight`, `sim_time`, `simulator_version`) are omitted in the disconnected
response. This design allows AI agents to poll `get_sim_state` as a connection probe
without implementing error-handling logic.

### Health Endpoint

```json
{
  "status": "ok",
  "mode": "simconnect",
  "sim_connected": true
}
```

`sim_connected` reflects `bridge.State() == StateConnected` at the moment of the health
check. The endpoint always returns `200 OK`; a `sim_connected: false` value is
informational, not an error condition.

---

## 10. CI Matrix

Two jobs run in parallel on every push and pull request to the `milestone/2-simconnect`
branch.

### Job: linux (ubuntu-latest)

Validates all cross-platform code: the Bridge interface, MockBridge, domain types, and
all tool handlers. The `simconnect_bridge.go` file and the simconnect mode registration
are excluded by their `//go:build windows` tags.

```yaml
- os: ubuntu-latest
- go-version: 'stable'
- run: |
    go build ./...
    go vet ./...
    go test ./...
```

No build tags are passed. The `//go:build windows` files are excluded automatically.
This job serves as the correctness gate for all portable code.

### Job: windows (windows-latest)

Validates Windows-specific code: `simconnect_bridge.go`, the simconnect mode registration,
and the 10 integration tests that exercise all 4 tools through the HTTP stack with
`MockBridge`. The real bridge is compiled but not exercised against a live simulator in CI.

```yaml
- os: windows-latest
- go-version: 'stable'
- run: |
    go build -tags windows ./...
    go vet -tags windows ./...
    go test -tags windows ./...
```

The `-tags windows` flag activates the `//go:build windows` files. Because `MockBridge`
backs the integration tests rather than a real simulator, these tests run without MSFS
installed on the CI runner.

### Matrix Summary

| Job | OS | Build flags | Tests | Real bridge exercised |
|---|---|---|---|---|
| `linux` | `ubuntu-latest` | (none) | All non-windows tests | No |
| `windows` | `windows-latest` | `-tags windows` | All tests including integration | No (MockBridge) |
