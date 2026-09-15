# Token cost — spend less for the same work

**Status:** in progress. Effort 1 shipped, and it is described in
[../tui/sessions/bars.md](../tui/sessions/bars.md). Efforts 2 to 5 are ahead,
in build order, and each one lands on its own. No effort depends on a later one.
Every effort measures itself with the cache hit rate that effort 1 added.

---

## Why

A session pays for every token of every request. The multiplexer runs many
sessions at once, and a scheduled session runs for ever. So a small per-request
saving repeats thousands of times, and a large per-request waste does the same.

Three costs are structural, and the multiplexer controls all three:

1. **The fixed prefix.** The multiplexer injects 76 MCP tool schemas and two
   rule files into the system prompt of every session.
2. **The cache state.** A resumed session reads its prefix from the prompt
   cache and pays about a tenth. A fresh session writes the prefix again and
   pays more than the base rate.
3. **The context size.** Every turn re-reads the whole context. A session at
   180k tokens costs many times the same session at 12k.

The multiplexer does not call the Anthropic API. It starts `claude` child
processes. So it cannot set a cache control header. It can control which
sessions live, which die, what goes in their prompts, and which model runs
them. That is enough.

---

## Decisions

- **Measure before you tune.** Effort 1 ships first and alone. Every later
  effort states the number it must move, and the number must exist first.
- **No automatic prompt rewriting.** The multiplexer will not edit a human
  prompt to shorten it. A silent rewrite breaks the one thing a user trusts.
- **A cost control is opt-in.** A new limit that stops a session, or compacts
  it, is off until a setting turns it on. A tool that saves money and surprises
  the user is a tool the user turns off.
- **Transcript compression is out of scope.** Gzip on the transcript files
  saves disk and saves zero tokens, because the file is never sent to a model.
  Do it as storage work, not as cost work.

---

## Resolved questions

1. **The price ratios.** A cache read costs about 0.1x the base input rate. A
   cache write costs 1.25x at the five-minute lifetime, and 2x at the one-hour
   lifetime. So a five-minute entry pays for itself on the second request, and a
   one-hour entry needs three. Source: the Anthropic prompt-caching reference.
2. **The prompt cache lifetime.** The provider offers two lifetimes only: five
   minutes, and one hour. There is no longer option, and the multiplexer cannot
   select one, because Claude Code sets the cache control header. So a keep-warm
   rule can hold a session process for a whole working day, but the prompt cache
   under it expires after one hour of silence at the most.
3. **Does a resume hit the cache?** Yes, inside the lifetime. A resume that
   comes back after the lifetime replays the history and writes the cache again.

## Open questions

4. **Does the search-and-dispatch shape break the model?** A model that cannot
   see a tool schema may not find the tool. Measure the call rate before and
   after. *Blocks:* the third option of effort 4 only.
5. **Does Claude Code ask for the five-minute lifetime, or the one hour?** The
   two have different break-even points, so the answer sets the cadence at which
   a keep-warm session stops paying. Read it from the cache-write count of a
   session that idles for ten minutes, with effort 1 in place. *Blocks:* the
   threshold of effort 3, not the build.

---

## Effort 2 — route a schedule to a cheap model

**Worktree:** `just worktree schedule-model`.

`manager.Schedule` already holds `Model` and `Effort`
(`internal/manager/schedule.go:22`), and `fireSchedule` already passes both to
`Spawn`. Nothing is missing except a default and a way to see it. A recurring
housekeeping prompt that runs on the most expensive model is pure waste, and
today nothing shows which model a schedule uses.

### The build

1. Add `defaultScheduleModel` to `config.Settings`, beside `defaultModel`. Add
   it to the table in [../config.md](../config.md).
2. `CreateSchedule` applies it when `ScheduleSpec.Model` is empty.
3. Show the model on the schedule row in the sidebar, so a costly schedule is
   visible without opening a file.

### How it is verified

A test asserts that a schedule created with no model takes the setting, and
that a schedule created with a model keeps it.

---

## Effort 3 — keep the cache warm across scheduled runs

**Worktree:** `just worktree schedule-keep-warm`.

A scheduled run arms `stop_when_idle`, so the process ends after the turn. See
[../sessions.md](../sessions.md). That is correct for a nightly schedule, and
wrong for a schedule that fires every two minutes, because each fire then pays
to write the whole prefix again.

### The limit of this effort

The prompt cache lives for one hour at the most. So a keep-warm session helps a
schedule that fires more often than the cache lifetime, and it helps nothing
else. A schedule that fires every four hours writes the cache again on every
fire, whether the process lives or dies. The setting is therefore a cadence
control, and not a working-day control.

### The negatives

Each one is a reason to keep the setting off by default.

1. **A spawn-mode schedule breaks.** `runDue` skips a fire when the last session
   of the schedule is still live (`internal/manager/scheduler.go`). A warm
   session is always live, so every fire after the first is skipped. The setting
   must therefore be valid in reuse mode only, and `CreateSchedule` must refuse
   it when `Session` is empty.
2. **The context grows without a limit.** A reuse session keeps its history
   across fires. A cache read is cheap, but the context it reads grows every
   fire, so the cost per fire climbs in a straight line. This is the problem of
   effort 5, and a warm schedule reaches it faster than a human session does.
