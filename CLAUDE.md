# SimConnect MCP

A Model Context Protocol (MCP) server in Go for Microsoft Flight Simulator 2020 and 2024. It serves the SimConnect SDK documentation and the mrlm-net/simconnect library guides anywhere, and, on Windows, live simulator data, user aircraft control and AI traffic with ATC through [github.com/mrlm-net/simconnect](https://github.com/mrlm-net/simconnect) (pure Go, no CGo).

## Stack

- Go 1.27; standard library plus Gin (`github.com/gin-gonic/gin`) for the HTTP transports, the `/health` endpoint and middleware.
- The MCP protocol is hand-written in `internal/mcpadapter` (JSON-RPC 2.0, no MCP SDK such as mcp-go).
- Transports: **stdio** (chosen automatically when stdin is a pipe, e.g. launched by Claude Code), **streamable HTTP** (`POST /mcp`) and legacy **SSE** (`GET /sse` + `POST /message`). HTTP listens on `PORT` (default 8080).
- Website: SvelteKit in `website/`, rendering `docs/*.md` (top level only).

## Modes (`MCP_MODE`)

| Value | Tools | Platform |
|-------|-------|----------|
| `docs` (default) | 15 docs tools | any |
| `simconnect` | 61 live tools | Windows |
| `both` | 76 = 15 + 61 | docs anywhere; live tools on Windows when SimConnect opens at start (else docs only) |

Env: `DOCS_MSFS_VERSION` (2020/2024, default 2024), `DOCS_OVERRIDE_PATH`, `DOCS_LIVE_SCRAPE`, `SIMCONNECT_APP_NAME`, `SIMCONNECT_AIRCRAFT_PROFILES`, `SIMCONNECT_TRAFFIC_DATA`, `PORT`.

## Layout

```
cmd/simconnect-mcp/        main: picks the mode (registry; simconnect only on Windows), stdio or HTTP, graceful shutdown
internal/mcpadapter/       hand-written MCP server: tool builder, JSON-RPC dispatch, stdio / streamable HTTP / SSE
internal/server/           Gin engine with recovery, request ID, CORS and JSON 404/405 middleware
internal/modes/            Mode interface (Mount, ServeStdio, HealthInfo; Closer)
internal/modes/docs/       docs mode; tools/ = SDK reference + library guide tools
internal/modes/simconnect/ live mode (windows); tools/ = every live tool, RegisterAll is the single list
internal/modes/both/       docs + live on one server (non-Windows build falls back to docs)
internal/bridge/           Bridge interface over SimConnect (SimVars, events, traffic, facilities); Windows impl + mock
internal/live/             (windows) the library's pkg/airport, pkg/nav, pkg/systems and pkg/traffic/world on the bridge's manager
internal/corpus/           embedded SimConnect SDK docs corpus (JSON) and its DocStore
internal/libdocs/          embedded mrlm-net/simconnect library guides, at the version go.mod requires
tools/scraper/             scrapes docs.flightsimulator.com into internal/corpus/assets (go generate ./internal/corpus/)
tools/libdocs/             copies the library guides into internal/libdocs/assets (go generate ./internal/libdocs/)
tools/changelog/           writes docs/changelog.md from CHANGELOG.md (go generate ./tools/changelog/)
tests/integration/         docs-mode and mock-bridge end-to-end tests
docs/                      website docs; docs/decisions/ = ADRs and the historic milestone plans (not on the site)
```

## Commands

```sh
go build ./... && go vet ./... && go test ./...
MCP_MODE=docs go run ./cmd/simconnect-mcp/
MCP_MODE=both go run ./cmd/simconnect-mcp/              # Windows; .mcp.json runs this over stdio
GOOS=windows go build ./cmd/simconnect-mcp/             # cross-compile
go generate ./internal/libdocs/                         # after a library bump
go generate ./tools/changelog/                          # after editing CHANGELOG.md
```

Windows-only code carries `//go:build windows` or a `_windows.go` suffix; the tag comes from `GOOS`, never from `-tags`. No `runtime.GOOS` checks in production paths.

## Tools

- 76 tools: 61 simconnect + 15 docs, documented in `docs/mcp-tools-simconnect.md` and `docs/mcp-tools-docs.md`. Every tool added, removed or changed updates those docs and the counts (docs front matter and intro, README, website home page, CHANGELOG "Tool counts").
- Tool schemas go to the client on every connect: keep a tool description to 1–2 sentences and each parameter to one short line (units, enum values, default). Detail belongs in the docs.
- Errors are tool results with a code prefix (`INVALID_ARGUMENT:`, `NOT_FOUND:`, `NOT_APPLICABLE:`, `BRIDGE_DISCONNECTED:`, `TIMEOUT:`), not protocol errors.

## Library and traffic

- Bump the library: edit the `github.com/mrlm-net/simconnect` version in go.mod, `go mod tidy`, then `go generate ./internal/libdocs/` to refresh the bundled guides.
- Reuse the library (pkg/airport, pkg/nav, pkg/traffic, pkg/systems, pkg/gsx, pkg/lvars); fix library bugs in the library, not here.
- SimConnect IDs: bridge from 1,000,002, live runtime from 900,000,000, traffic engine `IDBase` 950,000,000 (`internal/live/world_windows.go`).
- Stdout is the stdio MCP channel: `ServeStdio` redirects `os.Stdout` to stderr, so logs (including the library's) go to stderr.
- The traffic engine restarts on a SimConnect reconnect and re-applies the kept schedule, real traffic and corridor settings once it is up (`TrafficWorld.restart` / `reapply`).
- On shutdown the mode removes the AI aircraft it spawned.

## Conventions

- No third-party dependencies beyond what go.mod has; stdlib or mrlm-net libraries.
- No global state; pass dependencies explicitly. Unit tests next to the code, integration tests in `tests/`.
- Branches: `feat/…`, `fix/…`, `chore/simconnect-0.xx` (library bumps), `docs/…`. Squash-merge PRs into `main`.
- Releases are beta: a patch for fixes, a minor for features. `CHANGELOG.md` is the source of release notes (Keep a Changelog); regenerate `docs/changelog.md` from it, never edit that copy by hand.
- Licence: BSL 1.1 after v0.8.0; commercial use via support@mrlm.net.

## Workload Management

GitHub Issues in `mrlm-net/simconnect-mcp`. Post decisions, blockers and outcomes, not progress chatter. Project board IDs: `.claude/github-board.md`.
