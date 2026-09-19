# Example rules

This folder holds example rules for a session. Each rule tells an agent how to
use a multiplexer feature in a common workflow. The rules do not ship on their
own, so you adopt the ones that fit your work.

A rule here is an example, not a default. The multiplexer injects only the rules
under `internal/mcp/injected/rules/`. See [mcp/rules.md](mcp/rules.md) for that
mechanism.

## How to adopt a rule

You adopt a rule in one of two ways:

- **Per project.** Copy the rule text into the `CLAUDE.md` of your project, or
  into a file under `.claude/rules/`. Then a session that runs in that project
  reads it.
- **Every session.** Add the rule as a file under
  `internal/mcp/injected/rules/`, and build the multiplexer. Then the
  multiplexer injects it into every session. See [mcp/rules.md](mcp/rules.md).

Change the wording to fit your team, for example the exact status names of your
board.

## The rules

| Rule | What it tells the agent |
|---|---|
| [example-rules/work-item-progress.md](example-rules/work-item-progress.md) | Link a session to a Jira or Linear work item, and move the item status as the work moves. See [work-items.md](work-items.md). |
