# Background jobs

Claude Code can run a shell command in the background. This page is what the
wire carries for one, and where its output goes. For the `Job` records the
session builds from it, see [../sessions.md](../sessions.md).

A `Bash` tool call with `run_in_background: true` in its input starts a job. The
job keeps running after the turn ends. Claude Code then pushes three `system`
events for the job, each with a `task_id`. This was proven against Claude Code
2.1.176.

| Subtype | Fields | Meaning |
|---|---|---|
| `task_started` | `task_id`, `tool_use_id`, `description`, `task_type` | The job started. |
| `task_updated` | `task_id`, `patch` (`status`, `end_time`) | The status changed. |
| `task_notification` | `task_id`, `tool_use_id`, `status`, `summary`, `output_file` | The job stopped. |

The `patch.status` values seen are `completed` (exit 0) and `killed`. A
`task_notification` on a kill carries the status `stopped`. `Event.Task` holds
the union of the three shapes, with a `Patch` sub-struct for `task_updated`.

The `task_id` is also the background shell id. The launch result below names
`b14gccuz1`, and the `task_id` of the same job is `b14gccuz1`.

The session turns these events into `Job` records; see
[../sessions.md](../sessions.md).

## The job output is a file

Claude Code writes the output of a background job to a file, and it names the
path twice. The `tool_result` of the launching `Bash` call gives it at the start:

```
Command running in background with ID: b14gccuz1. Output is being written to:
/tmp/claude-501/<dir>/<session_id>/tasks/b14gccuz1.output. You will be notified
when it completes. To check interim output, use Read on that file path.
```

The `task_notification` then repeats it in `output_file`, with the exit code in
`summary`:

```json
{"status":"completed",
 "output_file":".../tasks/b14gccuz1.output",
 "summary":"Background command \"...\" completed (exit code 0)"}
```

A job that is killed sends an empty `output_file`, so the path from the launch
result is the one that always holds. `protocol.BackgroundOutputPath` reads it,
and `Block.BackgroundBash` reads the command of a `Bash` call that carries
`run_in_background`. The multiplexer reads the file itself; see
[../sessions.md](../sessions.md).

The `BashOutput` tool result also carries `<status>`, `<exit_code>`, and
`<output>` tags, but the multiplexer does not read them. Two reasons. The push
events carry the same status in a structured form. And the model does not call
that tool — Claude Code tells it to use `Read` on the output file instead, and a
probe that asked for `BashOutput` by name twice still got `Read`.
