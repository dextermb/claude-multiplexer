# The AskUserQuestion tool

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
[../sessions.md](../sessions.md) for the state, and [../tui/input.md](../tui/input.md)
for the pane.

`Event.AskUserQuestion` reads the questions and the block id from the `tool_use`
block. Each question holds a `question`, a `header`, a list of `options` (each
with a `label` and a `description`), and a `multiSelect` flag. Because the child
already closed the tool call, the multiplexer gives the human answer back as the
next prompt, not as a `tool_result`.
