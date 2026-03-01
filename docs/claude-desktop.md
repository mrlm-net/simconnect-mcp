---
title: Claude Desktop Setup
description: Configure Claude Desktop to use SimConnect MCP as an MCP server.
order: 2
section: integration
---

This guide walks through configuring Claude Desktop to use SimConnect MCP as an MCP server, enabling Claude to query SimConnect documentation or interact with a live flight simulator session.

## Prerequisites

- SimConnect MCP binary installed (see [Getting Started](/docs/getting-started))
- Claude Desktop installed — download from [https://claude.ai/download](https://claude.ai/download)

## Configuration

Open or create `claude_desktop_config.json` at the platform-specific location:

- **macOS / Linux**: `~/.config/claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

### Docs mode (cross-platform)

Use this configuration to enable SimConnect SDK documentation lookup. This mode runs on any operating system and does not require a running simulator.

```json
{
  "mcpServers": {
    "simconnect-mcp": {
      "command": "simconnect-mcp",
      "env": {
        "MCP_MODE": "docs"
      }
    }
  }
}
```

### SimConnect mode (Windows only)

Use this configuration to enable live data exchange with Microsoft Flight Simulator 2020 / 2024, Prepar3D, or FSX. Microsoft Flight Simulator must be running before Claude Desktop starts the server.

```json
{
  "mcpServers": {
    "simconnect-mcp": {
      "command": "simconnect-mcp",
      "env": {
        "MCP_MODE": "simconnect"
      }
    }
  }
}
```

> **Note**: The `simconnect-mcp` binary must be on your system `PATH`. If you installed via `go install`, the binary is placed in `$GOPATH/bin` (typically `~/go/bin` on macOS/Linux or `%USERPROFILE%\go\bin` on Windows). Add that directory to `PATH` if it is not already present.

## Verification

After saving the configuration file, restart Claude Desktop. Open a new conversation and ask:

> "What SimConnect variables are available for aircraft position?"

Claude should respond using the `list_simvars` tool, returning a list of position-related simulation variables such as `PLANE ALTITUDE`, `PLANE LATITUDE`, and `PLANE LONGITUDE`.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| "MCP server not found" | Binary not in PATH | Run `go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest` or add the binary directory to PATH |
| "Connection refused" | Server did not start | Check Claude Desktop logs at `~/Library/Logs/Claude/` (macOS) or `%APPDATA%\Claude\logs\` (Windows) |
| SimConnect mode not working | MSFS not running | Start MSFS 2020 / 2024 before launching Claude Desktop |
| "BRIDGE_DISCONNECTED" error | SimConnect disabled in sim | Enable SimConnect in MSFS General Options > Developers |
