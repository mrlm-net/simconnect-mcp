---
title: Claude Code Setup
description: Add SimConnect MCP to Claude Code for in-editor SimConnect SDK lookups and live simulator access.
order: 1
section: integration
---

[Claude Code](https://claude.ai/code) is Anthropic's CLI for AI-assisted development. SimConnect MCP runs as an MCP server inside your Claude Code session, giving Claude direct access to the SimConnect SDK corpus and — on Windows — live simulator data.

## Quick start (project scope)

Add SimConnect MCP to the current project's `.mcp.json` file:

```bash
claude mcp add --transport stdio \
  --env MCP_MODE=docs \
  --env DOCS_MSFS_VERSION=2024 \
  --scope project \
  simconnect-mcp \
  -- simconnect-mcp
```

This creates (or updates) `.mcp.json` at the repository root. Commit the file so all contributors pick up the server automatically.

## Manual configuration

Create or edit `.mcp.json` at the project root:

### Docs mode (cross-platform)

```json
{
  "mcpServers": {
    "simconnect-mcp": {
      "type": "stdio",
      "command": "simconnect-mcp",
      "env": {
        "MCP_MODE": "docs",
        "DOCS_MSFS_VERSION": "2024"
      }
    }
  }
}
```

### Using `go run` (no binary required)

```json
{
  "mcpServers": {
    "simconnect-mcp": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "./cmd/simconnect-mcp/"],
      "env": {
        "MCP_MODE": "docs",
        "DOCS_MSFS_VERSION": "2024"
      }
    }
  }
}
```

### SimConnect mode (Windows only)

```json
{
  "mcpServers": {
    "simconnect-mcp": {
      "type": "stdio",
      "command": "simconnect-mcp",
      "env": {
        "MCP_MODE": "simconnect"
      }
    }
  }
}
```

Start MSFS 2020 / 2024 before launching Claude Code. The server reconnects automatically if the simulator restarts mid-session.

## Scope options

| Scope | Flag | Location | Shared |
|-------|------|----------|--------|
| Project | `--scope project` | `.mcp.json` (repo root) | Yes — commit to git |
| Local | `--scope local` | `.claude/mcp.json` | No — gitignored |
| User | `--scope user` | `~/.claude/mcp.json` | No — your machine only |

Use **project** scope for SimConnect MCP so every contributor in the repo gets the server without manual setup.

## Verification

Inside a Claude Code session, run:

```
/mcp
```

The output lists all configured servers and their connection status. `simconnect-mcp` should show as **connected** with 11 tools available (docs mode) or 4 tools (simconnect mode).

From the CLI, outside a session:

```bash
claude mcp list
claude mcp get simconnect-mcp
```

Test the server is responding by asking Claude:

> "What SimConnect variables relate to aircraft altitude?"

Claude should call `list_simvars` and return variables such as `PLANE ALTITUDE`, `INDICATED ALTITUDE`, and `PLANE ALT ABOVE GROUND`.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Server not listed in `/mcp` | `.mcp.json` not found or malformed | Run `claude mcp list` to see all scopes; check JSON syntax |
| `command not found: simconnect-mcp` | Binary not on PATH | Run `go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest`, ensure `$GOPATH/bin` is in PATH |
| 0 tools available | Server started but mode incorrect | Verify `MCP_MODE` env var is set to `docs` or `simconnect` |
| SimConnect mode errors | MSFS not running | Start MSFS 2020 / 2024 before launching Claude Code |
