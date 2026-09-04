# Keys and the mouse

Every key, how the output scrolls, what the mouse does, and how to leave.

## The shape of a key sequence

An action takes two keys. The first key names the **target**. The second key
names the **action**. After the first key, the status bar lists the actions of
that target.

| Key | Target | The prompt form |
|---|---|---|
| `s` | The selected session | `ctrl+s` |
| `l` | The list | `ctrl+l` |
| `o` | The output pane | `ctrl+o` |

A sequence waits 1 second for the second key. After that the target is forgotten,
and the next key acts on its own. `Esc` cancels the sequence at once. An unknown
second key also cancels it, and the status bar shows a short notice, such as
`no key s w`.

The plain target keys work when the focus is the list or the output. The control
forms work everywhere, so a sequence still starts while you type a prompt.

## The session: `s`

| Keys | Action |
|---|---|
| `s c` | Start a new session |
| `s t` | Open the preset prompts |
| `s r` | Resume the selected session |
| `s n` | Rename the selected session |
| `s a` | Archive the selected session, or bring it back |
| `s x` | Stop the selected session, after a confirmation |
| `s j` | Show the background jobs of the selected session |
| `s f` | Open the working directory in the file manager |
| `s d` | Open the working directory in the editor |
| `s m` | Change the model of a running session |
| `s e` | Change the effort of a running session |
| `s p` | Change the permission mode of a running session |

## The list: `l`

| Keys | Action |
|---|---|
| `l f` | Fold or unfold the group of the selected session |
| `l F` | Fold every group but the one you are in |
| `l u` | Unfold every group |
| `l a` | Show or hide the archived sessions |

## The output pane: `o`

| Keys | Action |
|---|---|
| `o m` | Switch between rendered markdown and raw text |

## The keys that work on their own

`Tab` moves the focus through three panes in turn: the list, the prompt, and the
output.

| Key | Action |
|---|---|
| `Tab` | Move the focus to the next pane |
| `j`, `k`, `up`, `down` | Move through the list, or scroll the output |
| `Enter` | Type into a live session, or resume one that is not running |
| `Enter` (in the prompt) | Send the prompt, or send a queued prompt now while busy |
| `Enter` (in the output) | Open or close the block under the cursor, or move to the prompt when there is no capped block |
| `[`, `]` (in the output) | Move the cursor between the capped blocks |
| `i` (in the output) | Move to the prompt |
| `ctrl+j` | Add a new line inside the prompt |
| `Shift+Tab` (in the prompt) | Walk the paths that match an `@` word |
| `Esc` | Stop a running turn, leave the prompt or the output, close a dialog, or cancel a sequence |
| `n` or `ctrl+n` | Open the new session form |
| `t` or `ctrl+p` | Open the preset prompts |
| `u`, `d` | Scroll the output by half a pane |
| `pgup`, `pgdown` | Scroll the output by a page |
| `g`, `G`, `home`, `end` | Go to the top of the output, and to the bottom |
| `?` | Show every key, with a search |
| `ctrl+t` | Turn the mouse on or off |
| `q` | Stop every session, and quit |
| `ctrl+c` | Clear the prompt. Press it again to quit |

`n`, `t`, `?`, and `q` work when the focus is the list or the output. In the
prompt box, `Tab` completes a `/preset` name, then an `@` path, before it moves
the focus. See [input.md](input.md).

A dialog that names one session — rename, model, effort, mode, jobs, and the
stop confirmation — draws in the pane, so the sidebar stays on the screen. A dialog that names no session covers the sidebar too. See
[../tui.md](../tui.md).

### Folding a directory

The list groups the sessions under a header, by directory or by the control
session that created them. `l f` folds the group of the selected session, and
`l f` again unfolds it. `l F` folds every group but the one you are in, and
`l u` unfolds every group. A fold moves the selection to a row you can see, and
it is forgotten when the program stops. See [sessions.md](./sessions.md).

### Renaming a session

`s n` opens a small dialog with one text field, filled with the current name.
Type a new title, then press `Enter`. An empty field clears the title, so the
session shows its name again. A rename works on a live session and on a stored
session. The name stays the key on disk. The title is only the display text.
See [sessions.md](./sessions.md) and [manager.md](../manager.md).

### Opening the working directory

`s f` opens the directory of the selected session in the file manager of the
platform: Finder, Explorer, or `xdg-open`.

That directory is the one the session works in now. A session that moves into a
worktree says so with the `set_working_dir` tool, and both keys follow it. When
a session sets none, and when the one it set is gone, the keys open the
directory the session started in. See [../mcp/tools.md](../mcp/tools.md).

`s d` opens the same directory in your editor. Name the editor with `--editor`,
with `$VISUAL` or `$EDITOR`, or in the settings file. A terminal editor takes
the terminal until it stops, and a window editor starts beside the interface.
See [../config.md](../config.md).

