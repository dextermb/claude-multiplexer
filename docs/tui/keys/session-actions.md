# Acting on a session

These keys rename a session, open its working directory, change its model,
effort, and mode, act while it is busy, and show its background jobs. For the
full key tables, see [../keys.md](../keys.md).

## Renaming a session

`s n` opens a small dialog with one text field, filled with the current name.
Type a new title, then press `Enter`. An empty field clears the title, so the
session shows its name again. A rename works on a live session and on a stored
session. The name stays the key on disk. The title is only the display text.
See [../sessions.md](../sessions.md) and [../../manager.md](../../manager.md).

## Opening the working directory

`s f` opens the directory of the selected session in the file manager of the
platform: Finder, Explorer, or `xdg-open`.

That directory is the one the session works in now. A session that moves into a
worktree says so with the `set_working_dir` tool, and both keys follow it. When
a session sets none, and when the one it set is gone, the keys open the
directory the session started in. See [../../mcp/tools.md](../../mcp/tools.md).

`s d` opens the same directory in your editor. Name the editor with `--editor`,
with `$VISUAL` or `$EDITOR`, or in the settings file. A terminal editor takes
the terminal until it stops, and a window editor starts beside the interface.
See [../../config.md](../../config.md).

## Changing the model, effort, and mode

`s m`, `s e`, and `s p` each open a small dialog for a running session. The
dialog lists the values, marks the current one, and applies your choice at once.
`s m` sets the model, `s e` sets the effort, and `s p` sets the permission mode.
The session bar then shows the new value.

The model and the permission mode change on the running child. The effort has no
live switch in Claude Code, so `s e` resumes the session with the new level: it
stops the child and starts it again, and keeps the conversation. See
[../../protocol.md](../../protocol.md). Effort is also a field in the new session
form, next to the model and the mode.

## While a session is busy

A prompt you send while a session is busy waits in the queue, and the pane shows
it at once. Two keys then act on the running turn:

- `Esc` stops the turn and drops the queue, so the session waits for your next
  prompt. `s x` stops the whole session, which `Esc` does not.
- `Enter` on an empty prompt box sends the queued prompt now: it stops the turn,
  so the queued prompt goes at once instead of after the turn ends.

Both use the interrupt described in [../../sessions.md](../../sessions.md#interrupt).
When the session is not busy, `Esc` still leaves the pane as before.

## Showing background jobs

Press `s j` to open the jobs dialog for the selected session. It has two levels:
a list, and one job with its output.

The first level lists every background job, the running ones first, then the
finished ones. Move with `j` and `k`, or with the arrow keys, and jump to an end
with `g` and `G`. Press `Enter` to open the job under the cursor.

The second level shows that job and its output. The same scroll keys work
inside it. Press `Esc` to step back to the list, and `Esc` again to close.

The dialog is live, so a running job grows while you read it. See
[../sessions.md](../sessions.md).
