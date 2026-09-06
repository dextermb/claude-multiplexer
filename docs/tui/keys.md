# Keys and the mouse

Every key, what the mouse does, and how to leave. This page is the reference of
tables. Two pages carry the detail of how each action behaves:

| Page | Read it for |
|---|---|
| [keys/navigation.md](keys/navigation.md) | Folding, hiding the sidebar, the diff panel, scrolling, and opening a large block |
| [keys/session-actions.md](keys/session-actions.md) | Renaming, the working directory, the model, effort, and mode, a busy session, and the jobs dialog |

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
| `s d` | Show the working-tree diff of the selected session |
| `s E` | Open the working directory in the editor |
| `s m` | Change the model of a running session |
| `s e` | Change the effort of a running session |
| `s p` | Change the permission mode of a running session |

See [keys/session-actions.md](keys/session-actions.md) for what these do.

## The list: `l`

| Keys | Action |
|---|---|
| `l f` | Fold or unfold the group of the selected session |
| `l F` | Fold every group but the one you are in |
| `l u` | Unfold every group |
| `l a` | Show or hide the archived sessions |
| `l t` | Hide or show the sidebar |
| `l c` | Hide the sidebar, for more pane width |
| `l e` | Show the sidebar again |

See [keys/navigation.md](keys/navigation.md) for folding and the sidebar.

## The output pane: `o`

| Keys | Action |
|---|---|
| `o m` | Switch between rendered markdown and raw text |
| `o l` | Open the layouts, to switch between them. See [layouts.md](layouts.md) |

## The diff panel: `d`

The `d` target starts only while the diff panel is open. At every other time `d`
keeps its output-scroll meaning.

| Keys | Action |
|---|---|
| `d +` | Widen the diff panel |
| `d -` | Narrow the diff panel |
| `d /` | Show the panel at half the screen, or the set width |
| `d n` | Show or hide the line numbers |

See [keys/navigation.md](keys/navigation.md) for the full diff panel keys.

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
| `↑`, `↓` (in the prompt) | Recall an older or a newer prompt you sent |
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

## The key list

Press `?` for every key in one place, grouped by target. Type to search it: the
search reads the keys, what they do, and the group names, so `scroll` finds the
keys that scroll and `ctrl+j` finds itself. Press `esc` to close.

The list in this page and the list on the screen come from one table in the
code, so they cannot drift apart. The status bar reads the same table. It shows
the keys that work on their own and the three targets (`n new · t preset ·
s session · l list · o output · ? keys · q quit`). While a sequence waits, it
shows the actions of that target instead.

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
