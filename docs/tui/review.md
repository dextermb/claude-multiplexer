# The code review screen

`s R` opens a full-pane review screen for the selected session. It shows a large
diff on the left and the session output on the right. The user jumps between the
files and the hunks of the diff, and asks the live session to explain a hunk or a
file. The explanation shows in the output pane, so it carries the block cursor
and the capped blocks. See [diff.md](diff.md) for the diff the screen reads,
[output.md](output.md) for the blocks, and [keys.md](keys.md) for the key model.

```
 review · alpha · +120 −30 · 3 files          │ Explanation
                                              │
 M internal/tui/app.go            +12 −3      │ › Explain the change at
   @@ -1,3 +1,4 @@ func A()                    │   internal/tui/diff.go:128-131.
    one                                       │ ● reads internal/tui/diff.go
   +two                                       │   ⋯ 12 more lines
 A internal/git/hunks.go          +48 −0      │ This hunk caches the open-file map
   @@ -0,0 +1,48 @@                            │ before the lookup, so the toggle
   +package git                               │ does not allocate on every keypress.
 M internal/tui/diff.go           +64 −12     │
```

## What the screen reads

The screen reuses the diff of the session, the same data the diff panel reads:
the working tree against `origin/HEAD`, one group for each directory of the
project. See [diff.md](diff.md). The screen reads the diff of the selected file
with `git.FileDiff`, then splits it into hunks with `git.Hunks`. A hunk is one
`@@` block. A file with no `@@` (a rename with no content change, or a binary
change) has no hunks, but the screen still lists the file, and `E` still explains
it.

While the screen is open, the diff refreshes on the same 800 ms tick as the diff
panel, so the changes of a running agent show while it works.

If the selected session is not running — stopped, paused, or archived — `s R`
starts it first, and opens the review when it is running. See `reviewSelected`
and `handleSpawned` in the code.

## Navigation

The diff side holds the focus first. The keys move a selection through the files
and the hunks:

- `j` and `k` step through the hunks of the selected file. Past the last hunk,
  `j` rolls to the next file. Before the first, `k` rolls to the previous file.
- `}` and `{` jump to the next and the previous file.
- `g` and `G` go to the first and the last file.
- `pgup` and `pgdown` scroll the diff a page.

The selected file has a purple background. The selected hunk sits on a subtle
grey band, and its header is bold. The diff side scrolls to keep the selected
hunk in view.

`tab` moves the focus across the split: the diff, then the explanation, then the
prompt, then back to the diff. The focused pane has a blue header, so it is clear
which side takes the keys. The prompt bar shows "follow-up" when the prompt has
the focus.

`esc` closes the screen. In the prompt, `esc` returns the focus to the diff. The
sidebar hides while the screen is open, for the full width, and returns when the
screen closes.

The screen is modal, so it captures every key. The two-key sequences (`s`, `l`,
`o`, `d`) do not start while it is open, because their actions would move the
focus off the screen. So a review key never opens a panel or a dialog behind the
screen.

The mouse is modal too. A click on the diff, the explanation, or the prompt
focuses that side, and the wheel scrolls the side under the pointer. A click
never moves the focus off the screen.

## The explanation

The explanation side is the session output pane, in the review layout. So it
shows the whole conversation of the session, and it carries the block cursor and
the capped blocks of the normal output. See [output.md](output.md).

The explainer is the **live session**, not a separate call. So an explanation
uses the full context and the memory of the session, and it is a real turn of the
conversation. The ask shows in the pane as a prompt, and the reply shows below
it, the same as any turn.

`e` explains the selected hunk. `E` explains the whole selected file. The prompt
names a location, not the diff text:

```
Explain the change at internal/tui/diff.go:128-131. Keep the explanation short.
The change is against origin/HEAD. Read the file, or run
`git diff origin/HEAD -- internal/tui/diff.go`, for the surrounding context.
```

The location is `path:start-end`, from the new-side range of the hunk header. A
pure-deletion hunk has no new-side lines, so its range collapses to the anchor
line, and the session reads the diff for the removed lines. The prompt never
carries the hunk text, because the session reads the file itself.

A busy session queues the request, the same as any prompt.

## The blocks

The explanation is the output pane, so a long reply and a tool result cap the
same way they do in the pane. `[` and `]` move the block cursor, and `Enter` or
`Space` opens the block under the cursor. `j`, `k`, `u`, `d`, `g`, and `G` scroll
the pane. So the user opens the file the session read, or the diff it ran, inside
the explanation. See [output.md](output.md) and
[keys/navigation.md](keys/navigation.md).

## Follow-up questions

The explanation is a running conversation, not one answer. After the first
explanation, `tab` to the prompt, type a question, and press Enter. The prompt
goes to the same session, so it is the next turn of the same conversation, and
the reply shows in the output pane below the last one.