3. **Auto-archive never runs.** The sweep archives a stopped session only. A
   warm session is never stopped, so it never archives, and it holds its
   transcript and its process for ever. See [../sessions.md](../sessions.md).
4. **One live process per warm schedule.** Each one holds memory, file handles,
   and an MCP connection. Ten warm schedules are ten live child processes.
5. **A restart drops the state.** The arm lives in memory on the live session,
   so a restart of the multiplexer loses every warm session at once.

### The build

1. Add `keepWarm bool` to `Schedule` and `ScheduleSpec`, and to the
   `create_schedule` and `update_schedule` tools.
2. `CreateSchedule` and `UpdateSchedule` refuse `keepWarm` on a spawn-mode
   schedule, with a clear error. See negative 1.
3. When `keepWarm` is set, `fireSchedule` does not arm the stop.
4. When `keepWarm` is set and the cron fires less often than one hour, warn once
   at creation, because the cache is cold by the next fire. Do not override the
   choice.

### How it is verified

A test asserts that `keepWarm` on a spawn-mode schedule fails, and that a
reuse-mode fire with `keepWarm` set does not arm the stop. Then run one schedule
at a one-minute cron, with `keepWarm` on and then off, and compare the cache hit
rate from effort 1 across ten fires. Record both runs as evidence.

---

## Effort 4 — the tool schema diet

**Worktree:** `just worktree tool-diet`.

`internal/mcp/tools.go` builds 76 tools. `mcp.OpenTools` holds 40 of them, and
every session carries all 40 whether or not it ever calls one. The schemas sit
in the system prompt of every request of every session.

Position makes this worse than size alone. A prompt renders in the order tools,
then system, then messages, and a cache entry matches a prefix of that render.
So the tool list sits at the front of every prefix, and a change to it
invalidates the tools, the system, and the messages together. The tool list is
therefore both the largest fixed cost and the most cache-sensitive part of the
prompt.

A session picks its profile once, at spawn, so a profile never changes a live
conversation and never invalidates a live cache. A profile change reaches a
running session on its next start.

Three options. Build them in order, and stop when the hit rate stops moving.

### 4a — trim the descriptions

Cut every tool description to one line, and point at `get_api_docs` for the
detail. This is the smallest change, it risks nothing, and it needs no new
concept.

### 4b — tool profiles

Replace the `control bool` in `AllowedTools` and `build` with a profile:

| Profile | What it holds |
|---|---|
| `minimal` | The read tools, and `get_api_docs` |
| `standard` | `minimal`, plus config, layout, schedule, and share |
| `control` | `standard`, plus the control, credential, and peer tools |

A new setting `defaultToolProfile` picks the default, and the create-session
form and `create_session` take an override. A control session keeps its grant,
so the profile never widens what a session may do.

### 4c — search and dispatch

Replace the 40 open schemas with two: `find_tools(query)` returns the matching
names and their schemas, and `call_tool(name, args)` runs one. The prefix then
holds two schemas rather than 40, at the cost of one extra turn the first time
a session needs a tool.

**Resolve open question 4 first.** Measure the call rate before and after. A
model that no longer finds `rename_session` has cost more than it saved.

### How it is verified

Record the reported context size of the first `assistant` event of a fresh
session, before and after each option. That number is the prefix size, and it
should fall at each step.

---

## Effort 5 — the context-fill governor

**Worktree:** `just worktree context-governor`.

The session bar already shows `ctx 12.2k/200k (6%)`, from `contextTokens` in
`apply.go:34`. Nothing acts on it. A long session grows its context every turn,
and every turn pays for the whole of it, so the cost per turn climbs while the
work per turn does not.

### The workflow

```
   context fill crosses warnPercent
              |
              v
     a notice in the status bar,  <---- no state change, the session runs on
     and a flag on the sidebar row
              |
   context fill crosses actPercent
              |
              v
        action, by setting:
          "notify"  -> a notice only, the default
          "hold"    -> refuse a new prompt until the human clears the hold
```

A hold is a guard the human clears. The multiplexer does not compact the
session on its own, because a compaction throws away context the human may
need, and a silent loss is worse than a cost.

### The build

1. Add `contextWarnPercent`, `contextActPercent`, and `contextAction` to
   `config.Settings`. All three are absent by default, which means off.
2. The per-session pump compares the fill on every snapshot, and raises the
   notice once per crossing, not once per event.
3. A hold blocks `Send` with a clear error, and the human clears it with a key.

### How it is verified

A test drives a session past each threshold and asserts one notice per
crossing, and asserts that `Send` fails while the hold is set.

---

## When this lands

Each effort moves its durable part into `docs/` in the same change:

| Effort | Where the content goes |
|---|---|
| 2 | [../scheduler.md](../scheduler.md), [../config.md](../config.md) |
| 3 | [../scheduler.md](../scheduler.md), [../sessions.md](../sessions.md) |
| 4 | [../mcp/tools.md](../mcp/tools.md), [../config.md](../config.md) |
| 5 | [../sessions.md](../sessions.md), [../config.md](../config.md) |

Strike each effort from this file as it lands. Delete the file when the last
one is in. See [../../.claude/rules/plans.md](../../.claude/rules/plans.md).
