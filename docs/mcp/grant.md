# The grant, and what a tool refuses

A session gets `rename_session`, `list_sessions`, `get_messages`, `list_jobs`,
`get_config_path`, `get_template_path`, `set_editor`, `unset_editor`,
`set_block_cap`, `unset_block_cap`, `set_working_dir`, and `unset_working_dir`
always. It gets the five control tools **only** when it is started with
control.

- In the new session form, set the `Control` field to `yes`.
- On the command line, `multiplexer --dir <path> --control`.

The grant is stored in `meta.json`, so a resumed session keeps what it had.

A session without the grant is never offered the control tools. They are absent
from its `tools/list` answer, so the model does not know they exist, and it
cannot call one by name.

**A session with control can stop the work you are reading.** The interface
marks such a row with `⇄` in the sidebar, and the session bar names `control`
next to the model. See [tui/sessions.md](../tui/sessions.md).

## What a tool refuses

- An unknown token. The request never reaches a tool.
- An unknown session name.
- `send_message` to the calling session itself. That prompt would run after the
  turn that sent it, which is a loop with no human in it.
- `stop_session` on the calling session itself. The child would die before the
  tool result returned.
- `archive_session` on a session that still runs. Stop it first.
- `create_session` with no path, or with a path that is not a directory that
  exists.
- `stop_job` with no job id, or with a job id that no running job holds.

Unlike `send_message` and `stop_session`, `stop_job` may name the calling
session, because a session may stop its own background job. A self target
interrupts the caller's own turn.

## Two agents can talk in a circle

`send_message` returns as soon as the prompt is on the queue. It does not wait
for the answer, so no agent blocks on another and two agents cannot deadlock.

They can, however, trade prompts for as long as they have money. Nothing stops
session A prompting B while B prompts A. The multiplexer does not guard against
this. It makes the traffic visible instead: every prompt that came from another
session is marked in the pane of the session that received it.

```
← prompt from docs
› list the files in the repository
```

Watch for that line, and press `s x` to interrupt a session that runs away. See
[tui/keys.md](../tui/keys.md).
