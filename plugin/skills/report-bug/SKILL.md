---
name: report-bug
description: Write up a bug a tester just saw in the simulator or its traffic, with the sim's state captured right now, into a report file they can send. Use whenever someone says something looks wrong, broke, jumped, disappeared or got stuck, or that they want to report a problem, even if they don't say "bug".
allowed-tools: mcp__plugin_simconnect_simconnect__get_sim_state, mcp__plugin_simconnect_simconnect__get_aircraft_systems, mcp__plugin_simconnect_simconnect__get_nearest_airport, mcp__plugin_simconnect_simconnect__get_nearby_traffic, mcp__plugin_simconnect_simconnect__get_traffic_status, mcp__plugin_simconnect_simconnect__get_traffic_picture, mcp__plugin_simconnect_simconnect__get_conflicts, mcp__plugin_simconnect_simconnect__get_atc_log, mcp__plugin_simconnect_simconnect__screenshot
---

# Report a bug

The person is probably a tester, not a developer. Keep the conversation short and in everyday words. What the sim shows changes by the second, so capture first and ask afterwards.

## 1. Capture now

Call these straight away, before asking anything. Any of them may fail or return nothing; note that and carry on.

- `get_sim_state`, `get_aircraft_systems`, `get_nearest_airport`
- `get_nearby_traffic`
- `get_traffic_status`, `get_traffic_picture`, `get_conflicts`, `get_atc_log`
- `screenshot`, which shows only the simulator window

Only read. Don't move, spawn or remove anything, because that would change the evidence and could confuse what they are looking at.

## 2. Ask

Ask at most three short questions, skipping what they have already said:

- What did you see, and what did you expect instead?
- Which aircraft or call sign, if you noticed one?
- Did it happen once, or can you make it happen again?

## 3. Write the report

Save it to their Desktop as `sim-bug-<YYYY-MM-DD-HHMM>.md`, or to the current folder if the Desktop can't be written. Use this layout:

```markdown
# <one line: what went wrong>

**When:** <date and time, with time zone>
**Where:** <airport, position>
**Aircraft:** <title, call sign>
**Simulator:** <version>

## What happened
<their words>

## What was expected
<their words>

## Can it be repeated
<once / sometimes / always, with the steps if known>

## Captured from the simulator
<one short section per tool: the values that matter, then the raw answer in a code block>
```

Separate what they said from what the sim reported, and don't guess at the cause. A wrong guess sends whoever reads the report the wrong way, while the raw values let them work it out.

## 4. Finish

Describe what the screenshot shows in the report, because the file holds text only. Tell them where the file is and to send it to whoever coordinates the testing, along with any screen recording they have. Mention that the sim's data can be cleared by a restart, so they shouldn't restart just for the report.

## Reply

Keep it short: the answer and what they need to do next. No step-by-step narration or explaining yourself unless asked.
