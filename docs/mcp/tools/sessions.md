# The session tools

These tools name a session, read it, and — with the control grant — drive it. A
tool marked `open` goes to every session, and a tool marked `control` goes only
to a session that holds the grant. See [../grant.md](../grant.md).

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `rename_session` | `title` | Sets the title of the calling session. An empty title clears it. | open |
| `list_sessions` | `stopped`, `archived`, `last_active` | The sessions that run now: name, title, directory, state, model, turns, cost, last-active time, and the archive flag. `stopped` adds the stored sessions, and `archived` adds the archived ones. `last_active` drops a stored or archived session older than its window. | open |
| `list_inactive_sessions` | — | The stored sessions that do not run now and are not archived. | open |
| `list_archived_sessions` | — | The archived sessions. | open |
| `get_messages` | `session`, `limit` | The recent messages of a session, oldest last. 20 by default, 200 at most. | open |
| `list_jobs` | `session` | The background jobs of a session: id, description, task type, and status. An empty session means the caller. | open |
| `send_message` | `session`, `text` | Queues a prompt for another session, and returns the queue length. | control |
| `stop_session` | `session` | Ends another child in a clean way. Its transcript is kept. | control |
| `archive_session` | `session`, `restore` | Takes a stopped session out of the list, or with `restore` brings it back. | control |
| `stop_when_idle` | `stop`, `archive` | Arms this session to stop itself the next time it is idle, and to archive itself after the stop when `archive` is true. | open |
| `create_session` | `path`, `name`, `model`, `effort`, `profile` | Starts a new session in a directory. Returns the name it takes. | control |
| `stop_job` | `session`, `job` | Interrupts a session and asks it to kill one background job. An empty session means the caller. | control |

### The three list tools

`list_sessions` answers the sessions that run now. The `stopped` flag adds the
stored sessions that do not run, and the `archived` flag adds the archived ones,
so one call reaches every category. `list_inactive_sessions` and
`list_archived_sessions` name one category each, and take no argument.

The `last_active` window keeps the list short. It drops a stored or archived
session that took its last turn before the window: `1d`, `1w`, `1m`, `1y`, or
`unset` for no limit. The default is `1d`. A running session is always returned,
because it is active now, and a session that never took a turn is always
returned, because it has no last-active time. The window has no effect until
`stopped` or `archived` adds those categories.

### Reading a session

`get_messages` reads the transcript on disk, so it answers for a stored session
as well as a live one. It returns one entry for each user prompt, each assistant
message, and each result. A tool call becomes the short line `[used Bash]`,
because the whole input of a tool call is large and it is rarely what a reader
wants.

`list_jobs` reads the background jobs of a session; see
[../../sessions.md](../../sessions.md). It reads the live session, so it returns
an empty list for a stored session, which has no running jobs.

### Starting a session

`create_session` takes a directory path and an optional name. The directory
must exist. The manager makes the name unique, and it falls back to the last
element of the path when the name is empty, so the tool returns the real name
the session takes. The new session starts without the control grant.

The tool also takes an optional `model` and an optional `effort`. The `effort`
is `low`, `medium`, `high`, `xhigh`, or `max`. An empty field takes the default,
the same as the new session form.

The optional `profile` names the open tools the new session carries. It is
`minimal` or `standard`, and an empty field takes the `defaultToolProfile`
setting. A name that is not a profile fails the call. See
[../profiles.md](../profiles.md).

The manager writes the name of the caller into the record of the new session,
as its creator. The sidebar groups a session under the control session that
created it, and the record keeps that group after a restart. See
[../../tui/sessions.md](../../tui/sessions.md) and
[../../manager.md](../../manager.md).

### Stopping a session and a job

`stop_session` rejects a self-target, because a direct stop kills the process
that runs the tool call, and the turn never returns. `stop_when_idle` is the
deferred path a session uses on itself. The session arms the action, and the
manager runs it later, when the session is idle. See
[../../sessions.md](../../sessions.md) for the idle point and the action.

`stop_job` cannot reach a Claude Code background shell directly, because the
shell runs inside the child. So the tool interrupts the current turn of the
owning session, and it queues an instruction to run `KillShell` on the exact
shell. The interrupt ends the turn at once, so the instruction runs on the next
turn. The tool marks the pane with `← stop job <id> from <caller>`, the same way
`send_message` marks a prompt. It finds the job by its id first, so it never
interrupts a turn for a job that does not exist or already stopped.
