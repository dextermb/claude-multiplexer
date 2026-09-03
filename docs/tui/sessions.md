# The session list

The sidebar, what a row means, and the two bars that carry the numbers.
For the keys that drive it, see [keys.md](./keys.md).

## Live rows and stored rows

The list holds two kinds of row. A **live** row is a session this program runs
now. A **stored** row is work from an earlier run, read from disk. Live rows
come first, then stored rows, then archived rows. Within each group the order is
stable, so a row does not jump as its state changes.

## Reading a row

A row starts with a state glyph in the state colour, then the display name, then
`⇢n` when `n` prompts wait in the queue. The display name is the title when the
session has one, and the name when it does not. Press `R` to set the title; see
[keys.md](./keys.md). The glyph tells the state at a glance:

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

## The two bars

The **session bar** sits above the output, and it describes the selected session
only. The left side names it: the display name (the title or the name), the
model in use, and the permission mode. The right side gives the numbers: the state, the queue length, the tokens,
the cost, and the context fill.

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
A transient message (for example `copied 3 lines`) also appears on the left, for
its moment. The right side gives the keys, and the keys stay in one place.

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
queue length, then the context fill, and last of all the state, so that only the
name remains. The left side sheds the permission mode, then the model. The name
always stays, and only when the name alone cannot fit is it cut short.

The context fill sits next to the state, so it stays until the bar is almost
empty. Seeing it is the point of the feature, so it outlives the cost and the
tokens. The per-session cost drops early. The total cost of every session lives
in the status bar, and it stays.
