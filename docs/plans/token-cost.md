# Token cost — spend less for the same work

**Status:** in progress. Effort 1 shipped, and it is described in
[../tui/sessions/bars.md](../tui/sessions/bars.md). Effort 2 shipped, and it is
described in [../scheduler.md](../scheduler.md). Effort 3 is dropped, because
the measurement under Resolved question 5 removed its reason. Efforts 4 and 5
are ahead, in build order, and each one lands on its own. Every effort measures
itself with the cache hit rate that effort 1 added.

---

## Why

A session pays for every token of every request. The multiplexer runs many
sessions at once, and a scheduled session runs for ever. So a small per-request
saving repeats thousands of times, and a large per-request waste does the same.

Three costs are structural, and the multiplexer controls all three:

1. **The fixed prefix.** The multiplexer injects 76 MCP tool schemas and two
   rule files into the system prompt of every session.
2. **The cache state.** A stable prefix reads from the prompt cache and pays
   about a tenth. A prefix that changes writes again and pays more than the base
   rate, and the write starts at the first byte that differs.
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
4. **Does Claude Code ask for the five-minute lifetime, or the one hour?** The
   one hour, for the session of a multiplexer. Claude Code 2.1.231 selects the
   lifetime from the query source, against an allowlist of `repl_main_thread*`,
   `sdk`, `auto_mode`, and `memdir_relevance`. A session of the multiplexer runs
   the main loop, so it matches. Three conditions turn the choice back to five
   minutes: an API key in place of a subscription login, an account in usage
   overage, and the environment variable `FORCE_PROMPT_CACHING_5M`. The variable
   `ENABLE_PROMPT_CACHING_1H` forces the hour.
5. **Does the cache need the session process to live?** No. The cache is
   server-side, and it keys on the bytes of the prefix, not on the session or the
   process. Two fresh sessions in one directory, one after the other, read 18465
   and 22421 tokens from the cache on their first turn. So a fresh spawn already
   reads most of its prefix, and a session that lives longer buys nothing.

## Open questions

6. **Does the search-and-dispatch shape break the model?** A model that cannot
   see a tool schema may not find the tool. Measure the call rate before and
   after. *Blocks:* the third option of effort 4 only.

---

## Effort 4 — the tool schema diet

**Worktree:** `just worktree tool-diet`.

`internal/mcp/tools.go` builds 76 tools. `mcp.OpenTools` holds 40 of them, and
every session carries all 40 whether or not it ever calls one. The schemas sit
in the system prompt of every request of every session.

### How large the share is

Measure the share before you size the work. A one-word prompt on a machine with
14 MCP servers reported 280 tools and 35079 input tokens. The 40 open tools of
the multiplexer are one part of that, and the other servers hold the rest, so
the multiplexer is a minority of the tool surface on a machine like this one.
Read the tool count from the `init` event of a fresh session, and size the work
from it before you start.

### Position matters more than size

A prompt renders in the order tools, then system, then messages, and a cache
entry matches a prefix of that render. So the tool list sits at the front of
every prefix, and a change to it invalidates the tools, the system, and the
messages together.

The two measured sessions show the cost of that. The first reported 280 tools,
and the second reported 344, because one remote MCP server answered in time for
the second and not for the first. The 64 extra schemas landed at the front of
the prefix, so the second session wrote 13798 tokens to the cache that a stable
tool list would have read.

The multiplexer cannot order another server to answer on time. It can keep its
own contribution small and deterministic, which is what the three options below
do.

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

**Resolve open question 6 first.** Measure the call rate before and after. A
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
| 4 | [../mcp/tools.md](../mcp/tools.md), [../config.md](../config.md) |
| 5 | [../sessions.md](../sessions.md), [../config.md](../config.md) |

Strike each effort from this file as it lands. Delete the file when the last
one is in. See [../../.claude/rules/plans.md](../../.claude/rules/plans.md).