### Changing the model, effort, and mode

`s m`, `s e`, and `s p` each open a small dialog for a running session. The
dialog lists the values, marks the current one, and applies your choice at once.
`s m` sets the model, `s e` sets the effort, and `s p` sets the permission mode.
The session bar then shows the new value.

The model and the permission mode change on the running child. The effort has no
live switch in Claude Code, so `s e` resumes the session with the new level: it
stops the child and starts it again, and keeps the conversation. See
[protocol.md](../protocol.md). Effort is also a field in the new session form,
next to the model and the mode.

### While a session is busy

A prompt you send while a session is busy waits in the queue, and the pane shows
it at once. Two keys then act on the running turn:

- `Esc` stops the turn and drops the queue, so the session waits for your next
  prompt. `s x` stops the whole session, which `Esc` does not.
- `Enter` on an empty prompt box sends the queued prompt now: it stops the turn,
  so the queued prompt goes at once instead of after the turn ends.

Both use the interrupt described in [sessions.md](../sessions.md#interrupt). When
the session is not busy, `Esc` still leaves the pane as before.

### The key list

Press `?` for every key in one place, grouped by target. Type to search it: the
search reads the keys, what they do, and the group names, so `scroll` finds the
keys that scroll and `ctrl+j` finds itself. Press `esc` to close.

The list in this page and the list on the screen come from one table in the
code, so they cannot drift apart. The status bar reads the same table. It shows
the keys that work on their own and the three targets (`n new · t preset ·
s session · l list · o output · ? keys · q quit`). While a sequence waits, it
shows the actions of that target instead.

### Scrolling the output

Give the output pane the focus with `Tab`, or click in it. Then:

| Key | Action |
|---|---|
| `j`, `k`, `up`, `down` | One line |
| `u`, `d` | Half a pane |
| `pgup`, `pgdown` | A whole pane |
| `g`, `G`, `home`, `end` | The top, and the bottom |

The pane follows the newest line while you sit at the bottom. As soon as you
scroll up it holds your place, and new output no longer moves the text under
you. The session bar then shows how far up you are, such as `↑ 62%`. Press `G`
to return to the bottom and start following again.

### Opening a large block

A block of more than 20 rows draws its first 20 rows and a marker row, such as
`⋯ 4193 more lines`. `blockCap` in the settings file changes the 20, and `0`
caps nothing. See [../config.md](../config.md). A block is one piece of content: your prompt, one message,
one tool result, or the output of a `!` command. See [output.md](output.md).

The marker row of one block carries `▸` and a highlight. That is the cursor, and
`Enter` opens the block it names. `Enter` again closes it. `]` and `[` move the
cursor to the next capped block and to the block before it, and they stop at the
ends. A click on any marker row opens that block.

The cursor returns to the newest capped block at the end of every turn, so the
answer you are reading is the one `Enter` opens.

### Showing background jobs

Press `s j` to open the jobs dialog for the selected session. It has two levels:
a list, and one job with its output.

The first level lists every background job, the running ones first, then the
finished ones. Move with `j` and `k`, or with the arrow keys, and jump to an end
with `g` and `G`. Press `Enter` to open the job under the cursor.

The second level shows that job and its output. The same scroll keys work
inside it. Press `Esc` to step back to the list, and `Esc` again to close.

The dialog is live, so a running job grows while you read it. See
[sessions.md](./sessions.md).

## The mouse

- A click on a sidebar row selects that session.
- A click on a group header folds or unfolds that group.
- A click on the prompt area moves the focus there.
- The wheel over the sidebar moves the selection.
- The wheel over the output scrolls it.
- A left-drag over the output selects text.

Drag over the output to select text. When you release the mouse, the interface
copies the text to the system clipboard and the status bar shows `copied N
lines`. A click, a key press, or a change of session clears the selection. The
selected text is the plain text, without the colours.

The interface captures the mouse, so the terminal cannot select text itself.
To use the terminal selection instead (for example, to select across both
panes), press `ctrl+t` to release the mouse, select and copy, then press
`ctrl+t` again.

The wheel reports each notch as an escape sequence. A fast roll fills the input
buffer, so the terminal splits one sequence across two reads, and the parser
then reads the tail as ordinary key presses. The interface drops those
fragments, so a wheel roll never leaves stray characters in the prompt.

## Quitting

`ctrl+c` takes two presses. The first one clears the prompt, closes the new
session form, and drops a confirmation, and the status bar then reads `press
ctrl+c again to quit`. The second one quits.

Any other key press disarms it. So a `ctrl+c` you press now, and another you
press after typing, never add up to an accidental exit.

`q` quits with one press, when the focus is the list or the output.

Both stop every session first, with a grace period of 5 seconds, so no child
process is left behind.
