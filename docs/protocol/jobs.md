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

## A newer Claude Code carries the output inline

A later Claude Code sends a different shape, seen in the session
`claude-multiplexer-84`. The `Bash` `tool_use` holds no `run_in_background`, the
`task_notification` `output_file` is empty, and no `tool_result` names a file.
Instead the `tool_result` carries the whole output inline:

```json
{"type":"tool_result","tool_use_id":"toolu_01...",
 "content":"ok  github.com/.../manager\t2.522s\nok  github.com/.../mcp\t0.903s",
 "is_error":false}
```

The events still arrive in one order: `tool_use`, `task_started`,
`task_notification` (completed), then the `tool_result`. So the job is always
registered when the result arrives. The multiplexer captures the inline content
to the job's file, the same file a `local_agent` writes; see
[../sessions/jobs.md](../sessions/jobs.md).

## A local agent is a job too

A `Task` (or `Agent`) tool call starts a local agent. It is also a job, so it
arrives as the same three `system` events, with `task_type` `local_agent`
instead of `local_bash`. The `task_started` event also carries `subagent_type`
(for example `Explore`) and the full `prompt`.

The agent runs inside the parent Claude Code process, so it streams its turns on
the same output as the parent. Each turn of the agent carries a top-level
`parent_tool_use_id`, set to the `tool_use_id` of the `Task` call. A top-level
turn carries `parent_tool_use_id: null`, or omits the field.

```json
{"type":"assistant","parent_tool_use_id":"toolu_01",
 "message":{"role":"assistant","content":[{"type":"tool_use","name":"Read", ...}]}}
```

`protocol.Event.ParentToolUseID` holds this id, and `Event.HasParent` reports it.
The turns of an agent belong to the agent's job, not the session pane; see
[../sessions/jobs.md](../sessions/jobs.md) and [../tui/output.md](../tui/output.md).

Claude Code writes no output file for an agent. The agent's `task_notification`
carries an empty `output_file`, and there is no launching `Bash` call to name a
path. So the multiplexer captures the agent's turns and writes the file itself;
see [../sessions/jobs.md](../sessions/jobs.md). The agent's final report returns
as the `Task` tool_result on a parent turn, so the pane keeps it.
