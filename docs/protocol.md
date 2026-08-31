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
`default`, `dontAsk`, or `plan`. A headless session cannot ask a human for
permission, so the mode is fixed when the session starts.

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

## Line limits

`Reader` reads a line of any length up to 16 MiB, because a tool result can be
large. A longer line is cut at the limit, and the event carries `Truncated` with
`ErrNotJSON`. The reader then finds the next newline and continues. One
oversized line therefore costs one event, and not the session.

Blank lines are skipped. A final line without a newline is still returned.
