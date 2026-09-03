# Keys and the mouse

Every key, how the output scrolls, what the mouse does, and how to leave.

## Keys

`Tab` moves the focus through three panes in turn: the list, the prompt, and the
output.

| Key | Action |
|---|---|
| `Tab` | Move the focus to the next pane |
| `j`, `k`, `up`, `down` | Move through the list, or scroll the output |
| `Enter` | Type into a live session, or resume one that is not running |
| `Enter` (in the prompt) | Send the prompt, or send a queued prompt now while busy |
| `Enter` (in the output) | Open a large result, or move to the prompt when there is none |
| `i` (in the output) | Move to the prompt |
| `ctrl+j` | Add a new line inside the prompt |
| `Shift+Tab` (in the prompt) | Walk the paths that match an `@` word |
| `Esc` | Stop a running turn, or leave the prompt or the output, or close the form or a dialog |
| `n` or `ctrl+n` | Open the new session form |
| `t` or `ctrl+p` | Open the preset prompts |
| `r` | Resume the selected session |
| `a` | Archive the selected session, or bring it back |
| `A` | Show or hide archived sessions |
| `R` | Rename the selected session |
| `x` or `ctrl+x` | Stop the selected session, after a confirmation |
| `pgup`, `pgdown` | Scroll the output by a page |
| `m` | Switch between rendered markdown and raw text |
| `M` | Change the model of a running session |
| `e` | Change the effort of a running session |
| `p` | Change the permission mode of a running session |
| `?` | Show every key, with a search |
| `ctrl+t` | Turn the mouse on or off |
| `q` | Stop every session, and quit |
| `ctrl+c` | Clear the prompt. Press it again to quit |

In the prompt box, `Tab` completes a `/preset` name, then an `@` path, before it
moves the focus. See [input.md](input.md).

`n`, `t`, `r`, `a`, `A`, `R`, `m`, `M`, `e`, `p`, `?`, `x`, and `q` work when the focus is the
list or the output. The `ctrl` forms work everywhere, so they still work while you
type a prompt.

### Renaming a session

`R` opens a small dialog with one text field, filled with the current name.
Type a new title, then press `Enter`. An empty field clears the title, so the
session shows its name again. A rename works on a live session and on a stored
session. The name stays the key on disk; the title is only the display text.
See [sessions.md](./sessions.md) and [manager.md](../manager.md).

### Changing the model, effort, and mode

`M`, `e`, and `p` each open a small dialog for a running session. The dialog
lists the values, marks the current one, and applies your choice at once. `M`
sets the model, `e` sets the effort, and `p` sets the permission mode. The
session bar then shows the new value.

The model and the permission mode change on the running child. The effort has no
live switch in Claude Code, so `e` resumes the session with the new level: it
stops the child and starts it again, and keeps the conversation. See
[protocol.md](../protocol.md). Effort is also a field in the new session form,
next to the model and the mode.

### While a session is busy

A prompt you send while a session is busy waits in the queue, and the pane shows
it at once. Two keys then act on the running turn:

- `Esc` stops the turn and drops the queue, so the session waits for your next
  prompt. `x` stops the whole session, which `Esc` does not.
- `Enter` on an empty prompt box sends the queued prompt now: it stops the turn,
  so the queued prompt goes at once instead of after the turn ends.

Both use the interrupt described in [sessions.md](../sessions.md#interrupt). When
the session is not busy, `Esc` still leaves the pane as before.

### The key list

Press `?` for every key in one place, grouped by where it works. Type to search
it: the search reads the keys, what they do, and the group names, so `scroll`
finds the four keys that scroll and `ctrl+j` finds itself. Press `esc` to close.

The list in this page and the list on the screen come from one table in the
code, so they cannot drift apart. The status bar shows only three of these keys
(`n new · t preset · ? keys`), and it reads them from the same table.

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

### Opening a large result

A big tool result does not fill the pane. The pane shows a short summary, such
as `← 4213 lines`, with a `⏎` mark to show that it opens. See
[output.md](output.md).

Give the output pane the focus, then press `Enter`. A dialog lists every result
that opens. Move with `j` and `k`, then press `Enter` to page the one you want.
The same scroll keys work inside it. Press `Esc` to step back to the list, and
`Esc` again to close.

## The mouse

- A click on a sidebar row selects that session.
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

`q` quits with one press, when the focus is the list.

Both stop every session first, with a grace period of 5 seconds, so no child
process is left behind.
