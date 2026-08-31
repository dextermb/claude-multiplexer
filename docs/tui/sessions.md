# The session list

The sidebar, what a row means, and the two bars that carry the numbers.
For the keys that drive it, see [keys.md](./keys.md).

## Live rows and stored rows

The list holds two kinds of row. A **live** row is a session this program runs
now. A **stored** row is work from an earlier run, read from disk. Live rows
come first, then stored rows, newest first.

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
mode. The right side gives the numbers: the state, the queue length, the number
of turns, the duration of the last turn, the tokens, and the cost.

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
 alpha · fake-model · auto                      idle · 1 turn · 7ms · $0.2500
 alpha · fake-model                     idle · 1 turn · 7ms · $0.2500
 alpha · fake-model         idle · 1 turn · $0.2500
 alpha           idle · 1 turn · $0.2500
 alpha  · $0.2500
```

The order is: the permission mode, the tokens, the model, the duration, the
turn count, the queue length, and last of all the state. The name and the cost
always stay. Only when the name alone cannot fit is it cut short.
