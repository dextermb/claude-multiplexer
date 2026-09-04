# multiplexer — what is still ahead

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
  mcp__cmux__approve` points at an MCP tool that the multiplexer serves. Each
  request then appears in the interface, and the human answers with a key
  press. Two parts of this are built already: the MCP server, see
  [mcp.md](../mcp.md), and the `waiting` state with its `?` glyph, which the
  question dialog uses, see [../tui/input.md](../tui/input.md). What is left is
  the tool itself, and the reuse of that state for a request. A
  `control_request` event already reaches the app unparsed (`ev.Raw`), but its
  schema is not modelled, so this work must first probe what the installed
  version emits.
- **A task router.** A queue, and dispatch to the first free session. This is a
  second product rather than a feature, so it needs its own plan.
- **The sidebar groups, after they are used.** Two questions the build left
  open, both to answer from use. Must a fold survive a restart, which needs a
  file to hold it? And does a blank line between two groups read better than
  the header alone? See [../tui/sessions.md](../tui/sessions.md).
- **A large result, after it is used.** `Enter` in the output pane opens a
  dialog that lists the results that carry a body, and pages the one you pick.
  See [../tui/output.md](../tui/output.md). Three ideas follow from it, none
  decided:
  - **An in-pane cursor.** Highlight the openable line in place and open it
    with `Enter`, instead of a chooser. This needs a map from a display row
    back to its source line, because the pane renders `ClassText` into a
    variable number of rows.
  - **Read the body from the transcript.** Today the body rides on the line in
    the buffer. If a large body must not sit in memory, keep a reference on the
    line and read the body on open. This needs a per-block index that
    `session.Transcript` does not have.
  - **A search inside an open result.**

---

## Open question

**Session directories.** Every session takes a directory that already exists.
Should the multiplexer instead make a Git worktree for each session, so two
agents in one repository cannot tread on each other? That would tie the tool to
Git, and it would need a rule for removing a worktree when a session ends.
