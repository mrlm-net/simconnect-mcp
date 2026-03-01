---
title: Changelog
description: Release history for SimConnect MCP.
order: 1
section: changelog
---

All notable changes to SimConnect MCP are documented here.

## [0.1.1] - 2026-03-01

### Fixed

- Install command in hero no longer wraps to a second line; scrollbar hidden while scroll is preserved
- Removed empty Unreleased section from changelog

## [0.1.0] - 2026-03-01

### Added
- Documentation website at [simconnect-mcp.mrlm.net](https://simconnect-mcp.mrlm.net)
- Stdio transport — server auto-detects pipe and switches to stdio for Claude Code and other local MCP clients
- Claude Code integration via `.mcp.json` and documentation at `/docs/claude-code`
- GPS variables for MSFS 2020 (430 active) and MSFS 2024 (430 deprecated) across 5 subcategories
- GoReleaser cross-platform binary builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64)
- `--version` flag with ldflags-embedded version, commit, and build date
- Live SimConnect mode (`MCP_MODE=simconnect`, Windows only) with CGo/FFI bridge to SimConnect.dll
- `get_simvar_value` — read a single live simulation variable from a running simulator
- `get_simvar_values` — batch read up to 20 simulation variables
- `transmit_event` — send SimConnect client events to the simulator
- `get_sim_state` — simulator connection state snapshot
- Auto-reconnect on simulator restart
- 11 MCP tools for SimConnect SDK documentation: `list_simvars`, `get_simvar`, `list_events`, `get_event`, `list_functions`, `get_function`, `list_structures`, `get_structure`, `list_error_codes`, `get_error_code`, `search_docs`
- Full-text search across SimConnect SDK corpus
- Pagination support for all list tools (default 20, max 100 per page)
- MSFS 2020 and MSFS 2024 documentation corpus (1,800+ simulation variables)
- MCP-over-HTTP/SSE and streamable HTTP transports

### Changed
- Server startup checks `MCP_MODE` environment variable (`docs` default, `simconnect`, `both`)
