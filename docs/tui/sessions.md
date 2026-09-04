# The session list

The sidebar, what a row means, and the two bars that carry the numbers.
For the keys that drive it, see [keys.md](./keys.md).

## The groups

The list groups the sessions. Each group starts with a header, and its rows
follow it, indented by one column. A group holds either one directory or the
work of one control session.

### A directory group

The key of a directory group is the repository above the directory of the
session. The program walks up to the nearest `.git`. So a session in the
repository, a session in a subdirectory, and a session in a worktree all share
one group. A directory with no repository above it is a group of its own.

The header names the group with the last element of that path, for example
`claude-multiplexer`. When two groups have the same last element, both names
grow one element to the left, so `~/a/api` and `~/b/api` read `a/api` and
`b/api`. The number on the right of the header counts the rows of the group.

### A creator group

A control session can start another session with the `create_session` tool. See
[../mcp.md](../mcp.md). The multiplexer records the caller, and the session it
made joins the group of that control session, and not the group of its
repository. So one group holds the work of one agent, whatever repository each
member runs in.

The control session is the first row of its own group, whatever its state. Its
header carries the `⇄` mark before the display name of that session, and the
row itself drops the mark, because the header already gives it.

A control session that created nothing keeps its directory group. The grant
alone makes no group, and the group appears with the first session the control
session creates.

The multiplexer remembers the creator of a session, so a stored session still
joins its group after a restart. A session you started yourself has no creator.
See [../manager.md](../manager.md).

### The order of the groups

A group that holds a live session comes first. Then come the groups that hold
stored sessions, and last the groups that hold archived sessions only. Inside a
group, the order is the order below.

### Folding a group

Every group folds, of either kind. Press `z` to fold the group of the selected
session, and press `z` again to unfold it. A folded group keeps its header, and
its rows go. Press `Z` to unfold every group. When no group is folded, `Z`
folds every group but the one you are in. A click on a header folds or unfolds
that group. See [keys.md](./keys.md).

A folded header carries two marks that an open header does not need. The mark
`▸` replaces `▾`, and the glyph of the most urgent row it hides comes before
the count. The order of urgency is waiting, busy, failed, starting, and then
idle. So a session that waits for your answer is still visible when you fold
its group.

The selection is always a row you can see. Fold the group that holds the
selection, and the selection moves to the first row of the next group. When
there is no next group, it moves back to the row above. Start a session in a
folded group, and that group unfolds.

The multiplexer does not remember a fold. Every group is open when the program
starts.

## Live rows and stored rows

The list holds two kinds of row. A **live** row is a session this program runs
now. A **stored** row is work from an earlier run, read from disk. Live rows
come first, then stored rows, then archived rows. Within each kind the order is
stable, so a row does not jump as its state changes.

## Reading a row

A row starts with a state glyph in the state colour, then the display name, then
`⇄` when the session may drive its neighbours, then `⚙n` when `n` background jobs
run, then `⇢n` when `n` prompts wait in the queue. The display name is the title
when the session has one, and the name when it does not. Press `R` to set the
title; see [keys.md](./keys.md).
The glyph tells the state at a glance:

| Glyph | State | Colour |
|---|---|---|
| `◌` | starting | blue |
| `⠋` (animated) | busy | amber |
| `●` | idle | green |
| `?` | waiting | magenta |
| `●` | failed | red |
| `○` | stored | gray |
| `·` | archived | faint gray |

A `waiting` row asked a question and holds for the answer. See
[input.md](./input.md).

A row marked `⇄` can prompt, stop, and archive the other sessions. Give a
session that mark only when you mean it. See [mcp.md](../mcp.md).

The busy glyph is the dot spinner (`⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏`). It turns while any
session runs a turn, and it is the same spinner the output pane shows for
`thinking…`. The selected row is shown with a blue background, and the focused
pane (the list, the prompt, or the output) carries a blue left edge.

Select a stored row and the pane shows that conversation, replayed from its
transcript. It is a record: you cannot type into it. Press `Enter` and the
session starts again with `--resume`, under the same name, writing to the same
transcript. The pane keeps the history and marks the join with `— resumed —`.

A session you stop stays in the list until you leave the program. `Enter` brings
it back the same way.

