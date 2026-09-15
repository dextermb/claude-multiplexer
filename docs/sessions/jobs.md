# The background jobs of a session

A session derives its background jobs from the stream, the same way it derives
the cost and the token counts. Claude Code pushes three task events for each
job, and it writes the job output to a file; see
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

### The command and the output path

The two events above do not carry the command or the output path. Both come
from the message blocks around them, so `applyJobBlocks` reads two block types:

- A `Bash` `tool_use` whose input holds `run_in_background`. The session keeps
  the command under the block id, until `task_started` names that id.
- The `tool_result` of that block. Claude Code answers a background `Bash` with
  the text `Output is being written to: <path>.`, and the session keeps the
  path on the job.

`task_started` can arrive before or after the `tool_result`. So the session
holds a path that arrives first, and `task_started` takes it. The `output_file`
of the notification then replaces the parsed path, unless it is empty, which is
what a killed job sends.

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
