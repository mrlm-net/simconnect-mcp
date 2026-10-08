---
title: Getting Started
description: Install and run SimConnect MCP in docs mode or live SimConnect mode.
order: 1
section: getting-started
---

SimConnect MCP is a [Model Context Protocol](https://modelcontextprotocol.io/) server for Microsoft Flight Simulator (MSFS 2020/2024). It operates in three modes:

- **Docs mode** — serves SimConnect SDK reference documentation and the `github.com/mrlm-net/simconnect` Go library guides to MCP clients. Cross-platform, no simulator required.
- **SimConnect mode** — reads real-time simulator data via the SimConnect SDK. Windows only, requires a running simulator.
- **Both mode** — docs mode plus, on Windows with the simulator reachable at startup, the SimConnect tools. See [Configuration](/docs/configuration).

## Prerequisites

### All modes

- Go 1.27 or later

### SimConnect mode only

- Windows 10 or Windows 11
- Windows SDK with SimConnect SDK headers and libraries
- Microsoft Flight Simulator 2020 or Microsoft Flight Simulator 2024, running with SimConnect enabled

## Installation

### Option 1 — Download prebuilt binary (recommended)

Download the latest release for your platform from the [GitHub Releases page](https://github.com/mrlm-net/simconnect-mcp/releases).

Extract the archive and place the `simconnect-mcp` binary in a directory on your `PATH`.

### Option 2 — Install binary

Install the latest released binary directly from the module proxy:

```bash
go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest
```

### Option 3 — Build from source

Clone the repository and build locally:

```bash
git clone git@github.com:mrlm-net/simconnect-mcp.git
cd simconnect-mcp
go mod download
go build ./cmd/simconnect-mcp/
```

The resulting binary is placed in the current directory as `simconnect-mcp` (or `simconnect-mcp.exe` on Windows).

## Run in Docs Mode (cross-platform)

Docs mode scrapes and serves the SimConnect SDK reference to MCP clients. It runs on any operating system and does not require a simulator or Windows SDK.

```bash
MCP_MODE=docs simconnect-mcp
```

Verify the server is running:

```bash
curl http://localhost:8080/health
# {"docs_loaded":true,"docs_source":"embedded","library_version":"v0.31.1","mode":"docs","msfs_version":"2024","status":"ok",...}
```

The server listens on port `8080` by default. See [Configuration](/docs/configuration) to change the port or MSFS version used for documentation.

## Run in SimConnect Mode (Windows only)

SimConnect mode connects to a running instance of MSFS 2020 or MSFS 2024 and streams live simulator data to MCP clients. SimConnect must be enabled in the simulator's settings.

```bash
MCP_MODE=simconnect simconnect-mcp
```

Or using `go run` with the required build tag:

```bash
go run -tags windows ./cmd/simconnect-mcp/
```

Verify the server is running and connected:

```bash
curl http://localhost:8080/health
# {"app_name":"simconnect-mcp","connection_state":"connected","mode":"simconnect","sim_connected":true,"status":"ok"}
```

If the simulator is not running or SimConnect is disabled, `sim_connected` is `false` (`connection_state` says `disconnected` or `connecting`) and the server will return errors for data requests until a connection is established.

## MCP Client Setup

Point your MCP client at `http://localhost:8080`. The server implements the MCP protocol over HTTP with Server-Sent Events (SSE) for streaming responses.

See [Claude Desktop Setup](/docs/claude-desktop) for a complete configuration example including how to register SimConnect MCP as a local MCP server in Claude Desktop.

## Next: bring your airport to life

In SimConnect mode, with your aircraft at an airport, ask your AI assistant to "run the airline schedule here". AI departures and arrivals of your own then appear, and a tower and approach controller clear and sequence them. The [AI Traffic & ATC guide](/docs/ai-traffic) explains what happens and what you can ask.
