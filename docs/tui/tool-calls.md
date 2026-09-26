# Tool calls in the output pane

The output pane draws each tool call as one row, and puts what the result was at
the right end of that row. The result of a file read, an edit, or a search folds
into the call row, and `Enter` opens it under the call.

```
→ READ   internal/auth/middleware.go                         412 lines
→ EDIT   internal/auth/refresh.go                               +41 −0
→ GREP   owner: docs/                                          9 files
→ BASH   go test ./internal/auth/...
← ok  github.com/dextermb/claude-multiplexer/internal/auth  0.412s
→ BASH   go test -race ./internal/auth/...              ■ running 0:12
```

For the colour of each part of the row, see [theme.md](theme.md). For the other
lines of the pane, see [output.md](output.md).

## The call row

- The arrow is dimmed. The tool name is an uppercase label in muted grey, in a
  slot of six columns, so the input of each call starts in the same column. The
  input is secondary grey.
- The row is one row. A long input ends with `…`, so the note at the right end
  always shows. The raw view (`o m`) shows the whole line, wrapped.
- The input is the most useful field of the tool input: the command, the file,
  the pattern and then the path, the task subject, the task id and its status,
  or the number of questions. Another tool shows its input as compact JSON.

## The note

The note says what the result was:

| Tool | The note |
|---|---|
| `Read` | The rows of the result: `412 lines` |
| `Edit`, `Write`, `MultiEdit` | The lines added and removed: `+41 −0` |
| `Grep` | `9 files` from a `Found 9 files` result, or the rows of the result: `12 lines` |
| `Glob` | The rows of the result, as files: `9 files` |
| Any tool, when the result is an error | `× error`, in the danger colour |
| A call with no result, while its turn runs | `■ running 0:12`, in the warning colour |

Another tool has no note.

The renderer counts the lines of an edit from the tool input (`old_string`,
`new_string`, `content`, and `edits`), because only the call carries the input.
It puts the count in `render.Line.Note`. The pane counts the rows of the other
results from the result text.

The timer counts from the time of the call event (`render.Line.At`), and it moves
with the busy timer of the sidebar. A call shows a timer only while the session
is busy and no turn result follows the call. So a call that a stopped turn left
without a result shows no note.

## The folded result

The result of `Read`, `Edit`, `Write`, `MultiEdit`, `Grep`, and `Glob` draws no
`←` body. The call row is the marker of that result:

- The block cursor can land on the call row, and the row inverts under it.
- `Enter`, `Space`, or a click on the call row opens the body under the call,
  and a `[−] show less` marker under the body closes it.
- `]` and `[` move the cursor through the call rows and the other markers, in
  the order of the rows.

An error result does not fold. It keeps its `←!` body under the calls, capped as
any other block, so the error shows without a key.

A result from `Bash` and from every other tool keeps its `←` body, capped as
today. See [output.md](output.md) for the cap and the block cursor.

## How the pane pairs a call and its result

The call and its result arrive in different events: the call in an assistant
message, the result in a later user message. The renderer handles one event at
a time, so the pane pairs them.

1. The renderer writes the tool use id on the call line and on the result line
   (`render.Line.Tool`).
2. The pane keeps a map from each tool id to the line of its call and the line of
   its result, and a map from each tool id to the row of its call.
3. When a result arrives, the pane draws the call row again, with the note, in
   place. A folded result then draws no rows at its own place.
4. A rebuild pairs every line again, and draws the open body of a folded result
   under its call.
5. A new call with the id of an earlier call drops the earlier result from the
   map, so the new call does not take it.

`Tool` and `Note` travel with the line over the peer stream. A line from an older
peer has neither, so it draws as a plain tool line and a plain result.

The transcript, the `run` command, and the one-line printer do not change. They
print the lines as the renderer writes them.

The code is in `internal/tui/toolpairs.go`, and the note of an edit is in
`internal/render/edits.go`.
