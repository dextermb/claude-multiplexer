# Cost totals — make the money numbers true

**Status:** in progress. Every question is closed. The token-cost plan landed
and was retired, so nothing blocks this work. See the last section for what that
change left behind.

The multiplexer shows a cost for one session in the session bar, and a total for
the program in the status bar. See [../tui/sessions/bars.md](../tui/sessions/bars.md).
Both numbers are wrong in the cases below.

---

## What is wrong today

### 1. A live session drops the counters of its earlier runs

`entry.view` copies the counters of the child process into the row:

```
internal/manager/entry.go:82-89    live.Turns = cached.Turns
                                   live.Cost = cached.Cost
                                   live.InputTokens = cached.InputTokens
                                   live.CacheReadTokens = cached.CacheReadTokens
                                   live.CacheWriteTokens = cached.CacheWriteTokens
                                   live.OutputTokens = cached.OutputTokens
```

`cached` is the snapshot of the running child, and a child starts at zero. The
lifetime total lives in `item.base` (`internal/manager/entry.go:99-106`), and
`base` reaches the meta file only:

```
internal/manager/store.go:27-32     next.Cost = item.base.cost + snap.Cost
internal/manager/lifecycle.go:102   item.base = totals{...} from the stored meta
```

So a stored row shows the lifetime figures, and the same session shows near zero
the moment you resume it. Six counters fall together: the cost, the turns, the
input tokens, the cache-read tokens, the cache-write tokens, and the output
tokens. [../manager.md](../manager.md) promises the opposite: "The counts are
lifetime totals: a resumed session adds to them, and does not restart them."

The cache hit rate (`internal/session/session.go:156`) divides one fallen
counter by another, so the percentage stays correct. It changes meaning instead:
a live row reports the rate of this run, and a stored row reports the rate of the
whole session. The reader has no way to see which.

### 2. The status bar total counts live sessions only

```
internal/manager/store.go:224      for _, snap := range m.Snapshots()
internal/manager/store.go:170      Snapshots ranges over m.order
```

`m.order` holds the live local sessions. So the total leaves out every stored
session, every archived session, and every remote session. It also leaves out
the `base` of each live session, because of fault 1. The number falls when a
session ends, and falls again when you archive one.

### 3. The figure is an API price, and not a bill

Claude Code computes `total_cost_usd` from the first-party API rate card. A
subscription pays a plan fee, and not that number. The bar gives no sign of
this, so the reader takes an API-equivalent figure for a spend. The rate card
itself is recorded in [../caching.md](../caching.md).

### 4. `turns` counts the agent loop, and not the prompts

`num_turns` counts the internal turns of one prompt. One measured prompt
reported `num_turns: 6`, and another reported 10. The session `datalake-dbt-2`
holds 377 turns from 31 prompts. The number is correct for what it measures, and
the documentation never says what it measures.

### What is right today

The accumulation itself is sound. A check of every stored session with more than
one result event (58 of 190) found `meta.cost_usd` equal to the sum of
`total_cost_usd` in its transcript, to the last decimal place, and the same for
the turn count. `total_cost_usd` reports one prompt, and not the session, so the
sum in `apply` does not double-count:

```
internal/session/apply.go:38       s.cost += ev.Result.TotalCostUSD
internal/session/apply.go:43-44    s.cacheReadTokens  += usage.CacheReadInputTokens
                                   s.cacheWriteTokens += usage.CacheCreationInputTokens
```

---

## Decisions

1. **Add the base at the pump, and not at each reader.** The pump
   (`internal/manager/manager.go:164`) is the one place every counter passes
   through. `entry.view`, the event bus, the wire snapshot a peer reads, `List`,
   and the MCP tools all read what the pump published, so one conversion fixes
   all of them.
   *Rejected:* a fix in `entry.view` alone. It leaves the event bus and the peer
   stream on the raw counters, so a row would hold two different values at two
   moments.

