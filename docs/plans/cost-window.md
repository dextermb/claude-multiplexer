# A window for the status bar total

**Status:** in progress. Option B is chosen: the pump writes a cost ledger, and
the bar sums the lines inside the window.

The status bar total counts every session this host holds, live, stored, and
archived, with no time bound. See [../tui/sessions/bars.md](../tui/sessions/bars.md).
This plan adds a setting that bounds it: `1d`, `1w`, `1m`, or the whole history.
The default becomes `1d`, so the bar answers "what did today cost", and not
"what has this host ever cost".

---

## What the data allows

A cost has to carry a date before a window can select it. Today nothing does:

| Where the cost lives | The date it carries |
|---|---|
| `meta.json`, one lifetime total per session | `CreatedAt`, `LastActiveAt`, `ArchivedAt` |
| `session.Snapshot`, the live total | none |
| The transcript, one `result` event per prompt | none |

The last row is the one that hurts. A `result` event holds `total_cost_usd` for
one prompt, and it is the only per-prompt figure there is. A count over the 208
transcripts on this host found 591 `result` events, and none of them holds a
`timestamp` field. An `assistant` line and a `user` line do hold one, so a date
can be inferred from the line before, and that inference is a guess about which
line belongs to which turn.

So the multiplexer must write the date itself, or it must accept a coarse one.

---

## Resolved — how a cost gets its date

**Option A — date the whole session by `LastActiveAt`.**

The total counts a stored session when its last activity falls inside the
window, and it counts the lifetime cost of that session.

- Builds in one change. No new file, and no new writer.
- Wrong for a session that spans the edge. A session that ran for three weeks
  and answered a prompt this morning puts all three weeks into `1d`.
- Wrong in the other direction for a session that ran a month ago and has not
  been touched. It leaves `1m` entirely, cost and all.

**Option B — a cost ledger, written as the work happens.**

The pump appends one line per `result` event to `<root>/cost.jsonl`, with the
time the multiplexer saw it, the session name, and the cost of that prompt. The
bar sums the lines inside the window.

- Exact from the day it ships, for a live session and a stored one alike.
- A new file, a new writer in the pump, and a prune rule so the file does not
  grow without end.
- It knows nothing about the past. A backfill can only date the 190 stored
  sessions at their `LastActiveAt`, which is option A applied once, to history.

**The recommendation is B, with the option A backfill.** The bar then tells the
truth about the work it saw, the history is as good as the history can be, and
the backfilled rows age out of a `1m` window after one month. The cost is one
new file and one new writer.

**The `1d` default makes A hard to defend.** Under A, a session that you resume
today puts its whole lifetime into today. The session `datalake-dbt-2` holds
596.43 USD over many weeks, so one prompt to it this morning would read as
596.43 USD of spend today. A day is the window where the difference between a
dated cost and a dated session is at its worst, and the default puts every
reader in front of it.

Under B the same resume adds only the cost of that prompt, because the ledger
dates each prompt as the pump sees it. The backfill still dates history by
session, so the first day after this lands carries that error once: a session
last active today contributes its lifetime cost to today, and the line is marked
`"backfilled":true` so the page can say so.

---

## Decided, whichever option wins

1. **The window is a calendar bucket in UTC, and not a rolling period.** `1d`
   is the day from 00:00 to 24:00 UTC. `1w` is the week from 00:00 UTC on
   Monday, and `1m` is the month from 00:00 UTC on the first.
   A count above one adds the buckets before the current one, so `7d` starts at
   00:00 UTC on the day six days back, and it ends now.
   *Rejected:* the rolling period, `now` minus 24 hours. A rolling total never
   settles, so the same day reads differently at each glance, and two hosts
   never agree on what a day is. A bucket resets at a stated time instead.
   The reset is in UTC, so the bar rolls over at 01:00 local time in British
   Summer Time, and at 00:00 local in winter.

2. **The default is `1d`.** An absent setting means the current UTC day, so the
   bar shows the spend of today. The value `all` restores the all-time total
   this plan starts from, and nothing else brings it back.
   *The consequence:* the number in the bar falls to zero at 00:00 UTC every
   day. That is the feature, and [../tui/sessions/bars.md](../tui/sessions/bars.md)
   must say it, or the first rollover reads as lost data.
3. **The setting is `costWindow`, a string.** The grammar is a number and a
   unit, `d`, `w`, or `m`, so `1d`, `7d`, `2w`, and `1m` are all valid. The
   value `all` counts the whole history. `internal/config` validates the string,
   and rejects anything else, in the way `contextAction` is validated. See
   [../config.md](../config.md).
