# The code review screen

`s R` opens a full-pane review screen for the selected session. It shows a large
diff on the left and an explanation thread on the right. The user jumps between
the files and the hunks of the diff, and asks the live session to explain a hunk
or a file. See [diff.md](diff.md) for the diff the screen reads, and
[keys.md](keys.md) for the key model.

```
 review · alpha · +120 −30 · 3 files          │ Explanation
                                              │
 M internal/tui/app.go            +12 −3      │ › explain internal/tui/diff.go:128-131
   @@ -1,3 +1,4 @@ func A()                    │ This hunk caches the open-file map
    one                                       │ before the lookup, so the toggle
   +two                                       │ does not allocate on every keypress.
 A internal/git/hunks.go          +48 −0      │
   @@ -0,0 +1,48 @@                            │ › explain internal/git/hunks.go
   +package git                               │ ⋯ waiting
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

## Navigation

The diff side holds the focus first. The keys move a selection through the files
and the hunks:

- `j` and `k` step through the hunks of the selected file. Past the last hunk,
  `j` rolls to the next file. Before the first, `k` rolls to the previous file.
- `}` and `{` jump to the next and the previous file.
- `g` and `G` go to the first and the last file.
- `pgup` and `pgdown` scroll the diff a page.

The selected file has a purple background, and the selected hunk header is bold.
The diff side scrolls to keep the selected hunk in view.

`tab` moves the focus across the split: the diff, then the explanation, then the
prompt, then back to the diff. In the explanation, `j`/`k`/`g`/`G` scroll the
thread.

`esc` closes the screen. The sidebar hides while the screen is open, for the
full width, and returns when the screen closes.

## The explanation thread

The explainer is the **live session**, not a separate call. So an explanation
uses the full context and the memory of the session, and it is a real turn of
the conversation. The reply appears in the normal output pane as well as in the
thread.

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

A busy session queues the request, the same as any prompt. The thread shows
`⋯ waiting` until the reply starts.

## Follow-up questions

The thread is a running conversation, not one answer. After the first
explanation, `tab` to the prompt, type a question, and press Enter. The prompt
goes to the same session, so it is the next turn of the same conversation, and
the reply appends to the thread.

## How the reply reaches the thread

The screen mirrors the reply of the reviewed session into the last thread turn
while a turn is pending. It reads the assistant text (`render.ClassText`) and
the streaming tail (`Event.Partial`) of each event, and it stops at the end of
the turn. The reviewed session is always the selected session, because the
screen holds every key while it is open, so the selection cannot move. See
`captureExplain` in `internal/tui/review.go`.

Because the screen mirrors every turn of the session while it is open, a prompt
sent to the session from elsewhere also appends to the thread. This is
acceptable, because the thread is the conversation of the session.