2. **`rememberSession` keeps the meta file identical.** After the pump adds the
   base, the snapshot already holds the lifetime total, so `rememberSession`
   assigns it in place of adding the base a second time. The values written to
   `meta.json` do not change, and the file needs no migration.

3. **Every "did this child work" guard reads the raw child snapshot.** Two
   guards ask that question, and a lifetime total answers it wrongly, because a
   resumed session carries a non-zero base from its first event:

   | Guard | What breaks without this |
   |---|---|
   | `internal/manager/manager.go:195` `final.Turns == 0` | A session that ran no turn keeps its directory. |
   | `internal/manager/idle.go:29` `snap.Turns == 0` | A resumed session with `stop_when_idle` armed stops itself before the prompt runs. |

   The second one is the sharper risk, because a schedule arms that stop. See
   [../scheduler.md](../scheduler.md).

4. **The context governor needs no change.** `maybeContextNotice`
   (`internal/manager/context.go:67`) reads `ContextTokens` only, and that
   counter describes the window now, and not a lifetime. `entry.total` must
   leave it alone for this reason.

5. **The status bar excludes a remote session.** A remote session runs on a peer,
   and the peer account pays for it. A session this host runs for a peer stays in
   the total, because this host pays for it. See [../peers.md](../peers.md).

6. **Say what the numbers are, in the documentation.** Record in
   [../tui/sessions/bars.md](../tui/sessions/bars.md) what the status bar total
   counts, that the cost is the API list price that Claude Code reports, and
   that a live row and a stored row measure the cache rate over the same
   lifetime once this lands. Record in [../sessions.md](../sessions.md) that a
   turn is an internal turn of the agent loop. Faults 3 and 4 are a question of
   words, and the bars keep their shape.

---

## Resolved — what the status bar total counts

**The total counts every local session the multiplexer holds: live, stored, and
archived.** That is option C below. The question was open, because the status
bar describes the whole program, and three answers were possible:

| Option | What it counts | What it costs |
|---|---|---|
| A | The live local sessions only, with the base added. | The number still falls to zero when the last session ends. |
| B | The live local sessions, and the stored sessions that are not archived. | The number falls when you archive a session. |
| C | Every local session the multiplexer holds, live, stored, and archived. | The number only grows. It reads 3862.90 USD today, from 190 sessions. |

C wins, because "global" then means one thing, and the number never moves for a
reason the reader cannot see. A and B both fall when the human puts work away,
which is the behaviour that started this plan.

Option C needs no disk read. The interface already refreshes the stored records
on a tick (`internal/tui/app.go:301`), and holds them in `m.stored`. So the total
reads the live snapshots and that cached list. It must not read `m.rows`, because
the search box and the archive toggle filter those rows.

---

## Data flow

The path of one counter, after the change. The new step is marked.

```
claude child            result event, one prompt of usage and cost
   |
   v
session.apply           six counters accumulate                (run totals)
   |
   v
session.Snapshot        Cost, Turns, InputTokens, CacheRead,
                        CacheWrite, OutputTokens               (run totals)
   |
   v
manager.pump            snap = item.total(ev.Snapshot)   <-- NEW: adds item.base
   |                                                          (lifetime totals)
   +--> item.setSnapshot(snap) --> entry.view --> Snapshots --> TUI row, MCP
   +--> rememberSession(item, snap) --> meta.json         (values unchanged)
   +--> maybeIdleAction(item, RAW)                        (decision 3)
   +--> maybeContextNotice(item, snap)                    (reads ContextTokens)
   +--> bus.publishLines(Event{Snapshot: snap}) --> TUI events, peer stream
```

The base itself moves one way only. `lifecycle.go` reads `meta.json` on a resume,
fills `item.base`, and nothing writes it again while the session runs.

The status bar total, under option C:

```
live:    manager.Snapshots()   lifetime totals, local sessions
stored:  m.stored              every meta the last tick found, archived included
remote:  excluded              the peer account pays
```

