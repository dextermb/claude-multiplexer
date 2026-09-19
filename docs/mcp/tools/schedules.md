# The schedule tools

These tools create and drive durable tasks. The manager runs each schedule on its
own clock, so a schedule survives a restart. A schedule runs on a cron, or once
after a time. See [../../scheduler.md](../../scheduler.md) for the record, the two
run modes, and the clock.

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `create_schedule` | `cron`, `run_after`, `dir`, `prompt`, `name`, `session`, `model`, `permission_mode`, `effort`, `control` | Creates a durable schedule. Give a `cron` for a recurring run, or leave `cron` empty and set `run_after` for a one-off. Returns the schedule record. | open |
| `update_schedule` | `name`, `cron`, `run_after`, `dir`, `prompt`, `session`, `model`, `permission_mode`, `effort`, `control` | Changes the fields it is sent, and leaves the rest. An empty string clears an optional field. A `cron` switches to recurring; a `run_after` switches to one-off. Returns the schedule record. | open |
| `list_schedules` | — | Every schedule: name, cron, run-after time, directory, mode, model, permission mode, effort, the control grant, and the last run. | open |
| `delete_schedule` | `name` | Removes a schedule. A session it already started is left alone. | open |
| `set_schedule_enabled` | `name`, `enabled` | Turns a schedule on or off. A paused schedule stays on disk. | open |
| `run_schedule` | `name` | Runs a schedule now, whatever its cron or run-after time says. A one-off schedule pauses after it runs. Returns the session name. | open |
| `get_schedule_path` | — | The directory the multiplexer writes schedule records to, one JSON file per schedule. | open |

Every session may create and drive schedules, so the tools are open. The
`control` field is the one exception. It gives the spawned session the control
grant, so only a caller that holds the control grant may set it. The multiplexer
drops the flag from a caller without the grant, so a plain session cannot reach
the control grant through a schedule.

`get_schedule_path` answers with `dir`, the directory the multiplexer writes
schedule records to, one JSON file per schedule. It only reads.
