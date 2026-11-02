---
name: check-sim
description: Check that Claude can reach Microsoft Flight Simulator and say in plain words what is going on. Use whenever someone asks if the sim is connected, whether the setup works, why Claude can't see the sim, or wants a quick look at their flight, even if they don't say "check".
allowed-tools: mcp__plugin_simconnect_simconnect__get_sim_state, mcp__plugin_simconnect_simconnect__get_aircraft_systems, mcp__plugin_simconnect_simconnect__get_nearest_airport
---

# Check the simulator

The person is probably a tester, not a developer. Answer in everyday words, without tool names, error codes or file paths unless they ask.

1. Call `get_sim_state`.
2. If it is connected, call `get_aircraft_systems` and `get_nearest_airport`, then say in two or three lines: connected or not, which aircraft, where it is, and whether the sim is paused.
3. If the tool is missing or says it is disconnected, work through this list and stop at the first thing that helps:
   - The simulator may not be running, or may still be loading. The connection comes by itself once the sim reaches the main menu or a flight, so ask them to wait a minute and try again.
   - The live tools only run on Windows with Microsoft Flight Simulator 2020 or 2024.
   - The plugin's mode may be "docs". They can change it with `/plugin`, under Configure.
   - Windows may have blocked the program. If the server never started, ask them to look for a Windows Security or Smart App Control message and tell you what it says.

Only read here. Don't change anything in the sim, and don't start traffic.

## Reply

Keep it short: the answer and what they need to do next. No step-by-step narration or explaining yourself unless asked.