Only a session that finished at least one turn is remembered. See
[manager.md](../manager.md).

## Archived rows

Press `a` to archive the selected session. The row leaves the list, and nothing
on disk is deleted. Press `A` to show archived rows again, and `a` on one of
them to bring it back.

A running session cannot be archived. Stop it first.

## Background jobs

Claude Code can run a shell command in the background. The multiplexer shows
each background job in four places. See [../sessions.md](../sessions.md) for the
job model, and [../protocol.md](../protocol.md) for the wire events.

- The output pane marks each job in order. A start line reads `⚙ started ·
  <description>`. A stop line reads `⚙ done · <id>`, or `failed`, or `killed`.
- The sidebar row shows `⚙n` for `n` running jobs, next to the queue badge. The
  badge clears when the last job stops.
- The session bar shows a `⚙n` segment while jobs run, next to the queue segment.
- The side panel lists every job above the task list. See [tasks.md](tasks.md).

Press `J` to open the jobs dialog for the selected session. The dialog lists
every job, the running ones first, then the finished ones, in start order. Each
row shows a status glyph, the status word, and the job description. Press `esc`
to close it. The dialog draws in the pane, so the sidebar stays on the screen.
See [keys.md](./keys.md) and [../tui.md](../tui.md).

## The two bars

The **session bar** sits above the output, and it describes the selected session
only. The left side names it: the display name (the title or the name),
`control` when the session holds that grant, the model in use, and the
permission mode. The right side gives the numbers: the state, the running-job
count, the queue length, the tokens, the cost, and the context fill.

The model and the permission mode come from the `init` event, so the bar names
what the child confirms, and not what the flags asked for. The two can differ.

The tokens (`11.6k in 0.6k out`) add up every turn, so they show the total work
billed. The context fill (`ctx 12.2k/200k (6%)`) is different: it shows how full
the window is now.

The context fill comes from the last `assistant` message, not the `result`. One
`assistant` message reports the usage of one request. Its `input`, `cache_read`,
and `cache_creation` counts are the three parts of that one prompt, so their sum
is the size of the context now. The `result` usage is a session total, and its
cache-read count repeats the whole context every turn, so a sum of `result`
usage grows far past the window and is wrong for this number.

Each session is a separate child process, so each context fill is its own. When
the model window is not known, the bar shows the raw count only (`ctx 12.2k`).
The context fill shows for a live session only, because a stored session has no
running context.

The **status bar** at the bottom describes the whole program. The left side
gives the state: how many sessions run, how many are busy, and the total cost.
A transient message (for example `copied 3 lines`, or `docs archived landing`
when a session did it through a tool) also appears on the left, for its moment.
The right side gives the keys, and the keys stay in one place.

The bar is a footer, so its palette is muted. The default text is grey, and
colour marks only the cost, which keeps the green of the session bar so the same
number reads the same in both places. The busy count is hidden when no session
is busy, so a zero never shows.

When the window is too narrow for both sides, the keys go first. Then the left
side sheds from its end (the message, then the cost, then the busy count), and
the session count always stays.

So a number that belongs to one session appears at the top, and a number that
belongs to every session appears at the bottom.

### What the session bar drops first

The bar always fits on one line. When the window is too narrow, the two sides
shed detail in turn, and the least useful item goes first:

```
 alpha · fake-model · auto        idle · ctx 12.2k/200k (6%) · 11.6k in 0.6k out · $0.2500
 alpha · fake-model · auto        idle · ctx 12.2k/200k (6%) · 11.6k in 0.6k out
 alpha · fake-model               idle · ctx 12.2k/200k (6%)
 alpha · fake-model               idle
 alpha                            idle
 alpha
```

The right side sheds from the end: the cost first, then the tokens, then the
queue length, then the running-job count, then the context fill, and last of all
the state, so that only the name remains. The left side sheds the effort, then the permission mode, then
the model, and `control` last of all, because a session that can stop your work
is worth the space. The name always stays, and only when the name alone cannot
fit is it cut short.

The context fill sits next to the state, so it stays until the bar is almost
empty. Seeing it is the point of the feature, so it outlives the cost and the
tokens. The per-session cost drops early. The total cost of every session lives
in the status bar, and it stays.
