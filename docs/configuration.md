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
| `SIMCONNECT_APP_NAME` | `simconnect-mcp` | Application name registered with SimConnect (SimConnect mode only) |
| `DOCS_MSFS_VERSION` | `2024` | MSFS version for the docs corpus: `2020`, `2024`, or `both` |
| `DOCS_OVERRIDE_PATH` | *(embedded)* | Filesystem path to a directory of JSON corpus files. Overrides the embedded corpus. See security note below. |
| `GIN_MODE` | `debug` | Gin server mode: `debug` or `release`. In `release` mode, CORS is restricted to localhost only (DNS rebinding protection). |

### MCP_MODE

Selects the operating mode at startup. The server does not support switching modes at runtime; restart with a different value to change modes.

- `docs` — scrapes and serves SimConnect SDK reference documentation. Cross-platform, no simulator required.
- `simconnect` — connects to a running MSFS instance via the SimConnect SDK. Windows only; requires the `windows` build tag and the SimConnect SDK.

### PORT

The TCP port the HTTP server binds to. Use this to avoid conflicts with other local services or to expose the server on a non-default port.

### SIMCONNECT_APP_NAME

The name SimConnect MCP registers with the simulator when establishing a connection. This name appears in the simulator's SimConnect client list. Only applies in `simconnect` mode.

### DOCS_MSFS_VERSION

Controls which version of the SimConnect SDK documentation is fetched when running in `docs` mode. Set to `2020` for the original MSFS 2020 SDK, `2024` (the default) for the MSFS 2024 SDK, or `both` to merge both corpora — SimVars and events include a `versions` field indicating which simulator defines them. Only applies in `docs` mode.

### DOCS_OVERRIDE_PATH

Points the server at a local directory of JSON corpus files instead of the compiled-in embedded corpus. Useful for testing a freshly scraped corpus without rebuilding the binary.

**Security note**: `DOCS_OVERRIDE_PATH` must point to a trusted local directory. Never derive it from user-provided input. In production, leave it unset to use the embedded corpus.

### GIN_MODE

Controls Gin's logging verbosity and CORS behaviour.

- `debug` (default) — verbose request logging, all origins permitted. Suitable for local development.
- `release` — minimal logging, CORS restricted to `localhost` origins only. Recommended for any non-development deployment (DNS rebinding protection).

## Version

To check the installed binary version:

```bash
simconnect-mcp --version
# simconnect-mcp v0.1.0 (commit abc1234, built 2026-03-01)
```

The version, commit hash, and build date are embedded at build time by GoReleaser. Binaries built locally with `go build` will show `dev / none / unknown`.

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
