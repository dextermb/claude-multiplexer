# The two bars

A number that belongs to one session appears at the top, in the session bar. A
number that belongs to every session appears at the bottom, in the status bar.

## The session bar

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

## The status bar

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

## What the session bar drops first

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
