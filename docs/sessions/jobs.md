# The background jobs of a session

A session derives its background jobs from the stream, the same way it derives
the cost and the token counts. Claude Code pushes three task events for each
job, and it carries the job output in a file or inline in a `tool_result`; see
[protocol/jobs.md](../protocol/jobs.md). `apply` turns them into `Job` records:

- `task_started` adds a `Job` with the status `running`.
- `task_updated` finds the `Job` by its id, and sets the status from the patch.
- `task_notification` finds the `Job` by its id, stops it, and gives it the
  summary and the output path.

A `Job` holds the id, the description, the task type, the command, the summary,
the output path, the status, and the start and end times. The status is one of
`running`, `done`, `failed`, or `killed`.
The start event makes a job `running`. A later event moves it to one terminal
status:

| Wire value | Job status |
|---|---|
| `completed` | done |
| `failed` | failed |
| `killed`, `stopped` | killed |
| any other | running |

An unknown status stays `running`, so a new Claude Code value cannot lose a job.
A notification does not change a job that already reached a terminal status.

`Snapshot.Jobs` carries every job in start order, for the life of the session.
`Snapshot.RunningJobs` counts the jobs that still run. The interface shows both;
see [../tui/sessions.md](../tui/sessions.md).

### The command and the output

The two events above do not carry the command or the output. Both come from the
message blocks around them, so `applyJobBlocks` reads two block types:

- A `Bash` `tool_use` whose input holds `run_in_background`. The session keeps
  the command under the block id, until `task_started` names that id.
- The `tool_result` of that block, which the session reads two ways; see
  [../protocol/jobs.md](../protocol/jobs.md) for the two shapes.

An older Claude Code names an external file: the `tool_result` text holds
`Output is being written to: <path>.`, and the session keeps that path on the
job. `task_started` can arrive before or after this result, so the session holds
a path that arrives first, and `task_started` takes it. The `output_file` of the
notification then replaces the parsed path, unless it is empty, which is what a
killed job sends.

The current Claude Code names no file. The `tool_result` carries the whole
output inline, so the multiplexer writes it to the job's file itself, the same
file a local agent gets. `applyLaunchResult` returns the write, and
`applyJobBlocks` runs it once the lock is released. The write truncates the
file, because one `tool_result` is the whole output, so a replay on restart
cannot stack it.

### A generated file, when Claude Code names none

A job whose output the multiplexer writes gets a path under the session state
directory, at `<dir>/tasks/<task id>.output`, the shape `ReadOutput` accepts.
`generatedOutputPath` builds it. A background bash job takes this path when its
inline result arrives; a `local_agent` takes it at `task_started`.

A local agent is a job with `task_type` `local_agent`; see
[../protocol/jobs.md](../protocol/jobs.md). Claude Code writes no file for it, and
its `task_notification` carries an empty `output_file`. So the multiplexer writes
the file, from the agent's turns.

The agent's turns stream inline on the parent output, each with a
`parent_tool_use_id`. The multiplexer renders each such turn and appends it to
the agent job's output file, so the jobs dialog reads it the same way it reads a
background bash job. The turns do not go to the session pane; see
[../tui/output.md](../tui/output.md).

A turn that arrives before `task_started` names the job has no job yet, so the
session holds it under the parent id, and the flush writes it in order once the
job registers. `CaptureAgentTurn` and `FlushAgentTurns` do this, both from the
one manager pump, so every write to a job file stays sequential and in order.

### Reading the output

`session.ReadOutput(job)` opens the output path and returns the last
`MaxJobOutput` bytes, which is 256 KiB. A longer file gives its end, because a
job that prints for hours must not fill memory. A file that is absent or empty
returns an empty string and no error, because a job that has printed nothing yet
is not a failure.

The path comes off the child stream, so `ReadOutput` holds it to the shape
Claude Code writes a job to before it opens anything:

- The path must be absolute.
- Its last element must be `<job id>.output`.
- Its parent directory must be named `tasks`.

A path that fails any of these returns `ErrBadOutputPath`. A job that has no
path yet returns `ErrNoOutputPath`.

Nothing reads the file on its own. The interface reads it only while you have
that job open, so a job nobody looks at costs nothing. See
[../tui/sessions.md](../tui/sessions.md).
