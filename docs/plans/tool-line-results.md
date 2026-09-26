# Tool results on the tool line

**Status:** awaiting go-ahead. The decisions below are proposals. The three open
questions need an answer before the build starts.

The canvas (<https://claude.ai/artifact/B53CvRFbfHa1UnkisHtGjC>, board
"workspace") draws a tool call and its result as one row:

```
 → READ   internal/auth/middleware.go                         412 lines
 → EDIT   internal/auth/refresh.go                               +41 −0
 → BASH   go test ./internal/auth/...
 ← ok  github.com/dextermb/claude-multiplexer/internal/auth  0.412s
   [+] 14 more lines
 → BASH   go test -race ./internal/auth/...              ■ running 0:12
 → GREP   owner: docs/                                          9 files
```

Today the pane draws each call as its own `→` row, and each result as its own
`←` block under the calls, capped to a few rows.

---

## What changes

1. **A note at the right end of the call row.** It says what the result was:
   - `READ` → the rows of the result: `412 lines`.
   - `EDIT`, `WRITE`, `MULTIEDIT` → the lines added and removed: `+41 −0`.
   - `GREP`, `GLOB` → the rows of the result, as files: `9 files`.
   - An error result → `× error`, in the danger colour.
   - Any other tool → no note.
2. **A timer on a call with no result**, while the session is busy:
   `■ running 0:12`, in the warning colour. It ticks with the busy timer.
3. **The `←` body goes away for the folded tools** (see open question 1). Bash
   and every tool with no note keep the `←` body, capped as today.

The transcript, the `run` command and the raw view (`o m`) do not change.

## Data flow

The call and its result arrive in different events: the call in an assistant
message, the result in a later user message. The renderer handles one event at
a time, and a replay uses a new renderer, so the renderer cannot join them. The
pane joins them.

```
assistant event                user event
  tool_use {id, name, input}     tool_result {tool_use_id, content, is_error}
        │                               │
        ▼ render                        ▼ render
  Line{ClassToolUse,              Line{ClassToolResult,
       Tool: id,                       Tool: tool_use_id, ...}
       Note: "+41 −0" (edits only)}
        │                               │
        └──────────► manager buffer ◄───┘   (append-only, as today)
                           │
                           ▼ pane
          pair lines by Tool; draw the note on the call row;
          rewrite the call row when its result arrives
```

- `render.Line` gets two fields, `Tool` (the tool use id) and `Note`. Both are
  `omitempty` on the wire, so an older peer ignores them, and a line from an
  older peer draws as today.
- The renderer computes the edit note from the input (`old_string` and
  `new_string` line counts), because only the call has the input. The pane
  computes the row count from the result text.
- The pane keeps a map from tool id to the row of the call in `outputText`.
  When a result arrives, the pane rewrites that one row, the same way
  `setBlockCursor` rewrites a marker row today. A rebuild pairs every line again
  from `shownLines`.
- The timer reads `Line.At` of the call. The busy tick already redraws the
  screen each second; it also rewrites the rows of the running calls.

## Open questions

1. **The folded body.** The canvas shows no `←` body for `READ`, `EDIT` and
   `GREP`. Choose one:
   - (a) Hide it. The block cursor can land on the call row, and `enter` opens
     the body under it. This changes the block model: a block whose marker is
     the call row, not the last row of the body. **Recommended**, because it
     matches the canvas and a file read is often hundreds of rows.
   - (b) Keep the body, capped, as today, and only add the note. Smaller and
     safer, but the pane stays longer than the canvas.
2. **The turn result line.** The canvas writes `✓ done · 3.2s · …`. The renderer
   writes `✓ success · 7ms · …`, with the word from Claude Code. Change the word
   to `done` (the job vocabulary), or keep `success`?
3. **The input of a tool with no known key.** `summariseInput` falls back to
   compact JSON, such as `{"activeForm":"…","subject":"First task"}` for
   `TaskCreate`. The canvas shows short text only. Add keys for the task tools
   (`subject`, `taskId`), or leave the JSON?

## The build

1. `internal/render`: the `Tool` and `Note` fields, the edit note, tests.
2. `internal/tui`: pair the lines, the note and the timer on the call row, the
   row rewrite on a result, tests with a call and a result in separate events,
   and with a rebuild between them.
3. If question 1 is (a): the block model change in `internal/tui/block.go` and
   `blocks.go`, with tests for open and close from the call row.
4. Docs: `docs/tui/output.md` (the class table, the tool line), and
   `docs/protocol.md` if the wire fields need a line there.
5. Check in tmux with the fake Claude. It needs a new mode that sends a
   `Read`, an `Edit` and a long `Bash`, with a delay before the Bash result, so
   the timer shows.
