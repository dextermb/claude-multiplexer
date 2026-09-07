# The scheduler

A schedule is a durable, recurring task. The multiplexer runs it on its own
clock. A schedule survives a restart, because it lives on disk, not in a session.

The example is a session that polls a website every few minutes. Any session
creates the schedule with `create_schedule`, and the manager runs it from then
on. See [mcp/tools.md](mcp/tools.md) for the five tools.

## Why the manager owns the clock

A recurring task needs a process that stays alive to run it. A session ends, so a
session cannot be that process. The settings file does not run, so it cannot be
that process. The manager is the one long-lived process behind the interface, so
the clock lives there. See [manager.md](manager.md).

A schedule record holds only provider-neutral fields: a directory, a prompt, a
model name, a permission mode. It never names Claude. On each run the manager
calls the same `Spawn` path that `create_session` uses. So the scheduler carries
over to a second session provider with no change.

## Where a schedule lives

Each schedule is one JSON file in the state directory, next to the sessions:

```
~/.claude-multiplexer/
  sessions/
    <name>/ meta.json, transcript.jsonl, mcp.json
  schedules/
    <name>.json
```

The state directory is the one `--root` sets, the same directory that holds the
sessions. See [config.md](config.md). The manager scans `schedules/` at start,
and holds the schedules in memory. Each change writes the one file, so two
schedules never share a write.

## The record

| Field | What it holds |
|---|---|
| `name` | The unique name of the schedule |
| `cron` | A 5-field cron expression, in local time |
| `dir` | The directory a run works in |
| `prompt` | The prompt each run sends |
| `session` | The reuse target; empty means spawn a fresh session |
| `model` | The model of the session a run starts |
| `permission_mode` | The permission mode of that session |
| `effort` | The effort level of that session |
| `control` | True gives that session the control grant; only a control caller may set it |
| `enabled` | True runs the schedule; false pauses it |
| `created_at` | When the schedule was created |
| `last_run` | When the schedule last ran |
| `last_session` | The session the last run started |

A schedule refers to a session by name, through `session` and `last_session`. The
reference is loose. The named session may not exist, and a missing session is not
an error. Nothing refers back to a schedule, so a session does not know that a
schedule started it.

## The two run modes

The `session` field picks the mode.

- **Spawn mode** (empty `session`). Each run starts a fresh session in the
  directory. The task begins with a clean context every time. This mode fits a
  poll that only reports a change.
- **Reuse mode** (named `session`). Each run queues the prompt into that one
  session, so the session keeps its memory across runs. This mode fits a poll
  that compares against what it saw last time. When the session is not live, the
  manager resumes the stored session, or starts a fresh one under that name.

Each run marks the session pane with `← prompt from schedule:<name>`, the same
way `send_message` marks a prompt, so the human sees where the prompt came from.

## The clock

One goroutine wakes every 30 seconds and runs each schedule that is due. The
manager starts it once, after the MCP server, because a run spawns a session and
a session needs the MCP address.

A schedule is due when it is enabled and `now` is at or after the next cron time.
The manager computes the next time from `last_run`, or from `created_at` when the
schedule has not run yet. Three rules protect the machine:

- **A spawn cap.** One tick starts at most 3 fresh sessions. When more spawn-mode
  schedules are due at once (for example after downtime), the tick leaves the rest
  for the next ticks. A left schedule keeps its old `last_run`, so it stays due.
- **No overlap in spawn mode.** When the last spawned session of a schedule is
  still live, the tick skips that run. This stops a slow task from piling
  sessions on top of each other. The skip publishes a notice.
- **One make-up run after downtime.** The next time comes from `last_run`. When
  the machine was off past a slot, that slot is in the past, so the schedule runs
  once, and then `last_run` moves to now. The next tick sees the following slot in
  the future, so no burst follows.

A schedule does not expire. It runs until the human pauses or deletes it. This is
the difference from an in-session cron, which lapses on its own.

## The cron dialect

A schedule takes a standard 5-field cron expression, in local time:

```
minute hour day-of-month month day-of-week
```

For example, `*/5 * * * *` runs every 5 minutes, and `0 9 * * 1-5` runs at 09:00
on the weekdays. The parser is `robfig/cron/v3`. The `create_schedule` tool
rejects an expression the parser does not accept, so a bad cron never writes a
file.

## The workflows

- **Create.** Any session calls `create_schedule` with a cron, a directory, and a
  prompt. The manager validates all three, writes the file, and returns the name.
  Only a control caller may set the `control` field, and the manager drops it from
  a plain caller, so a plain session cannot reach the control grant this way.
- **Run.** The clock runs the schedule, in spawn mode or reuse mode. `run_schedule`
  runs a schedule at once, whatever its cron says, to test it.
- **Pause and resume.** `set_schedule_enabled` flips `enabled`. A paused schedule
  stays on disk, so a resume needs no re-entry of the cron and the prompt.
- **Delete.** `delete_schedule` removes the file. A session it started is left
  alone.

## What is not here yet

The first version has no panel in the interface. A session reads the schedules
with `list_schedules`. A schedule stores a raw prompt only, not a
reference to a template. See [templates.md](templates.md) for the template system.
