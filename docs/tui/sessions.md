# The session list

The sidebar, what a row means, and the two bars that carry the numbers.
For the keys that drive it, see [keys.md](./keys.md).

## Live rows and stored rows

The list holds two kinds of row. A **live** row is a session this program runs
now. A **stored** row is work from an earlier run, read from disk. Live rows
come first, then stored rows, then archived rows. Within each group the order is
stable, so a row does not jump as its state changes.

## Reading a row

A row starts with a state glyph in the state colour, then the name, then `⇢n`
when `n` prompts wait in the queue. The glyph tells the state at a glance:

| Glyph | State | Colour |
|---|---|---|
| `◌` | starting | blue |
| `⠋` (animated) | busy | amber |
| `●` | idle | green |
| `●` | failed | red |
| `○` | stored | gray |
| `·` | archived | faint gray |

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
only. The left side names it: the session, the model in use, and the permission
mode. The right side gives the numbers: the state, the queue length, the tokens,
and the cost.

The model and the permission mode come from the `init` event, so the bar names
what the child confirms, and not what the flags asked for. The two can differ.

The **status bar** at the bottom describes the whole program: how many sessions
run, how many are busy, the total cost, the number of dropped events, and the
keys.

So a number that belongs to one session appears at the top, and a number that
belongs to every session appears at the bottom.

### What the session bar drops first

The bar always fits on one line. When the window is too narrow, the two sides
shed detail in turn, and the least useful item goes first:

```
 alpha · fake-model · auto        idle · 11.6k in 0.6k out · $0.2500
 alpha · fake-model               idle · 11.6k in 0.6k out
 alpha · fake-model               idle
 alpha                            idle
 alpha
```

The right side sheds from the end: the cost first, then the tokens, then the
queue length, and last of all the state, so that only the name remains. The
left side sheds the permission mode, then the model. The name always stays, and
only when the name alone cannot fit is it cut short.

The per-session cost lives here, and it drops first. The total cost of every
session lives in the status bar, and it stays.
