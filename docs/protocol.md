# The stream-json transport

`multiplexier` drives each Claude Code session as a child process in headless
mode. The two processes exchange one JSON object for each line. The package
`internal/protocol` holds the encoder and the decoder.

## Why a child process, and not something else

Two other shapes were considered and rejected.

**A pseudo-terminal running the real Claude Code interface.** It would show the
true interface, but reading it means emulating a terminal and scraping the
screen. Scraping breaks whenever the layout changes, and it gives no structured
events, so the state of a session could only be guessed.

**A direct Anthropic API client.** It would mean re-implementing the agent loop,
the tools, the permission model, and the settings that Claude Code already has.
The multiplexer would then be a second, worse Claude Code.

A child process in headless mode keeps all of Claude Code and adds a documented
event stream. The cost is that the multiplexer is bound to the shape of that
stream, which is why the decoder never fails on a line it does not know.

## The command line

```
claude -p
  --output-format stream-json
  --input-format  stream-json
  --verbose
  --permission-mode <mode>          auto by default
  [--model <model>]
  [--effort <level>]                low, medium, high, xhigh, or max
  [--allowedTools A,B]
  [--disallowedTools A,B]
  [--session-id <uuid>]
  [--resume <session_id>]
  [--include-partial-messages]
  [--replay-user-messages]
```

`--verbose` is mandatory. Claude Code refuses `--output-format stream-json`
under `-p` without it.

The permission mode is one of `acceptEdits`, `auto`, `bypassPermissions`,
`default`, `dontAsk`, or `plan`. The mode sets the start value, but it is not
fixed for the session: a control request changes it later (see below).

The effort level sets a thinking-token budget. It is `low`, `medium`, `high`,
`xhigh`, or `max`. Claude Code has no `--effort` default, so the flag is left
off unless the session asks for a level.

## The init event arrives after the first input

Claude Code emits `system`/`init` only after it reads the first input line. A
supervisor that waits for `init` before it sends the first prompt deadlocks.
Both sides wait, and nothing moves.

So the session sends its first prompt while its state is still `starting`. It
waits for `idle` only for the second prompt and the ones after it. See
[sessions.md](./sessions.md) for the state machine.

## The prompt comes back to you

With `--replay-user-messages` the child writes each prompt back on stdout as
soon as it reads it. The echo carries `"isReplay": true`:

```json
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello"}]},"isReplay":true,"uuid":"…"}
```

`Event.IsReplay` carries that marker. It is the only way to tell a prompt from a
tool result, because both arrive as `type: user`.

The multiplexer turns the flag on for every session it supervises. So the prompt
lands in the transcript with everything else, in the right order, and a replay
of the file shows the conversation and not only one half of it.

## Claude Code reads its own slash commands

A prompt that starts with `/name` is read by Claude Code before the model sees
it, even in headless mode, and with no flag from us. A file at
`~/.claude/commands/name.md` or `<dir>/.claude/commands/name.md` is expanded
into the prompt.

In version 2.1.176, `$ARGUMENTS` holds the whole argument string and is
reliable. `$1` returned the **second** argument and `$2` was empty, so
positional arguments in a Claude Code command are off by one. Use `$ARGUMENTS`.

The multiplexer has its own preset prompts, which use the same `/name` syntax.
Its own templates win, and every other name goes to the child unchanged. See
[templates.md](./templates.md).

## Text as it arrives

With `--include-partial-messages` the child also emits `stream_event` lines
while the model writes:

```json
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"A mutex is "}}}
```

Claude Code groups the tokens, so the pieces are large. A 250-word answer
arrives in about 13 pieces, roughly 150 characters each, and the first piece
comes about half way through the turn.

Only `text_delta` matters here. A `thinking_delta` carries the reasoning, and a
`signature_delta` carries a signature, and both are ignored. The whole message
still arrives afterwards as a normal `assistant` event, so nothing is lost when
the pieces are ignored.

**Stream events are not written to the transcript.** They repeat what the
`assistant` event already holds, and they would roughly double the file. The
transcript therefore holds every other line, and a replay of it shows the
conversation once. See [sessions.md](./sessions.md).

## Output events

The `type` field selects the shape:

| `type` | Meaning |
|---|---|
| `system`, subtype `init` | The session is ready. It carries `session_id`, `model`, `cwd`, `tools`, and `permissionMode`. |
| `assistant` | An assistant message. The content holds `text`, `thinking`, and `tool_use` blocks. |
| `user` | A tool result that returns to the model. |
| `result` | The turn is complete. It carries `is_error`, `duration_ms`, `num_turns`, `total_cost_usd`, and `usage`. |
| `stream_event` | A partial message delta. It needs `--include-partial-messages`. |
| `system`, subtype `task_started` / `task_updated` / `task_notification` | A background job lifecycle event. See below. |

Version 2.1.176 also emits `system`/`hook_started`, `system`/`hook_response`,
`system`/`session_state_changed`, and a top-level `rate_limit_event`. The
decoder does not model these. It keeps the raw line, and the caller ignores it.

**An unknown `type` is never an error.** `Decode` fills `Event.Type` and
`Event.Raw`, and it returns no error. A new event kind in a later Claude Code
version therefore cannot stop a session.

A `result` can carry `is_error: true` with the subtype `success`. The renderer
reads `is_error`, not the subtype.

### Content blocks

A `content` field is either a string or an array of blocks. `Content` accepts
both, and it turns a bare string into one text block. A `tool_result` block
carries its own nested content, which follows the same rule.

## Input events

The writer sends one line for each prompt:

```json
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"..."}]},"parent_tool_use_id":null}
```

