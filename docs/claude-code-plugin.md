---
title: "Claude Code Plugin"
description: "Install SimConnect MCP as a Claude Code plugin in one line: no Go, git or GitHub account needed."
order: 2
section: "getting-started"
---

# Claude Code Plugin

The `simconnect` plugin puts the server and two ready-made skills into Claude Code. You don't need Go, git or a GitHub account.

## Install

You need Windows 10 or 11 and Microsoft Flight Simulator 2020 or 2024.

1. Install Claude Code, if you don't have it. In PowerShell:

   ```powershell
   irm https://claude.ai/install.ps1 | iex
   ```

2. Install the plugin:

   ```powershell
   claude plugin install simconnect --marketplace https://raw.githubusercontent.com/mrlm-net/simconnect-mcp/marketplace/marketplace.json
   ```

3. Start the simulator, run `claude`, and type:

   ```
   /simconnect:check-sim
   ```

If the sim is still loading, wait a minute and try again. The connection comes by itself.

## What you can do

| Type this, or just ask | What happens |
|---|---|
| `/simconnect:check-sim` | Checks that Claude can see the sim, and says which aircraft and where |
| `/simconnect:report-bug` | Captures what the sim shows right now, asks you up to three questions, and saves a report on your Desktop |

You can also ask in your own words, such as "what's my fuel?" or "which aircraft are near me?". All 96 tools are available.

## Settings

Choose the mode with `/plugin`, under Configure:

- `both` (default): the documentation and the live simulator.
- `docs`: the documentation only. It never touches the simulator, which suits a second session on a machine where another test is running.

While the plugin is enabled, Claude gives short answers without explaining itself in all your sessions, not only about the sim. This replaces your own output style. To get yours back, turn the plugin off with `claude plugin disable simconnect@simconnect`.

## Update and remove

```powershell
claude plugin update simconnect@simconnect
claude plugin uninstall simconnect@simconnect
```
