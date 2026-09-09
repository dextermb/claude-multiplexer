# The session list

The sidebar, and what a row means. For the keys that drive it, see
[keys.md](./keys.md).

## The pages

| Page | Read it for |
|---|---|
| [sessions/jobs.md](sessions/jobs.md) | Background jobs: the four places one shows, and the jobs dialog |
| [sessions/bars.md](sessions/bars.md) | The session bar and the status bar, and what each drops when narrow |

## The sections

With peering on, the sidebar splits into bands above the groups, so a row names
where its session runs. A band starts with a divider that names it. See
[../peers.md](../peers.md).

Two bands, in order:

1. **local sessions** — the sessions this host runs for itself.
2. **remote sessions** — the sessions that involve a peer:
   - a session that runs on a peer and streams into this host, and
   - a session this host runs for a peer, which started it through the peer
     listener. A hosted session shows muted, the same as an archived one, so a
     row names its kind without a divider of its own.

A row sorts to a band from two fields: a streamed session carries the peer as
its host, and a hosted session carries the hosted mark. Everything else is
local. The groups work inside each band, the same as without peering.

Without peering, and with no hosted or streamed session, the sidebar shows no
band dividers — it is the group list below. The new-session form gains a `host`
field only with peering on: choose `local` or a peer, and the session starts
there. The directory field follows the host: it clears for a peer, because the
directory is on the peer, and it returns to the local default for `local`.

## The groups

The list groups the sessions. Each group starts with a header, and its rows
follow it, indented by one column. A group holds one directory, the work of one
control session, or the sessions that involve one peer.

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
[../mcp/tools.md](../mcp/tools.md). The multiplexer records the caller, and the
session it made joins the group of that control session, and not the group of its
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

### A peer group

A remote session groups by its peer, not by its directory, because the directory
is on the other host and often a temporary one. So every session that runs on a
peer shares one group, and the header names the peer, for example `touchbar`. A
session this host runs for a client shares one group under the client name. The
group folds like any other, so you fold a whole peer into one line.

The header takes the client name from the credential store, because a hosted
session carries the client id, not the name. A name the store does not hold
falls back to the id. See [../peers.md](../peers.md).

### The order of the groups

A group that holds a live session comes first. Then come the groups that hold
stored sessions, and last the groups that hold archived sessions only. Inside a
group, the order is the order below.

### Folding a group

Every group folds, of either kind. Press `l f` to fold the group of the selected
session, and press `l f` again to unfold it. A folded group keeps its header, and
its rows go. Press `l u` to unfold every group, and `l F` to fold every group but
the one you are in. A click on a header folds or unfolds that group. See
[keys.md](./keys.md).

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

### Hiding the sidebar

Press `l c` to hide the whole sidebar. The output pane and the diff panel gain
its width, so a wide diff has more room. Press `l e` to show the sidebar again,
and press `l t` to toggle it. A hide moves the focus off the sidebar, to the
output. While the sidebar is hidden, `Tab` and `Esc` skip it, so the focus never
lands on a pane you cannot see. The multiplexer does not remember a hide. The
sidebar is open when the program starts. See [keys.md](./keys.md) and
[diff.md](./diff.md).

## Live rows and stored rows

The list holds two kinds of row. A **live** row is a session this program runs
now. A **stored** row is work from an earlier run, read from disk. Live rows
come first, then stored rows, then archived rows. Within each kind the order is
stable, so a row does not jump as its state changes.

## Reading a row

A row starts with a state glyph in the state colour, then the display name, then
`⇄` when the session may drive its neighbours, then `⏱` when a schedule spawned
the session, then `⚙n` when `n` background jobs run, then `⇢n` when `n` prompts
wait in the queue. The display name is the title
when the session has one, and the name when it does not. Press `s n` to set the
title. See [keys.md](./keys.md).
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
session that mark only when you mean it. See [mcp/grant.md](../mcp/grant.md).

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

Press `s a` to archive the selected session. The row leaves the list, and
nothing on disk is deleted. Press `l a` to show archived rows again, and `s a`
on one of them to bring it back.

A running session cannot be archived. Stop it first.
