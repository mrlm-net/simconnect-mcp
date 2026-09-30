# MCP Tool API Reference

This page has moved. The MCP tool reference is maintained in two documents, which are also published on the project website:

- [MCP Tools — Docs Mode](../mcp-tools-docs.md) — the 15 tools available with `MCP_MODE=docs`: 12 SimConnect SDK reference tools (SimVars, events, functions, structures, error codes, search) and 3 tools for the `github.com/mrlm-net/simconnect` Go library guides.
- [MCP Tools — SimConnect Mode](../mcp-tools-simconnect.md) — the 19 live-data tools available with `MCP_MODE=simconnect` (Windows only).

With `MCP_MODE=both` on Windows the server exposes both sets (34 tools) when SimConnect is reachable at startup, and the 15 docs tools otherwise.
