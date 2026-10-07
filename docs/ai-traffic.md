---
title: AI Traffic & ATC
description: Bring your airport to life from an AI chat — AI departures and arrivals of your own with ground services, a tower and approach controller that clear and sequence them, an airline schedule or real-world traffic, and airliners around you in cruise.
order: 1
section: guides
---

SimConnect MCP can fill the airport you're flying from with traffic of its own and control it the way a tower and approach controller would. You ask in plain language, for example "run the airline schedule at Prague". The server then puts aircraft into the simulator and flies them from stand to runway and from their origin to the stand, clearing and spacing them as it goes.

It runs the traffic engine of the [mrlm-net/simconnect](https://github.com/mrlm-net/simconnect) Go library, `pkg/traffic/world`, the same engine as the library's airport map. Every tool mentioned here is described in full in the [SimConnect Mode reference](/docs/mcp-tools-simconnect).

> **Beta.** The traffic tools change the simulator: they add and remove aircraft. They're new and still changing between releases.

## What you need

- **Windows with MSFS 2020 or 2024 running**, and the server in `simconnect` or `both` mode (see [Getting Started](/docs/getting-started)).
- **Your aircraft at or near the airport.** The simulator only serves an airport's stands, taxiways and procedures once it has loaded the airport around the user aircraft.
- **AI aircraft models.** Each flight uses an installed aircraft of its type, in its airline's livery where one is installed. The more airline liveries you have, for example from an AI traffic package, the more the traffic looks like the real thing. `list_aircraft_models` shows what's installed.
- **Ground vehicles.** Tugs, fuel trucks, stairs and GPUs use GSX's models when GSX is installed, else MSFS's own.

The engine starts with the first traffic tool you use. If the simulator restarts, it starts again by itself and sets your schedule, real traffic and corridor again.

## Ways to add traffic

**By hand**, one aircraft at a time. `spawn_departure` puts an aircraft on a stand and `spawn_arrival` puts one on its arrival route (STAR). The runway, stand, SID or STAR and model are chosen for you unless you name them. By default each flight waits for your clearances: pushback, taxi, cross, line up, take off. You give them with `atc_clearance`, so you can play the controller yourself. Spawn with `hold_for_clearances: false`, or give `atc_clearance` `manual` with `on: false`, to let the engine's controllers handle it.

**By schedule.** `start_schedule` runs a realistic airline day at your airports, with light aircraft in the circuit by day, and the flights go by themselves. It uses the same generator as `generate_schedule`. `get_schedule` shows the departure and arrival boards, and `stop_schedule` ends the run.

**At chosen times.** `add_flights` adds flights to the schedule at the times you want: an arrival a few minutes before your ETA, a departure just after your off-block. Start the schedule with `generator: false` to fly only those.

**Real-world traffic.** `set_real_traffic` switches an airport from the timetable to the aircraft a feed such as ADS-B sees, which you pass with `observe_traffic`. Parked aircraft go on a stand near where they're seen and wait until their departure is seen. Departures push, and arrivals appear on their track and join a STAR. Unknown origins and destinations stay unknown.

**Around you in cruise.** `set_traffic_corridor` takes your route ahead, level and speed, and keeps a few airliners around you. One flies ahead the same way, one comes the other way and one crosses, each 1,000 or 2,000 ft above or below. They're replaced once they're far behind.

## What each flight gets

| Flight | What happens |
|---|---|
| **Departure** | Appears on a stand that fits it before its departure time (STD). Stand services come from the airport's fleet: a fuel truck, plus boarding stairs and a GPU at a remote stand. It pushes back behind a tug, taxis the planned route, is cleared by the tower to line up and take off, and flies its SID. Its crew talks to delivery, ground and tower. |
| **Arrival** | Comes in en route and joins its STAR. The approach controller sequences it, and it flies the approach, lands, vacates and taxis to a stand. |
| **Turnaround** | Stays on its stand and departs again as its later flight. |
| **Overflight** | Crosses the area at cruise level. |

Each airport has a fleet of tugs, fuel trucks, stairs and GPUs sized by its stands. A departure waits for a free tug. Fuelling, stairs and the GPU are left out when it's too late for them.

## The tower and the approach controller

At each airport it works, the engine runs:

- **A tower per runway**, which clears line-ups, take-offs and runway crossings, and sends an arrival around when the runway won't be free.
- **A landing sequence per runway end**, which orders the arrivals on the final by wake, keeps at least the minimum spacing (more in low visibility), and has them lose delays with speed, vectors and holds.
- **Separation**, with conflicts predicted and resolved for ours, and TCAS II on each of ours (`get_tcas`).
- **The radio**: every instruction and readback, by position and frequency.

You can watch them with `get_landing_sequence`, `get_conflicts` and `get_atc_log`, and step in with `approach_instruction`. With it you can move an arrival up or down the order, slow it or set its speed, hold it (at its STAR fix or at a point), release it, send it direct, have it join the final at a point, or send it around. `get_traffic_airport_info` shows the runways in use and the ATIS the engine's controllers work with.

**Your own flight.** The engine never controls or calls your aircraft. When your ATC clears you onto a runway, `set_player_clearance` tells the engine, and while you line up, take off or land there, none of its traffic is cleared onto it. Landing, you're put in the landing sequence and the traffic fits around you.

## Things to ask

- "Run the airline schedule at LKPR, at most 8 aircraft at once."
- "What's on the departure board? Who's delayed, and why?"
- "Show me the landing sequence for runway 24 and the last radio calls."
- "Put a Smartwings 737 on stand C22 departing runway 24, and let me give the clearances."
- "Clear TVS1860 to push back." … "Taxi." … "Line up and wait." … "Cleared for take-off."
- "Slow down AFR1094 and move it behind KLM1008."
- "Add an arrival from Munich 20 minutes from now."
- "I'm cruising at FL360 along this route: put some traffic around me."
- "Stop the schedule and let the aircraft finish their flights." (`stop_schedule`)
- "Stop the schedule and remove everything now." (`stop_schedule` with `remove: true`)

## Limits

- The airport must be loaded around your aircraft. Traffic at an airport far from you can't be spawned.
- En route aircraft the schedule created aren't on the engine's list of ours. When the server stops they stay until its connection closes.
- `observe_traffic` takes sightings you pass to it. The server doesn't fetch an ADS-B feed itself.

## The tools

| Purpose | Tools |
|---|---|
| Run a schedule | [`start_schedule`](/docs/mcp-tools-simconnect#start_schedule), [`get_schedule`](/docs/mcp-tools-simconnect#get_schedule), [`stop_schedule`](/docs/mcp-tools-simconnect#stop_schedule), [`add_flights`](/docs/mcp-tools-simconnect#add_flights) |
| Real traffic and cruise | [`set_real_traffic`](/docs/mcp-tools-simconnect#set_real_traffic), [`observe_traffic`](/docs/mcp-tools-simconnect#observe_traffic), [`set_traffic_corridor`](/docs/mcp-tools-simconnect#set_traffic_corridor) |
| Add and clear aircraft by hand | [`spawn_departure`](/docs/mcp-tools-simconnect#spawn_departure), [`spawn_arrival`](/docs/mcp-tools-simconnect#spawn_arrival), [`atc_clearance`](/docs/mcp-tools-simconnect#atc_clearance), [`list_our_traffic`](/docs/mcp-tools-simconnect#list_our_traffic) |
| Approach and tower | [`get_tcas`](/docs/mcp-tools-simconnect#get_tcas), [`get_landing_sequence`](/docs/mcp-tools-simconnect#get_landing_sequence), [`approach_instruction`](/docs/mcp-tools-simconnect#approach_instruction), [`get_atc_log`](/docs/mcp-tools-simconnect#get_atc_log), [`get_conflicts`](/docs/mcp-tools-simconnect#get_conflicts), [`separation_minima`](/docs/mcp-tools-simconnect#separation_minima), [`set_player_clearance`](/docs/mcp-tools-simconnect#set_player_clearance) |
| Look around | [`get_traffic_picture`](/docs/mcp-tools-simconnect#get_traffic_picture), [`get_traffic_status`](/docs/mcp-tools-simconnect#get_traffic_status), [`list_aircraft_models`](/docs/mcp-tools-simconnect#list_aircraft_models), [`generate_schedule`](/docs/mcp-tools-simconnect#generate_schedule) |
| The airport | [`get_traffic_airport_info`](/docs/mcp-tools-simconnect#get_traffic_airport_info), [`get_active_runway`](/docs/mcp-tools-simconnect#get_active_runway), [`get_atis`](/docs/mcp-tools-simconnect#get_atis), [`plan_taxi_route`](/docs/mcp-tools-simconnect#plan_taxi_route), [`find_stands`](/docs/mcp-tools-simconnect#find_stands), [`get_airport_procedures`](/docs/mcp-tools-simconnect#get_airport_procedures) |

For a worked example, see [Scenario 4 in Examples](/docs/examples).
