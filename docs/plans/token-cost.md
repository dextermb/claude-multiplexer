# Token cost — spend less for the same work

**Status:** in progress. Effort 1 shipped, and it is described in
[../tui/sessions/bars.md](../tui/sessions/bars.md). Effort 2 shipped, and it is
described in [../scheduler.md](../scheduler.md). Effort 3 is dropped, because
the measurement under Resolved question 5 removed its reason. Effort 4 shipped
in part, and it is described in [../mcp/profiles.md](../mcp/profiles.md).
Effort 5 shipped, and it is described in
[../sessions/context.md](../sessions/context.md). Only one part of effort 4 is
ahead, and it waits on the open question below.

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

## What is left of effort 4

Two parts of the tool schema diet did not ship, and one of them never will.

**4a, trimming the descriptions, is rejected.** An audit of all 93 descriptions
found no cruft: no steering language, no worked examples, no cross-references,
and 80 of them are one or two sentences. The reference on tool use names
under-description as the common failure, so a blanket trim would cost more in
tool selection than it saves in tokens. The descriptions stay as they are.

**4c, search and dispatch, is not built.** It replaces the open schemas with a
`find_tools` and a `call_tool`, and it still waits on open question 6. The
minimal profile now covers the same ground for a session that needs few tools,
so measure the profile in use before you build the harder thing.

**A measurement note.** The `cmux run` command starts a bare child and attaches
no MCP server, so a transcript from it holds no `mcp__cmux__` tool. Measure the
tool surface from a session the manager spawns, and not from `cmux run`.
