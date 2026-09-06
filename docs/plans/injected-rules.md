# Plan: the multiplexer enforces the worktree rule

Status: **in progress**. The injected rule shipped. See
[../mcp/rules.md](../mcp/rules.md) for the mechanism. Only the automation below
stays ahead, and the injected rule is the required fallback for it.

# Shipped

The multiplexer injects its rules into every session, through the MCP
`instructions` field. The user no longer keeps the worktree rule in a
`CLAUDE.md`. For the mechanism, the file layout, and how to add a rule, see
[../mcp/rules.md](../mcp/rules.md).

# Ahead: enforce the working directory without the model

The injected rule asks the model to call `set_working_dir`. The model can forget.
The goal below is to set the working directory without the model. The rule stays
the required fallback in every option, because automation can fail or be off.

**Option A — read `Init.CWD`.** The init event carries the child cwd
(`internal/protocol/event.go:42`), decoded but unused. It equals the start
directory, so it does not find a worktree made later. **Rejected.**

**Option B — watch the event stream.** The manager pumps every event
(`internal/manager/manager.go:284`) and already reads Bash tool-use blocks
(`internal/protocol/bash.go`). It could match `git worktree add` and `cd`, then
resolve with the worktree logic the TUI has (`internal/tui/group.go:40`). Cost:
it parses a shell string, so `pushd`, a subshell, or `cd a && cd b` can fool it.
It cannot read the real cwd of the persistent shell.

**Option C — a Claude Code hook.** The multiplexer spawns the child, so it can
write a `settings.json` with a `PostToolUse` hook and pass `--settings`. Claude
Code gives the hook the real cwd, so no shell parsing. The hook reports the cwd
to the multiplexer (for example, a hidden `multiplexer` subcommand that reads the
hook JSON and calls back through the loopback endpoint the session already
knows). Cost: a new mechanism, and it depends on the hook support of the
installed Claude Code.

**Recommendation.** Prefer Option C, because it uses the cwd that Claude Code
reports, not a guess from a shell string. Keep the rule as the fallback, because
a hook can be off.

# When this plan is done

Option C is not decided. When the automation lands, move its durable parts into
`docs/` and delete this plan. Until then, this plan holds the open question.
