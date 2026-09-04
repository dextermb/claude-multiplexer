# Inline expansion in the session pane

Status: `awaiting go-ahead`. The scope, the keys, and the fate of the result
pager are decided. Nothing is built yet.

A block of more than 20 lines draws its first 20 lines in the session pane, with
a marker row under them. The marker opens the rest in place. No dialog.

## Decisions

1. **Every block caps, not only a tool result.** Your prompt, the assistant
   text, a tool result, and the output of a `!` command all follow one rule.
   Rejected: capping only what comes back, which leaves a pasted 200-line prompt
   filling the pane.

2. **The cap counts the rows the pane draws, after wrapping.** A block of 21
   rows shows rows 1 to 20 and says `⋯ 1 more line`. A count in source lines
   would let one very long line still fill the pane, which is the thing this
   change exists to stop. The consequence: the count moves with the width of the
   pane.

3. **A tool result draws its body, not a summary.** Today a result of two lines
   collapses to `← 4213 lines`. With a cap of 20 lines the summary is no longer
   needed, so the pane draws the body with the `←` mark on its first line. A
   result of 5 lines now takes 5 rows, where it took 1.

4. **The output pane gets a block cursor.** `[` and `]` move it between the
   capped blocks, and `Enter` opens or closes the block under it. The marker row
   of the block under the cursor is highlighted.

5. **The cursor returns to the newest capped block at the end of a turn.** It
   also returns when you select another session. So the block you want is
   almost always under the cursor already, and `[` is only for going back.

6. **The result pager goes.** `o r`, the `pager` type, and the two-level dialog
   are deleted. Inline expansion is the one way to read a large result.

7. **`render.Line.Full` becomes `render.Line.Summary`.** `Text` holds the whole
   body, so the pane has everything it draws. `Summary` holds the one-line form
   (`← 4213 lines`) for the `run` command, which prints one line per event and
   keeps the shape it has today.

## Open questions

None.

## What a block is

A block is the run of rendered lines that came from one piece of content. Most
lines are a block on their own. Two renderers make a run of lines, so
`render.Line` gains a `Cont` field, true on every line that continues the block
the line before it started:

| Content | The block |
|---|---|
| A prompt | Every line of it, `› ` on the first |
| Assistant text | One line, whose `Text` holds the whole message |
| A tool call | One line, clipped as it is today |
| A tool result | One line, whose `Text` holds the whole body |
| A `!` command | The command line, then its output as a second block |
| Everything else | One line, one block |

The command of a `!` line is its own block, so it stays on the screen when its
output is capped.

## What the pane draws

```
› write the summary
● 2127c615 · claude-opus-4-8 · 31 tools
→ Bash ./scripts/build.sh
← go: downloading github.com/charmbracelet/bubbletea v1.3.4
  go: downloading github.com/charmbracelet/lipgloss v1.1.0
  … 18 more body lines …
▸ ⋯ 4193 more lines
The build has three problems▌
```

- A collapsed block ends with `⋯ N more lines`.
- An open block ends with `⋯ show less`.
- The row of the block under the cursor carries `▸ ` and the selected
  background. Every other marker row carries two spaces.
- A click on a marker row opens or closes that block, and moves the cursor to
  it.

## Data flow

```
manager.Lines(name) ─▶ []render.Line ─▶ blocks() ─▶ []block
                                                      │
                              m.expanded[i] ──────────┤
                                                      ▼
                                             wrap() ─▶ m.outputText ─▶ viewport
```

- `blocks()` groups `m.shownLines` by the `Cont` field. It is pure, so a test
  reads it on its own.
- `m.expanded` is a set of block indexes. A block index is stable, because a new
  block only ever lands at the end.
- `m.blockCursor` is an index into the capped blocks.
- An append adds blocks at the end, so `appendOutput` still appends its chunk
  and does not rebuild.
- An expansion, a collapse, a resize, and a change of selection rebuild
  `m.outputText` from `m.shownLines`.
- A rebuild from `Lines(name)` clears `m.expanded`, because the buffer may have
  dropped lines from the front and the indexes then mean something else.

## The keys

| Keys | Action |
|---|---|
| `]` (in the output) | Move the cursor to the next capped block |
| `[` (in the output) | Move the cursor to the block before it |
| `Enter` (in the output) | Open or close the block under the cursor |
| `Enter` (in the output, no capped block) | Move to the prompt, as today |

`o r` is deleted. `o m` stays.

## The build

1. `internal/render`: add `Cont`, rename `Full` to `Summary`, and make
   `toolResultLine` return the body as `Text`. Update `PromptLines` and
   `BashLines` to mark their continuation lines.
2. `cmd/multiplexier`: print `Summary` when it is set, else `Text`. The `run`
   command keeps the output it has today.
3. `internal/tui/block.go`: `blocks()`, the cap, and the marker row.
4. `internal/tui/app.go`: `m.expanded`, `m.blockCursor`, the block-aware `wrap`,
   the `[`, `]` and `Enter` keys, the cursor reset on a turn result, and the
   click on a marker row.
5. Delete `pager.go`, `pager_test.go`, `pager_e2e_test.go`, `openResults`,
   `collectExpandables`, `pagerKey`, and the `o r` binding.
6. Docs: rewrite the collapse section of `docs/tui/output.md`, the two key
   tables in `docs/tui/keys.md`, and the dialog list in `docs/tui.md`.

## How it is verified

- `blocks()` groups a prompt, a `!` command with output, and a tool result.
- A 21-line block draws 20 lines and `⋯ 1 more line`.
- A 20-line block draws no marker.
- `Enter` on the marker draws every line, and `Enter` again draws 20 again.
- `[` and `]` walk the capped blocks and stop at the ends.
- A result event returns the cursor to the newest capped block.
- The view still fills the window exactly, with a block open and closed.
- `run` prints `← 4213 lines` for a large result, as it does today.
