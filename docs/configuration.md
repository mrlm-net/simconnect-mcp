---
title: Configuration
description: Environment variables and configuration options for SimConnect MCP.
order: 2
section: getting-started
---

# Configuration

SimConnect MCP is configured entirely through environment variables. There are no configuration files or command-line flags. All variables have safe defaults so the server runs without any configuration in docs mode.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_MODE` | `docs` | Operating mode: `docs` (cross-platform) or `simconnect` (Windows only) |
| `PORT` | `8080` | TCP port to listen on |
| `SIMCONNECT_APP_NAME` | `SimConnectMCP` | Application name registered with SimConnect (SimConnect mode only) |
| `DOCS_MSFS_VERSION` | `2024` | MSFS version to use when scraping docs: `2020` or `2024` |

### MCP_MODE

Selects the operating mode at startup. The server does not support switching modes at runtime; restart with a different value to change modes.

- `docs` — scrapes and serves SimConnect SDK reference documentation. Cross-platform, no simulator required.
- `simconnect` — connects to a running MSFS instance via the SimConnect SDK. Windows only; requires the `windows` build tag and the SimConnect SDK.

### PORT

The TCP port the HTTP server binds to. Use this to avoid conflicts with other local services or to expose the server on a non-default port.

### SIMCONNECT_APP_NAME

The name SimConnect MCP registers with the simulator when establishing a connection. This name appears in the simulator's SimConnect client list. Only applies in `simconnect` mode.

### DOCS_MSFS_VERSION

Controls which version of the SimConnect SDK documentation is fetched when running in `docs` mode. Set to `2020` to serve documentation for the original MSFS 2020 SDK, or `2024` (the default) for the MSFS 2024 SDK. Only applies in `docs` mode.

## Examples

Run in docs mode on a non-default port:

```bash
MCP_MODE=docs PORT=9000 simconnect-mcp
```

Run in SimConnect mode with a custom application name:

```bash
MCP_MODE=simconnect SIMCONNECT_APP_NAME=MyApp simconnect-mcp
```

Run in docs mode targeting MSFS 2020 documentation:

```bash
MCP_MODE=docs DOCS_MSFS_VERSION=2020 simconnect-mcp
```

## Build Tags

SimConnect mode requires the `windows` build tag. This tag gates the CGo/FFI bindings to the SimConnect SDK so the package compiles on non-Windows platforms without the SDK present.

Build with SimConnect support (Windows only):

```bash
go build -tags windows ./cmd/simconnect-mcp/
```

Cross-platform build for docs mode (no build tag required):

```bash
go build ./cmd/simconnect-mcp/
```

When cross-compiling for Windows from another OS, combine the build tag with `GOOS`:

```bash
GOOS=windows go build -tags windows ./cmd/simconnect-mcp/
```

Attempting to run `MCP_MODE=simconnect` with a binary built without the `windows` tag will result in a startup error, preventing accidental use of an unsupported configuration.
