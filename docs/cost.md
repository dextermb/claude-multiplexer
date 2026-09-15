# What the multiplexer spends

The status bar shows one figure for the whole host. By default it shows the
spend of today, from 00:00 to 24:00 UTC. This page says where that figure comes
from, and what it can and cannot know.

For the two bars and what else they draw, see
[tui/sessions/bars.md](tui/sessions/bars.md). For what a turn pays for, see
[caching.md](caching.md).

## The figure is an API price

Claude Code reports `total_cost_usd` for each prompt it answers, against the
first-party API rate card. A subscription pays a plan fee in place of that
price. So the figure is what the same work costs through the API, and it is not
a bill.

## The window

`costWindow` in the settings bounds the total. The value is a count and a unit:

| Value | What it counts |
|---|---|
| `1d` | Today, from 00:00 UTC. This is the default. |
| `7d` | Today and the six days before it, from 00:00 UTC on the first of them. |
| `1w` | This week, from 00:00 UTC on Monday. |
| `1m` | This month, from 00:00 UTC on the first. |
| `all` | Every session this host holds, with no window. |

The window is a calendar bucket, and not a rolling period. A rolling total never
settles, so the same day reads differently at each glance. A bucket resets at a
stated time instead, and the total returns to zero at 00:00 UTC each day. The
boundary is UTC and not local time, so the day rolls over at 01:00 local time in
British Summer Time.

An empty value takes `1d`, and so does a value the grammar does not accept.
See [config.md](config.md).

The bar names the window it shows (`$12.3456 1d`), and `all` shows no name.

## The ledger

A window needs a date for each cost, and nothing in a session carries one. A
`result` event holds the cost of one prompt and no timestamp. A count over the
208 transcripts on one host found 591 `result` events, and none of them holds a
`timestamp` field. `meta.json` holds one lifetime total for the whole session.

So the multiplexer writes the date itself. The pump appends one line to
`<root>/cost.jsonl` each time a snapshot moves a counter:

```
{"at":"2026-09-15T14:02:11Z","session":"alpha","cost":0.2506,
 "input":31314,"output":1113,"cache_read":248793,"cache_write":31314}
```

Each line holds the difference since the line before it, and not a total,
because a snapshot carries the lifetime figures. See [manager.md](manager.md).
A resumed session starts its difference from the totals in the meta file, so it
never writes its history again.

The ledger holds its lines in memory, and reads the file again when the file
grows by more bytes than this process wrote. So a second multiplexer on the same
state directory is seen, and a read never costs a scan.

Lines older than 90 days are dropped when the program starts. That covers the
longest window a setting can name.

## The backfill, and what it estimates

The ledger knows nothing before the day it arrives. So the first start after it
lands writes one line per stored session, dated at the last activity of that
session, for the whole lifetime cost of that session. Those lines carry
`"backfilled":true`.

A backfilled line is an estimate, and it is wrong in one direction: it puts the
whole life of a session into the day that session was last active. A session
that ran for three weeks and answered one prompt on the day of the backfill
counts all three weeks as that day. The estimate ages out of a `1d` window after
one day, and out of a `1m` window after a month.

Every line the pump writes afterwards is exact, because the multiplexer dates
the prompt as it sees it end.

## What stays out

A remote session is not counted. It runs on a peer, and the peer account pays
for it. See [peers.md](peers.md).

The search box and the archive toggle do not change the total. They filter the
sidebar, and the total describes the host.
