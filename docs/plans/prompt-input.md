# Plan: a prompt box that grows, and `@` path completion

Status: **awaiting go-ahead.** The three open questions are settled (see
Decisions). Nothing is built yet.

Two changes to the prompt box, in one effort because both touch the same lines
of `promptView` and `promptKey`:

1. The box grows from 2 rows to 4 rows as the prompt gets longer.
2. A word that starts with `@` suggests paths, and `Tab` completes one.

The shipped behaviour of the box is in [../tui/input.md](../tui/input.md) and
[../tui/keys.md](../tui/keys.md).

---

## Decisions

| Decision | Choice | Why |
|---|---|---|
| Height range | 2 to 4 rows | The floor stays where it is today, so no layout gets shorter than the one the user knows. |
| What counts as a row | Wrapped rows as well as newlines | A long pasted path or a long sentence is the common case, and a box that ignores it looks broken. |
| `@file` base | The selected session's directory | The child Claude runs in that directory, so the path you complete is the path it resolves. |
| The `@` stays in the sent text | Yes | `@path` is Claude Code's own file reference. The multiplexer completes it and sends it through. |
| Completion position | Only when the cursor is at the end of the prompt | `textarea` can set a value or a column, but it cannot restore a cursor row after a soft wrap. See "Ahead". |

## Rejected

- **A row count from the textarea itself.** `LineCount` counts newlines only,
  and `Height` returns the height we set. Neither sees a soft wrap, so the
  count is computed in the TUI (see below).
- **`@` relative to the multiplexer's working directory.** The prompt goes to a
  child in a different directory, so a path completed against our own directory
  can name a file the child cannot open.

---

## Part 1 — the growing box

### Today

```
styles.go     promptHeight = 3          a constant
app.go:129    prompt.SetHeight(promptHeight - 1)      2 text rows
app.go:1279   bodyHeight = height - promptHeight - statusHeight
app.go:771    the mouse hit test for the prompt area
promptView    one hint row + prompt.View()
```

The block is a hint row above two text rows, and every other height is measured
from the constant.

### After

`promptHeight` becomes a method on the model, because it now depends on what is
typed:

```
promptRows(value, width) = 1 + clamp(wrappedRows(value, textWidth), 2, 4)
```

- `wrappedRows` counts the display rows of the value: for each line, a greedy
  word wrap at `textWidth`, and an overlong word breaks. It stops counting at
  4, because nothing above the cap changes the answer.
- `textWidth` is `m.prompt.Width()`, which the textarea reports after
  `SetWidth`. It is the total width less the `> ` prompt.
- The count is exact for the ordinary cases and can differ from the textarea's
  own wrap by one row for a line of double-width runes. The textarea scrolls
  inside itself when that happens, so the text stays reachable.

The height is set in one place. A helper `m.syncPromptHeight()` runs after any
change to the value — a key, a paste, a drop, a preset fill, a reset — and
calls `m.prompt.SetHeight(rows)`, then the output viewport is resized because
`bodyHeight` moved.

Every reader of the constant takes the method instead:

- `bodyHeight`, and through it `outputHeight` and `visibleRows`.
- The mouse hit test at `app.go:771`.
- `View`, which already joins body, prompt, and status.

The constant stays as `promptRowsMin`/`promptRowsMax` (2 and 4) plus the one
hint row.

### Verify

`internal/tui` table tests, driven the way `app_test.go` drives the model:

- An empty prompt keeps a 3-row block.
- Two `ctrl+j` presses give a 4-row block, a third gives 5, a fourth gives no
  more.
- A pasted line longer than the text width grows the block by a row.
- Removing the lines shrinks the block back to 3.
- `outputHeight` falls by exactly the rows the prompt gained.

## Part 2 — `@` completes a path

### The token

The completion looks at the word the cursor sits in: the text of the prompt
after the last space, tab, or newline. It is a path token when it starts with
`@`, and the rest of it is the path.

```
write a note about @internal/tui/pa
                   ^token           ^cursor
base = the selected session's directory
```

`~` expands to the home directory, as it does in the form. A token that starts
with `/` is absolute and the base is not used.

### The matches

`paths.go` holds `completePath`, which lists directories only, for the new
session form. It gains a shared core and two entry points:

```go
func completePath(input string) (string, []string)              // directories, unchanged
func completeEntry(base, input string) (string, []string)       // files and directories
```

The core keeps the rules the form already has: a case-insensitive prefix match,
a dot-name offered only once the dot is typed, one match completes in full,
several match up to their common prefix. A directory match ends in a separator,
so you can carry on into the next level. A file match ends in a space, because
there is nothing more to complete.

### The keys

In the prompt box:

- The matches appear under the label as you type, in the row `pathHint` already
  renders for the form.
- `Tab` grows the token. With nothing left to grow, `Tab` moves the focus, as
  it does today.
- `Shift+Tab` walks the matches backwards and marks the one you are on, the
  same as the form's directory field.
- Any other key stops the walk.

`Tab` already runs `m.complete()` for a `/preset` name. The order becomes:
a `/preset` name first, then a path token, then the focus move. The two cannot
both match, because one starts at the head of the prompt and the other starts
at an `@`.

### The state

The model gains the three fields the form carries — `pathMatches`,
`pathPicked`, `pathStem` — refreshed after each key in the prompt. `promptView`
renders `pathHint(...)` from them, so no key press reads the disk twice.

### Verify

- `@` with a prefix lists the matching files and directories of the session's
  directory, not of ours.
- `@/tmp/` lists an absolute path.
- `@~/` expands.
- `Tab` on one match finishes it, and adds a separator for a directory and a
  space for a file.
- `Tab` on several grows to the common prefix.
- `Tab` with no token still moves the focus.
- `Shift+Tab` walks and wraps.
- The `@` survives into the sent prompt.

---

## The build

1. `just worktree prompt-input`.
2. Part 1, with its tests.
3. Part 2, with its tests.
4. Docs, in the same change: the growing box and `@` completion in
   [../tui/input.md](../tui/input.md), the two keys in
   [../tui/keys.md](../tui/keys.md).
5. `just test`, then `just collapse prompt-input`.
6. Delete this plan, and move anything still true into the two pages.

## Ahead — not in this change

- **Complete in the middle of a prompt.** The cursor must be at the end today.
  A completion further back needs a cursor row restored across a soft wrap, and
  `textarea` gives no way to do it.
- **A name that holds a space.** It is inserted as it is, so the token breaks.
  A drop quotes such a path; `@` cannot, because Claude Code reads the `@` and
  the path as one word.
