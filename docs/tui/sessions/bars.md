# The three bars

A number that belongs to one session appears in the session bar, at the top of
the output pane. A number that belongs to every session appears in the band, the
top row of the screen. The status bar, the bottom row, holds the keys and the
last message.

## The session bar

The **session bar** heads the output, and it describes the selected session
only. It takes two rows. The top rule of the output pane holds the display name
(the title or the name), uppercase, and it inverts when the output has the
focus. The row under the rule holds the details. The left side names the
session: `control` when the session holds that grant, the model in use, the
permission mode, the effort, and the work item. The right side gives the
numbers: the state, the running-job count, the queue length, the tokens, the
cache hit rate, the cost, and the context fill.

The model and the permission mode come from the `init` event, so the bar names
what the child confirms, and not what the flags asked for. The two can differ.

The tokens (`11.6k in 0.6k out`) add up every turn, so they show the total work
billed. The cost and the tokens are lifetime figures: a session that you resume
carries the counts of its earlier runs, and does not restart them. See
[../../manager.md](../../manager.md). The context fill (`ctx 12.2k/200k (6%)`) is
different: it shows how full the window is now.

The cost is the price Claude Code reports for the work, against the first-party
API rate card. A subscription pays a plan fee in place of that price, so the
figure is what the same work costs through the API, and not a bill. See
[../../cost.md](../../cost.md), and for the rate card,
[../../caching.md](../../caching.md).

## The cache hit rate

The cache hit rate (`cache 94%`) is the share of the prompt tokens that came
from the prompt cache. It is the cache-read count over the input count, and the
input count is the sum of the three parts of the prompt.

The rate matters because the three parts have three prices. A cache read costs
about a tenth of the base input rate, and a cache write costs more than the base
rate. So `11.6k in` is cheap when the rate is high, and expensive when it is low,
and the token count alone cannot tell the two apart. For the prices, the
lifetime, and what invalidates the cache, see
[../../caching.md](../../caching.md).

The bar hides the rate until the session counts its first prompt token, because
a rate of zero and no data look the same. A stored session keeps its counts in
the meta file, so the rate survives a restart, and it covers the whole lifetime
of the session for a live row and a stored row alike.

The context fill comes from the last `assistant` message, not the `result`. One
`assistant` message reports the usage of one request. Its `input`, `cache_read`,
and `cache_creation` counts are the three parts of that one prompt, so their sum
is the size of the context now. The `result` usage is a session total, and its
cache-read count repeats the whole context every turn, so a sum of `result`
usage grows far past the window and is wrong for this number.

A local agent reports its own `assistant` usage, but that usage is the context of
the agent, not the session. So the fill skips a turn with a parent tool-use id,
and it tracks the session context alone. See
[../../protocol/jobs.md](../../protocol/jobs.md).

Each session is a separate child process, so each context fill is its own. When
the model window is not known, the bar shows the raw count only (`ctx 12.2k`).
The context fill shows for a live session only, because a stored session has no
running context.

## The band

The **band** is the top row of the screen. Its left side holds the product name
and the screens. Its right side describes the whole program:
`sessions (8 · 5 live) · 2 busy · 1 waiting · $4.2100 / 1d`. The session count
counts every row of the sidebar, then the live ones. The busy, waiting, and
failed counts take their state colours, and each one hides at zero, so a zero
never shows.

The total covers a window, and the band names it: `$12.3456 / 1d`. The default
window is the current UTC day, so the figure answers what today cost, and it
returns to zero at 00:00 UTC. The `costWindow` setting takes another window, and
`all` counts every session this host ever ran and names no window. For the
grammar, the ledger behind the figure, and the one estimate it holds, see
[../../cost.md](../../cost.md).

Two things stay out of the total. A remote session stays out, because the peer
account pays for it, and see [../../peers.md](../../peers.md). The search box and
the archive toggle stay out, because they filter the sidebar, and the total
describes the host and not the view.
When the band is too narrow, it drops the parts that call you least: a custom or
session part first, then the cost, then the session count, then the busy count.
The waiting and failed counts stay, because they ask for you.

## The status bar

The **status bar** is the bottom row. The left side gives the keys, each one in
brackets: `[n] new  [t] preset  [s] session`. While a key sequence waits, it
gives the actions of that target (see [../keys.md](../keys.md)). The right side
gives a transient message, for example `copied 3 lines`, or `docs archived
landing` when a session did it through a tool, for its moment.

The bar is a footer, so its palette is muted: grey on the surface grey, the keys
in white, and the brackets and dots dimmed.

When the window is too narrow, the key list sheds whole keys from its end and
ends with `…`, so the message keeps its text. Only when the key list is down to
`…` does the message drop.

## What the session bar drops first

The row of details always fits on one line. When the window is too narrow, the
two sides shed detail in turn, and the least useful item goes first:

```
 fake-model · auto  idle · ctx 12.2k/200k (6%) · 11.6k in 0.6k out · cache 94% · $0.2500
 fake-model · auto  idle · ctx 12.2k/200k (6%) · 11.6k in 0.6k out · cache 94%
 fake-model · auto  idle · ctx 12.2k/200k (6%) · 11.6k in 0.6k out
 fake-model         idle · ctx 12.2k/200k (6%)
 fake-model         idle
                    idle
```

The right side sheds from the end: the cost first, then the cache hit rate, then
the tokens, then the queue length, then the running-job count, then the context
fill, and last of all the state. The rate goes
before the tokens, because the tokens are the headline and the rate explains
them. The left side sheds the effort, then the permission mode, then
the model, and `control` last of all, because a session that can stop your work
is worth the space. The name is in the rule above, so it always stays, and only
when the name alone cannot fit is it cut short.

The context fill sits next to the state, so it stays until the bar is almost
empty. Seeing it is the point of the feature, so it outlives the cost and the
tokens. The per-session cost drops early. The total cost of every session lives
in the band.
