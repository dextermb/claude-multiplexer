# Moving and reading

These keys fold the list, hide the sidebar, page through the output, and open a
large block. For the full key tables, see [../keys.md](../keys.md).

## Folding a directory

The list groups the sessions under a header, by directory or by the control
session that created them. `l f` folds the group of the selected session, and
`l f` again unfolds it. `l F` folds every group but the one you are in, and
`l u` unfolds every group. A fold moves the selection to a row you can see, and
it is forgotten when the program stops. See [../sessions.md](../sessions.md).

## Hiding the sidebar

`l c` hides the whole sidebar, and the output pane and the diff panel gain its
width. `l e` shows the sidebar again, and `l t` toggles it. A hide is forgotten
when the program stops. See [../sessions.md](../sessions.md).

## The diff panel

`s d` opens the diff panel of the selected session, and gives it the focus. While
the panel holds the focus, these keys work:

| Key | Action |
|---|---|
| `j`, `k` | Step through an open diff, then to the next or previous file |
| `up`, `down` | Scroll the panel one line |
| `Enter` | Expand or collapse the selected file |
| `pgup`, `pgdown` | Scroll the panel a page |
| `g`, `G` | Go to the top or bottom of the open diff, or of the file list |
| `}`, `{` | Jump to the next or previous empty line of an open diff |
| `d +`, `d -` | Widen or narrow the panel |
| `d /` | Toggle the panel between half the screen and the set width |
| `d n` | Show or hide the line numbers |
| `Tab` | Move the focus on, and keep the panel open |
| `s d` | Move the focus to the panel, or close it when it holds the focus |
| `Esc` | Close the panel |

The mouse wheel scrolls the panel too. See [../diff.md](../diff.md) for the count,
the panel, and the live refresh.

## Scrolling the output

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

## Opening a large block

A block of more than 20 rows draws its first 20 rows and a marker row, such as
`⋯ 4193 more lines`. `blockCap` in the settings file changes the 20, and `0`
caps nothing. See [../../config.md](../../config.md). A block is one piece of
content: your prompt, one message, one tool result, or the output of a `!`
command. See [../output.md](../output.md).

The marker row of one block carries `▸` and a highlight. That is the cursor, and
`Enter` opens the block it names. `Enter` again closes it. `]` and `[` move the
cursor to the next capped block and to the block before it, and they stop at the
ends. A click on any marker row opens that block.

The cursor returns to the newest capped block at the end of every turn, so the
answer you are reading is the one `Enter` opens.
