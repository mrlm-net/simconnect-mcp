---
title: Changelog
description: Release history for SimConnect MCP.
order: 1
section: changelog
---

# Changelog

All notable changes to SimConnect MCP are documented here.

## [Unreleased]

### Added
- Milestone 3: Documentation website at simconnect-mcp.mrlm.net
- Offline docs snapshot with pre-built corpus (scheduled updates)
- Scraping guard requiring explicit confirmation for live doc scraping

## [0.2.0] — 2025-XX-XX — Milestone 2: SimConnect Live Data

### Added
- Live SimConnect mode (`MCP_MODE=simconnect`) with Windows-only build tag
- `get_simvar_value` — read a single live simulation variable
- `get_simvar_values` — batch read up to 20 simulation variables
- `transmit_event` — send SimConnect client events to the simulator
- `get_sim_state` — simulator connection state snapshot
- CGo/FFI bridge to SimConnect.dll via Windows SDK
- Auto-reconnect on simulator restart

### Changed
- Server startup now checks `MCP_MODE` environment variable

## [0.1.0] — 2025-XX-XX — Milestone 1: Documentation Mode

### Added
- Initial release with `MCP_MODE=docs` (cross-platform)
- 11 MCP tools for SimConnect SDK documentation: `list_simvars`, `get_simvar`, `list_events`, `get_event`, `list_functions`, `get_function`, `list_structures`, `get_structure`, `list_error_codes`, `get_error_code`, `search_docs`
- Full-text search across SimConnect SDK corpus
- Pagination support for all list tools
- MSFS 2020 and MSFS 2024 documentation support
- Gin HTTP framework with MCP-over-HTTP/SSE protocol
- Cross-platform build (linux/amd64, darwin/amd64, windows/amd64)