`Stored` already skips a session the manager holds live, so the two lists never
name the same session, and the sum never counts one twice.

---

## The build

Do the work in a worktree, `just worktree cost-totals`. See
[../../.claude/rules/worktrees.md](../../.claude/rules/worktrees.md).

**1. `internal/manager/entry.go`** — add `func (e *entry) total(snap
session.Snapshot) session.Snapshot`. It adds each field of `totals` to the
matching field of the snapshot, and returns the copy. That is the six counters
of `totals` (`entry.go:99-106`) and no more. It leaves `ContextTokens` alone,
for the reason in decision 4. Remove the six base-blind assignments from `view`,
and leave the rest of `view` as it is.

**2. `internal/manager/manager.go`** — in `pump`, wrap the snapshot once:
`snap := item.total(ev.Snapshot)`. Wrap `final` the same way. Pass the raw child
snapshot to `maybeIdleAction`, and keep the raw one for the `Turns == 0` delete
guard at line 195.

**3. `internal/manager/store.go`** — `rememberSession` assigns each counter of
the snapshot in place of adding the base, all six of them. Rewrite `TotalCost`
for the option the open question settles.

**4. `internal/manager/list.go:26`** — `List` reads `item.sess.Snapshot()`, which
is the raw child. Change it to `item.view()`, so a tool and the interface agree.

**5. `internal/tui/layout.go:703`** — the status bar reads the total. Under
option C the interface holds both halves already, so the sum belongs in the
interface, next to `m.stored`. Give it a name, `m.totalCost()`.

**6. Nothing changes** in `internal/session`, in the wire types, in the session
bar, or in `meta.json`. A peer reads the totals through the same wire snapshot.

### Verification

- A manager test: run a session, stop it, resume it, and assert that
  `Snapshot(name).Cost` holds the stored cost plus the new cost, and the same
  for the five other counters. The fake child emits a fixed cost per result,
  0.25 USD on the default path
  (`internal/testutil/fakeclaude/main.go:205`), so the expected value is exact.
  The test beside it, `internal/manager/manager_test.go:664`, asserts the meta,
  and must stay green without a change. That is the proof of decision 2.
- A manager test: a resumed session that arms `stop_when_idle` waits for its
  prompt, and does not stop at the first idle event. That is the proof of
  decision 3, and it is the regression this change can cause.
- A manager test: a session that finishes no turn keeps no directory, as it does
  today.
- A TUI test: build rows from one live session and one archived stored session,
  and assert the status bar figure. Set a search needle, and assert the figure
  does not move.

### Documentation

In the same change, and not after it:

- [../tui/sessions/bars.md](../tui/sessions/bars.md) — say what the status bar
  total counts, say that the cost is an API list price, and say that the cache
  rate of a live row covers the whole session.
- [../sessions.md](../sessions.md) — say that a turn is an internal turn of the
  agent loop, beside the counter list at line 202.
- [../manager.md](../manager.md) — say that a snapshot of a live session reports
  the lifetime totals, and name `entry.total` as the place the base enters. The
  lifetime promise at line 33 becomes true for the first time.

Delete this plan when the change lands, because nothing is left ahead of it. See
[../../.claude/rules/plans.md](../../.claude/rules/plans.md).

---

## What the token-cost change left behind

The token-cost plan shipped and was retired at `2dd8af3`. Its first effort added
`CacheReadTokens` and `CacheWriteTokens` to `session.Snapshot`, to `manager.Meta`,
and to `wire.Snapshot`, and it followed the shape the code already had. So the
two new counters carry the same fault as the cost, and this plan now converts six
counters in place of four. The cost of that is two lines in `entry.total`.

Two other parts of that change touch this one, and neither contradicts it. The
context governor added `maybeContextNotice` to the pump loop, which decision 4
covers. The session bar gained a `cache 94%` item at
`internal/tui/layout.go:580`, which reads the row and needs no edit.
