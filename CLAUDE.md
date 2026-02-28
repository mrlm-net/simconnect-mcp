# SimConnect MCP

A Model Context Protocol (MCP) server for Microsoft Flight Simulator / Prepar3D / FSX, written in Go with Gin routing. Operates in two modes reflecting the project milestones: **documentation fetch** (cross-platform) and **live SimConnect data** (Windows-only via SimConnect SDK).

## Tech Stack

- **Language**: Go 1.23
- **HTTP framework**: Gin (`github.com/gin-gonic/gin`)
- **Protocol**: Model Context Protocol (MCP) over HTTP/SSE
- **Platform**: Cross-platform for Milestone 1 (docs); Windows-only for Milestone 2 (SimConnect SDK)
- **Build tool**: `go build` / `go test`

## Architecture

The server supports two operating modes, selected at startup via the `MCP_MODE` environment variable:

| Mode | Value | Milestone | Platform |
|------|-------|-----------|----------|
| Documentation fetch | `docs` | 1 | Any |
| Live SimConnect data | `simconnect` | 2 | Windows only |

```
cmd/simconnect-mcp/       Entry point — reads MCP_MODE, boots Gin, registers routes
internal/server/          Shared Gin router setup and middleware
internal/modes/docs/      Milestone 1: scrapes/serves SimConnect SDK documentation
internal/modes/simconnect/ Milestone 2: CGo/FFI bridge to SimConnect.dll (windows build tag)
```

The SimConnect mode (`internal/modes/simconnect/`) is gated with `//go:build windows` so the binary compiles cross-platform for docs mode.

## Development

### Prerequisites

- Go 1.23+
- Windows SDK + SimConnect SDK (for Milestone 2 only)
- `gh` CLI (for issue/project management)

### Getting Started

```sh
git clone git@github.com:mrlm-net/simconnect-mcp.git
cd simconnect-mcp
go mod download

# Run in docs mode (default, cross-platform)
MCP_MODE=docs go run ./cmd/simconnect-mcp/

# Run in live SimConnect mode (Windows only)
MCP_MODE=simconnect go run -tags windows ./cmd/simconnect-mcp/
```

### Commands

| Command | Purpose |
|---------|---------|
| `go build ./...` | Build all packages |
| `go build ./cmd/simconnect-mcp/` | Build the server binary |
| `go test ./...` | Run all unit tests |
| `go vet ./...` | Static analysis |
| `GOOS=windows go build ./cmd/simconnect-mcp/` | Cross-compile for Windows |

## Conventions

- **Milestones drive branching**: `milestone/1-docs`, `milestone/2-simconnect`
- **Build tags**: SimConnect-specific code always carries `//go:build windows`; never use `runtime.GOOS` checks in production paths
- **Package naming**: `internal/modes/<mode>` — one package per operating mode
- **Error handling**: always return structured JSON errors from Gin handlers; never `log.Fatal` inside handlers
- **MCP tools**: each SimConnect variable/doc endpoint maps to one MCP tool definition
- **No global state**: pass dependencies explicitly; avoid `init()` side effects
- **Tests**: unit tests live alongside the code (`_test.go`); integration tests in `tests/`

## Workload Management

Agents track work decisions, blockers, and outcomes in GitHub Issues.

**System**: GitHub Issues
**Repository**: `mrlm-net/simconnect-mcp`
**Configuration**:
- Use the `github-issues` skill for issue management
- Agents post decisions (e.g., "Chose CGo over purego because SimConnect requires COM"), blockers, quality gate failures, and milestone outcomes
- Agents do NOT post progress notifications or status updates — keep it human-consumable
- Milestone 1 issues carry label `milestone-1`; Milestone 2 issues carry label `milestone-2`

## GitHub Projects v2

**Project Number**: 11
**Project ID**: `PVT_kwDOBxaH0c4BPjQl`
**Owner**: `mrlm-net`
**Level**: organization

**Custom Fields**:
- **Status** (single-select)
  - Field ID: `PVTSSF_lADOBxaH0c4BPjQlzg97WCg`
  - Options:
    - Backlog: `f75ad846`
    - Ready: `61e4505c`
    - In progress: `47fc9ee4`
    - In review: `df73e18b`
    - Done: `98236657`
- **Priority** (single-select)
  - Field ID: `PVTSSF_lADOBxaH0c4BPjQlzg97WHo`
  - Options:
    - P0: `79628723`
    - P1: `0a877460`
    - P2: `da944a9c`
- **Size** (single-select)
  - Field ID: `PVTSSF_lADOBxaH0c4BPjQlzg97WHs`
  - Options:
    - XS: `6c6483d2`
    - S: `f784b110`
    - M: `7515a9f1`
    - L: `817d0097`
    - XL: `db339eb2`

## MRLM Plugin Usage

This project uses the [mrlm devstack plugin](https://github.com/mrlm-net/devstack) for AI-assisted development. Available commands:

| Command | What it does |
|---------|-------------|
| `/spec` | Gather requirements, write user stories and acceptance criteria |
| `/design` | Design system architecture, define interfaces and technical patterns |
| `/build` | Implement code and unit tests (engineer only, no review) |
| `/review` | Systematic code review for correctness, style, and performance |
| `/test` | Run E2E, performance, UX, and accessibility testing |
| `/secure` | Vulnerability scan, SBOM generation, OWASP compliance check |
| `/deploy` | Infrastructure provisioning and deployment automation |
| `/make` | Full SDLC pipeline — from requirements through security scan |
| `/ask` | Ask any question using full agent toolkit (read-only) |
| `/write` | Generate articles, documentation, or marketing content |
| `/release` | Publish versioned release with changelog, git tag, and GitHub Release |
| `/scope` | Plan from issue/work item or topic — analysis, design, planning, and backlog creation |
| `/init` | Initialize project structure and CLAUDE.md |

### Recommended Workflow

**Milestone 1 — Documentation mode**: `/make "implement SimConnect docs MCP server"`

**Milestone 2 — Live data mode**: `/make "implement SimConnect live data MCP server"`

For focused work, chain individual commands:
1. `/spec` — define what to build
2. `/design` — plan how to build it
3. `/build` — implement it
4. `/review` — review the code
5. `/test` — verify it works
6. `/secure` — check for vulnerabilities
