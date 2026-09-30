---
title: AI Traffic & ATC
description: Bring your airport to life from an AI chat — AI departures and arrivals of your own, a tower and approach controller that clear and sequence them, and an airline schedule that runs by itself.
order: 1
section: guides
---

SimConnect MCP can fill the airport you're flying from with traffic of its own and control it the way a tower and approach controller would. You ask in plain language, for example "run the airline schedule at Prague". The server then puts aircraft into the simulator and flies them from stand to runway and from their origin to the stand, clearing and spacing them as it goes.

It's built on the traffic packages of the [mrlm-net/simconnect](https://github.com/mrlm-net/simconnect) Go library. Every tool mentioned here is described in full in the [SimConnect Mode reference](/docs/mcp-tools-simconnect).

> **Beta.** The traffic tools change the simulator: they add and remove aircraft. They're new and still changing between releases.

## What you need

- **Windows with MSFS 2020 or 2024 running**, and the server in `simconnect` or `both` mode (see [Getting Started](/docs/getting-started)).
- **Your aircraft at or near the airport.** The simulator only serves an airport's stands, taxiways and procedures once it has loaded the airport around the user aircraft. Traffic is also scanned within about 43 NM of you.
- **AI aircraft models.** Each flight uses an installed aircraft of its type, in the livery of its airline (the first three letters of the call sign, e.g. `CSA` in `CSA123`). The more airline liveries you have installed, for example from an AI traffic package, the more the traffic looks like the real thing. `list_aircraft_models` shows what's installed.
- **A pushback tug.** Departures are pushed by the ground vehicle `FSDT_Pushback_03`, the library's default. `spawn_departure` takes `tug: false` to push without one.

## Two ways to add traffic

**By hand**, one aircraft at a time. `spawn_departure` puts an aircraft on a stand and `spawn_arrival` puts one on its arrival route (STAR). The runway, stand, SID or STAR and model are chosen for you unless you name them. By default each flight waits for your clearances: pushback, taxi, cross, line up, take off. You give them with `atc_clearance`, so you can play the controller yourself.

**By schedule.** `start_schedule` runs a realistic airline day at your airports, with flights that go by themselves. It uses the same generator as `generate_schedule`: airlines and their bases, fleets, routes and time-of-day waves. `get_schedule` shows the departure and arrival boards, and `stop_schedule` ends the run.

## A scheduled day at your airport

| Flight | What happens |
|---|---|
| **Departure** | Appears on a free stand that fits its wing span and airline, 10 minutes before its departure time (STD). It pushes back behind the tug at its STD, taxis the planned route, is cleared by the tower to line up and take off, and flies its SID. It's removed 30 minutes after take-off. |
| **Arrival** | Appears in the air 45 minutes before its arrival time (STA), on a flight plan from its origin, and flies to the start of its STAR. There it's handed to the arrival controller, which flies the STAR and approach, lands, vacates and taxis to a stand, and is removed 20 minutes after parking unless it turns around. If it can't fly en route (no flight plan, or already too close), it appears at the start of its STAR 25 minutes before its STA instead. |
| **Turnaround** | An arrival whose airline and type fly out again 40 minutes to 3 hours later stays on its stand. At its departure time it becomes that departure: the same aircraft, not a new one on another stand. |
| **Overflight** | Crosses the area within 100 NM of the first airport at cruise level, between airports outside it, and is removed once it has left. |

The runway in use is chosen for the wind at your aircraft, and it stays in use while the wind allows it (up to 5 kt tailwind), so a calm wind doesn't flip it from one flight to the next. Spawns are spaced out (arrivals 3 minutes apart, departures 1 minute), limited by `max_aircraft`, and retried with another model or stand if one fails.

## The tower and the approach controller

Every airport with traffic of ours gets two controllers:

- **A tower per runway** clears line-ups, take-offs and runway crossings. Departures go in the gaps between arrivals, after the wake and same-SID interval. If the runway won't be free on short final, it sends the arrival around onto the published missed approach and sequences it again.
- **A landing sequence per runway end** orders our arrivals and other traffic on the final. Spacing is the wake minimum and at least 5 NM, more in low visibility. Arrivals lose their delays by themselves: slower first, then a longer downwind, then a hold at the STAR's hold fix, stacked 1,000 ft apart.

You can watch them with `get_landing_sequence` and `get_atc_log`, and step in with `approach_instruction`: move an arrival up or down the order, slow it, hold it, release it, send it direct to final, or send it around. `get_conflicts` predicts losses of separation, and `separation_minima` gives the wake spacing for a pair of types.

Flights you spawn with `hold_for_clearances` (departures) or `hold_for_clearance` (arrivals) stay yours. The controllers count them as traffic but never clear or instruct them.

## Things to ask

- "Run the airline schedule at LKPR, at most 8 aircraft at once."
- "What's on the departure board? Who's delayed, and why?"
- "Show me the landing sequence for runway 24 and the last ATC instructions."
- "Put a Smartwings 737 on stand C22 departing runway 24, and let me give the clearances."
- "Clear TVS1860 to push back." … "Taxi." … "Line up and wait." … "Cleared for take-off."
- "Slow down AFR1094 and move it behind KLM1008."
- "Stop the schedule and let the aircraft finish their flights." (`stop_schedule`)
- "Stop the schedule and remove everything now." (`stop_schedule` with `remove: true`)

## Limits

- At most 32 aircraft of ours at once, and at most 24 from a schedule.
- The airport must be loaded around your aircraft. Traffic at an airport far from you can't be spawned.
- En route aircraft far from you show their last known position until they come within about 43 NM.
- The simulation rate and pause aren't followed yet: traffic runs on real time.

## The tools

| Purpose | Tools |
|---|---|
| Run a schedule | [`start_schedule`](/docs/mcp-tools-simconnect#start_schedule), [`get_schedule`](/docs/mcp-tools-simconnect#get_schedule), [`stop_schedule`](/docs/mcp-tools-simconnect#stop_schedule) |
| Add and clear aircraft by hand | [`spawn_departure`](/docs/mcp-tools-simconnect#spawn_departure), [`spawn_arrival`](/docs/mcp-tools-simconnect#spawn_arrival), [`atc_clearance`](/docs/mcp-tools-simconnect#atc_clearance), [`list_our_traffic`](/docs/mcp-tools-simconnect#list_our_traffic) |
| Approach and tower | [`get_landing_sequence`](/docs/mcp-tools-simconnect#get_landing_sequence), [`approach_instruction`](/docs/mcp-tools-simconnect#approach_instruction), [`get_atc_log`](/docs/mcp-tools-simconnect#get_atc_log), [`get_conflicts`](/docs/mcp-tools-simconnect#get_conflicts), [`separation_minima`](/docs/mcp-tools-simconnect#separation_minima) |
| Look around | [`get_traffic_picture`](/docs/mcp-tools-simconnect#get_traffic_picture), [`list_aircraft_models`](/docs/mcp-tools-simconnect#list_aircraft_models), [`generate_schedule`](/docs/mcp-tools-simconnect#generate_schedule) |
| The airport | [`get_active_runway`](/docs/mcp-tools-simconnect#get_active_runway), [`get_atis`](/docs/mcp-tools-simconnect#get_atis), [`plan_taxi_route`](/docs/mcp-tools-simconnect#plan_taxi_route), [`find_stands`](/docs/mcp-tools-simconnect#find_stands), [`get_airport_procedures`](/docs/mcp-tools-simconnect#get_airport_procedures) |

For a worked example with the real tool results, see [Scenario 4 in Examples](/docs/examples).
