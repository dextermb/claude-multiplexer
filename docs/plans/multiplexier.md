# multiplexier — what is still ahead

**Status:** the multiplexer works, and it is verified against Claude Code
2.1.176. What it does is described in [protocol.md](../protocol.md),
[sessions.md](../sessions.md), [manager.md](../manager.md),
[tui.md](../tui.md), [templates.md](../templates.md), and
[markdown.md](../markdown.md).

This file holds only the work that is not built. Delete it when nothing is left.

---

## Control

- **A configuration file** at `~/.multiplexier/config.json`, to hold the default
  model, the default permission mode, and the state directory. Every one of
  those is a flag today, and a flag is the wrong place for a setting you never
  change.

---

## Optional, after a review

- **A permission prompt for each request.** `--permission-prompt-tool
  mcp__mux__approve` points at an MCP tool that the multiplexer serves. Each
  request then appears in the interface, and the human answers with a key
  press. The MCP server this needs is built — see [mcp.md](../mcp.md) — so what
  is left is the tool itself, a new `input required` state, and its own
  indicator (a `◉` glyph fits the circle set in
  [tui/sessions.md](../tui/sessions.md)). A `control_request` event already
  reaches the app unparsed (`ev.Raw`), but its schema is not modelled, so this
  work must first probe what the installed version emits.
- **A task router.** A queue, and dispatch to the first free session. This is a
  second product rather than a feature, so it needs its own plan.

---

## Open question

**Session directories.** Every session takes a directory that already exists.
Should the multiplexer instead make a Git worktree for each session, so two
agents in one repository cannot tread on each other? That would tie the tool to
Git, and it would need a rule for removing a worktree when a session ends.
