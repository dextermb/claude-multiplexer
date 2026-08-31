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
| `Enter` (in the prompt) | Send the prompt |
| `ctrl+j` | Add a new line inside the prompt |
| `Esc` | Leave the prompt or the output, or close the form |
| `n` or `ctrl+n` | Open the new session form |
| `t` or `ctrl+p` | Open the preset prompts |
| `r` | Resume the selected session |
| `a` | Archive the selected session, or bring it back |
| `A` | Show or hide archived sessions |
| `x` or `ctrl+x` | Stop the selected session, after a confirmation |
| `pgup`, `pgdown` | Scroll the output by a page |
| `m` | Switch between rendered markdown and raw text |
| `?` | Show every key, with a search |
| `ctrl+t` | Turn the mouse on or off |
| `q` | Stop every session, and quit |
| `ctrl+c` | Clear the prompt. Press it again to quit |

In the prompt box, `Tab` completes a `/preset` name before it moves the focus.

`n`, `t`, `r`, `a`, `A`, `m`, `?`, `x`, and `q` work when the focus is the list. The `ctrl`
forms work everywhere, so they still work while you type a prompt.

### The key list

Press `?` for every key in one place, grouped by where it works. Type to search
it: the search reads the keys, what they do, and the group names, so `scroll`
finds the four keys that scroll and `ctrl+j` finds itself. Press `esc` to close.

The list in this page and the list on the screen come from one table in the
code, so they cannot drift apart. The short hints in the status bar come from
the same table.

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

## The mouse

- A click on a sidebar row selects that session.
- A click on the prompt area moves the focus there.
- The wheel over the sidebar moves the selection.
- The wheel over the output scrolls it.

A terminal cannot select text while a program captures the mouse. Press `ctrl+t`
to release it, copy your text, and press `ctrl+t` again.

## Quitting

`ctrl+c` takes two presses. The first one clears the prompt, closes the new
session form, and drops a confirmation, and the status bar then reads `press
ctrl+c again to quit`. The second one quits.

Any other key press disarms it. So a `ctrl+c` you press now, and another you
press after typing, never add up to an accidental exit.

`q` quits with one press, when the focus is the list.

Both stop every session first, with a grace period of 5 seconds, so no child
process is left behind.