The `session_id` field appears only after the `init` event gives the session its
identifier.

**Send only the fields the API accepts.** The encoder uses a small struct with
two fields, and not the rich `Block` struct that the decoder fills. A `Block`
carries empty `thinking`, `id`, and `name` fields. Claude Code forwards them to
the API, and the API answers `400 messages.0.content.1.text.thinking: Extra
inputs are not permitted`.

## The interrupt control request

The writer stops a running turn with a control request:

```json
{"type":"control_request","request_id":"int-1","request":{"subtype":"interrupt"}}
```

Claude Code answers with a `control_response`, then ends the turn with a
`result` event whose subtype is `error_during_execution`, and stays alive for
the next prompt. This was proven against Claude Code 2.1.176. The multiplexer
ignores the `control_response`, because the `result` event is the signal it
already acts on. See [sessions.md](./sessions.md).

## The model and the mode change while the session runs

Two more control requests change a running child. Each is proven against Claude
Code 2.1.176:

```json
{"type":"control_request","request_id":"model-1","request":{"subtype":"set_model","model":"sonnet"}}
{"type":"control_request","request_id":"mode-1","request":{"subtype":"set_permission_mode","mode":"plan"}}
```

`set_model` takes a full name or an alias (`opus`, `sonnet`, `haiku`).
`set_permission_mode` takes one of the six modes and answers with the new mode.

The writer sends each request with a counter in the id (`model-1`, `mode-2`).
The multiplexer ignores the `control_response` and trusts the change, the same
as it trusts the interrupt. So the session bar shows the new value at once. A
later version that rejects a request would leave the bar wrong until the next
`init` or `result` event, which is the same risk the interrupt already takes.

## Effort does not change live

There is no live effort switch. `set_effort` is not a request — Claude Code
answers `Unsupported control request subtype`. So the multiplexer changes effort
by a resume: it stops the child and starts it again with the new `--effort`
level, and keeps the conversation through the session id. A resume needs a
session id, so it fails on a session that has not run a turn yet. See
[sessions.md](./sessions.md).

## The AskUserQuestion tool

The model asks the human a multiple-choice question with the `AskUserQuestion`
tool. In headless mode the child does not wait for the human. It emits the
`tool_use` block, and at once it answers the block itself:

```json
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_…","name":"AskUserQuestion","input":{"questions":[…]}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_…","is_error":true,"content":"Answer questions?"}]}}
```

The model then recovers with a line of text, or with real work, and ends the
turn with a normal `result`. This was proven against Claude Code 2.1.176. So the
supervisor gets no window to send its own `tool_result`, and no flag makes the
child wait.

**The session interrupts the turn and waits.** When `apply` sees the
`AskUserQuestion` tool_use, the session fires an `interrupt` at once. The
interrupt cuts the turn before the model recovers and acts on a guess. The
interrupt result ends the turn in the `waiting` state, not `idle`, so the
sidebar shows which session needs an answer. The human answer then runs as the
next prompt, and it moves the session to `busy`. See
[sessions.md](./sessions.md) for the state, and [tui/input.md](./tui/input.md)
for the pane.

`Event.AskUserQuestion` reads the questions and the block id from the `tool_use`
block. Each question holds a `question`, a `header`, a list of `options` (each
with a `label` and a `description`), and a `multiSelect` flag. Because the child
already closed the tool call, the multiplexer gives the human answer back as the
next prompt, not as a `tool_result`.

## Background jobs

Claude Code can run a shell command in the background. A `Bash` tool call with
`run_in_background: true` in its input starts a job. The job keeps running after
the turn ends. Claude Code then pushes three `system` events for the job, each
with a `task_id`. The multiplexer reads these events, and it does not parse the
tool result text. This was proven against Claude Code 2.1.176.

| Subtype | Fields | Meaning |
|---|---|---|
| `task_started` | `task_id`, `tool_use_id`, `description`, `task_type` | The job started. |
| `task_updated` | `task_id`, `patch` (`status`, `end_time`) | The status changed. |
| `task_notification` | `task_id`, `tool_use_id`, `status`, `summary`, `output_file` | The job stopped. |

The `patch.status` values seen are `completed` (exit 0) and `killed`. A
`task_notification` on a kill carries the status `stopped`. `Event.Task` holds
the union of the three shapes, with a `Patch` sub-struct for `task_updated`.

The session turns these events into `Job` records; see [sessions.md](./sessions.md).
The `BashOutput` tool result also carries `<status>`, `<exit_code>`, and
`<output>` tags, but the multiplexer does not read them, because the three push
events carry the same status in a structured form.

## The multiplexer serves the child an MCP server

The command line above carries two more flags for every session the interface
supervises: `--mcp-config <file>` and `--allowedTools mcp__mux__…`. They point
the child at an HTTP server inside the multiplexer, so the Claude in a session
can name itself and read its neighbours.

That channel runs beside this one, and it is not stream-json. It is described in
[mcp.md](./mcp.md), together with what Claude Code 2.1.176 was proven to do with
it.

The `init` event reports each MCP server it connected to, and the decoder reads
that list as `protocol.Init.MCPServers`:

```json
{"type":"system","subtype":"init","mcp_servers":[{"name":"mux","status":"connected"}]}
```

## Line limits

`Reader` reads a line of any length up to 16 MiB, because a tool result can be
large. A longer line is cut at the limit, and the event carries `Truncated` with
`ErrNotJSON`. The reader then finds the next newline and continues. One
oversized line therefore costs one event, and not the session.

Blank lines are skipped. A final line without a newline is still returned.