4. **The bar names the window.** `$12.3456 1d`, so the figure is never read as
   an all-time total. Only `all` shows no suffix.
5. **A remote session stays out**, as it does now, because the peer account pays
   for it.
6. **The window does not change the session bar.** A per-session cost stays a
   lifetime figure, because that is what a session is.

---

## The build, under option B

Do the work in a worktree, `just worktree cost-window`.

**1. The ledger.** `internal/manager/ledger.go` holds an append-only writer for
`<root>/cost.jsonl`:

```
{"at":"2026-09-15T14:02:11Z","session":"alpha","cost":0.2506,
 "input":31314,"output":1113,"cache_read":248793,"cache_write":31314}
```

Each line is a delta, and not a total. The pump already computes the lifetime
snapshot, so the delta is the new snapshot minus the last one the entry wrote.
`entry` gains one field to hold that, beside `base`.

**2. The reader.** `Manager.CostSince(start time.Time) float64` sums the lines
at or after `start`. The interface calls it on the stored tick it already runs,
and not once per frame, so the file is read on a tick and never in the render
path. See `internal/tui/app.go:301`.

**2a. The bucket.** `config.WindowStart(value string, now time.Time)` returns the
start of the window and whether one applies. It truncates `now` in UTC to the
day, the Monday, or the first of the month, and then steps back the count minus
one. `all` reports no window. This function holds the whole of decision 1, so
one test pins every boundary.

**3. The prune.** The manager drops lines older than 90 days when it starts, so
the file stays bounded and still covers the longest window a setting can name.

**4. The backfill.** On the first start after this lands, and only when the
ledger is absent, write one line per stored session at its `LastActiveAt`, for
its lifetime cost. Mark those lines `"backfilled":true`, so the page can say
which part of a window is an estimate.

**5. The setting.** `CostWindow string` in `config.Config`, validated, and
documented in the table in [../config.md](../config.md). An absent value reads
as `1d`, so the default lives in one function beside the grammar, and not in
each reader.

**6. The bar.** `Model.totalCost` takes the window. With `all` it sums the
snapshots and the stored records, which is the whole-history path this plan
starts from. With any other value it calls `CostSince` with the bucket start.
`internal/tui/cost.go` holds both paths.

### Verification

- A unit test for the window grammar: `1d`, `7d`, `2w`, `1m`, `all`, an absent
  value that reads as `1d`, and a rejection of `1y`, `d1`, and an empty unit.
- A unit test for the bucket, against a fixed `now` and no call to the clock:
  `1d` starts at 00:00 UTC today, `1d` at 00:30 UTC starts half an hour back,
  `1w` starts on Monday even when `now` is a Sunday, `1m` starts on the first,
  and `7d` starts six days before today at 00:00 UTC. One case runs at 23:59
  UTC on 31 December, so the year rolls over.
- A test that a time zone does not move the boundary: the same `now` in a
  `+13:00` zone gives the same UTC bucket start.
- A ledger test: three lines at three dates, and a sum that takes only the lines
  inside the window.
- A manager test: two turns write two ledger lines, and the second line holds
  the delta and not the total.
- A TUI test: the bar shows the `1d` suffix by default, and no suffix under
  `all`.
- A backfill test: a ledger absent, two stored sessions, and two lines written
  at their `LastActiveAt`.

### Documentation

- [../tui/sessions/bars.md](../tui/sessions/bars.md) — what the total counts
  under a window, that the default is the current UTC day, that the figure
  therefore returns to zero at 00:00 UTC, and that a backfilled figure dates a
  whole session at its last activity.
- [../config.md](../config.md) — the `costWindow` key, its grammar, its `1d`
  default, and that the boundary is UTC and not local time.
- A new page, `docs/cost.md`, for the ledger: what it holds, where it lives, how
  it is pruned, and why a `result` event cannot be dated from the transcript.
  Add it to the index in the same change.

### The build, under option A

`Model.totalCost` filters `m.stored` by `LastActiveAt`, and keeps every live
session, because a live session is active now. The setting, the bucket, the bar
suffix, and their tests are the same. There is no ledger, no prune, and no
backfill, and `docs/cost.md` is one paragraph in `bars.md` in place of a page.

Under this option the page must say that the window selects sessions and not
prompts, and that a `1d` total therefore holds the whole life of every session
touched today. That sentence is the reason the recommendation is B.
