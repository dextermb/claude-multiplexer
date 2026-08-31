# Plan: expand a collapsed output line

Status: **shipped, first iteration.** The durable behaviour now lives in
[../tui/output.md](../tui/output.md) and [../tui/keys.md](../tui/keys.md). This
plan holds only what is still ahead.

## What shipped

- `render.Line` gained a `Full` field. A collapsed tool result keeps its whole
  body there. A line shown in full leaves it empty.
- The pane marks a line that carries a body with a `⏎`.
- `Enter` in the output pane opens a `pager` modal. It lists every result that
  carries a body, then pages the chosen one in a scrollable view.

## The one decision that changed during the build

The plan chose an **in-pane line cursor** (Option A). The build used a **result
chooser** instead — a dialog that lists the openable results.

Reason: the pane markdown-renders `ClassText` into a variable number of display
rows, so a per-line cursor needs a map from a display row back to its source
line. That map is fragile and touches the pane scroll model and the width test.
The chooser reaches the same goal — open a particular result — with far less
risk, and it is the cleaner base to iterate on.

## Ahead — after user testing

The user expects to change how an expanded result is shown, once the feature is
in use. Open ideas, none decided:

- **In-pane cursor.** Highlight the openable line in place and open it with
  `Enter`, instead of a chooser. Needs the display-row map above.
- **Read from the transcript instead of the buffer.** Today the body rides on
  the line in the buffer. If large bodies must not sit in memory, store a
  reference on the line and read the body from the session transcript on open.
  This needs a per-block index that `session.Transcript` does not have yet.
- **Search inside the open result.** A find, for a long body.

Resolve these from what the testing shows, then update or delete this plan.
