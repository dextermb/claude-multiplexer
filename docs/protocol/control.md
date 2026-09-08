# The control requests

The writer changes a running child with a control request, and trusts the
change rather than the response. Each request below is proven against Claude
Code 2.1.176. See [../protocol.md](../protocol.md) for the stream, and
[../sessions.md](../sessions.md) for the state machine.

## The interrupt control request

The writer stops a running turn with a control request:

```json
{"type":"control_request","request_id":"int-1","request":{"subtype":"interrupt"}}
```

Claude Code answers with a `control_response`, then ends the turn with a
`result` event whose subtype is `error_during_execution`, and stays alive for
the next prompt. This was proven against Claude Code 2.1.176. The multiplexer
ignores the `control_response`, because the `result` event is the signal it
already acts on. See [../sessions.md](../sessions.md).

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
[../sessions.md](../sessions.md).
