# Moving and reading

These keys fold the list, hide the sidebar, page through the output, and open a
large block. For the full key tables, see [../keys.md](../keys.md).

## Folding a directory

The list groups the sessions under a header, by directory or by the control
session that created them. `l f` folds the group of the selected session, and
`l f` again unfolds it. `l F` folds every group but the one you are in, and
`l u` unfolds every group. A fold moves the selection to a row you can see, and
it is forgotten when the program stops. See [../sessions.md](../sessions.md).

## Searching the list

`l s` opens a search box at the top of the sidebar and gives it the focus. As
you type, the list narrows to the sessions whose name, title, or directory holds
the text. The search reads only the rows the current filters keep, so it narrows
the list and never widens it. An archived session stays hidden unless `l a`
shows it first.

The first `Esc` blurs the box but keeps the text, so the list stays narrowed
while you move through the results with `j` and `k`. `Enter`, or an arrow key,
also steps the focus into the list. A second `Esc`, on the list, clears the text
and restores the full list. The search is forgotten when the program stops.

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

## The task and job panel

`s k` moves the focus to the task and job panel, and `Tab` reaches it too. The
panel takes the focus only when it shows. A focused panel draws its left border
in white. While the panel holds the focus, these keys work:

| Key | Action |
|---|---|
| `j`, `k`, `up`, `down` | Scroll one line |
| `u`, `d` | Scroll half a panel |
| `pgup`, `pgdown` | Scroll a whole panel |
| `g`, `G` | Go to the top, and to the bottom |
| `Tab` | Move the focus on |
| `Esc` | Leave the panel for the output |

The mouse wheel scrolls the panel too, when the pointer is over it. The scroll
returns to the top when you select a different session. See [../tasks.md](../tasks.md)
for the panel content.

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

## Searching the output

`s /` opens a find box in the status row and searches the output pane
backwards, from the top of what you can see towards the start of the session.
So the prompt or the answer you scrolled past is the first hit. The box takes
the focus from wherever you were, and the pane moves as you type. In the prompt
box, use `ctrl+s /`, because a plain `s` there is text.

`Enter` keeps the hit and closes the box, and the status row then shows the
count, such as `/parser  2/7`. `n` steps to the next hit in the direction of the
search, which is upwards, and `N` steps the other way. Both wrap around the
pane, and the status row says `wrapped` when they do.

A needle that starts with `>` reads the prompt rows alone. So `s />` steps from
prompt to prompt with no text at all, and `s />fix` finds the prompt that holds
"fix". A plain needle reads every row, and it ignores case.

The row of the current hit is marked, the same as the marked hunk of the review
screen. See [../review.md](../review.md).

`Esc` in the box drops the needle and returns the pane to where it was. After
`Enter`, `Esc` on the pane drops the needle and leaves the pane where it is.
While a needle is live, `n` and `N` step the hits instead of opening the new
session form, so clear the needle with `Esc` to get `n` back.

The search reads the rows the pane draws. A closed block hides its rows, so the
search does not find them until you open the block. The three keys are in the
keymap, so they can be rebound. See
[../../config/keybindings.md](../../config/keybindings.md).

## Opening a large block

A block of more than 20 rows draws its first 20 rows and a marker row, such as
`[+] 4193 more lines`. `blockCap` in the settings file changes the 20, and `0`
caps nothing. See [../../config.md](../../config.md). A block is one piece of
content: your prompt, one message, one tool result, or the output of a `!`
command. See [../output.md](../output.md).

The marker row of one block carries `▸` and a highlight. That is the cursor, and
`Enter` opens the block it names. `Enter` again closes it. `]` and `[` move the
cursor to the next capped block and to the block before it, and they stop at the
ends. A click on any marker row opens that block.

The cursor returns to the newest capped block at the end of every turn, so the
answer you are reading is the one `Enter` opens.
