# The rules the multiplexer injects

The MCP initialize result carries an `instructions` string. Claude Code puts this
string in the system prompt of the child, under "MCP Server Instructions". So the
multiplexer tells every session how to use the `cmux` tools, and the user does
not keep the rule in a `CLAUDE.md`.

The rule belongs to the tool. A session that gets `mcp__cmux__set_working_dir`
also gets the instruction that says when to call it.

## Where the rules live

Each rule is one Markdown file under `internal/mcp/injected/rules/`. A
`//go:embed` directive in `internal/mcp/rules.go` reads the directory. The
`instructions()` function reads the files in name order, trims each one, and
joins them with a blank line. The server passes the result as
`ServerOptions.Instructions` when it builds a session server.

The join is in name order, so the result is the same on every start, and a test
can assert it.

## Add a rule

1. Add a Markdown file to `internal/mcp/injected/rules/`.
2. Name the file for the rule, in kebab-case.
3. Write the rule in the project language (see the rules under `.claude/rules/`).

The file is the content. This page describes the mechanism only, so it does not
copy the rule text.

## The rules today

| File | What it tells the session |
|---|---|
| `worktree-working-dir.md` | Call `set_working_dir` after you move into a worktree, and `unset_working_dir` after you collapse it, so `s f` and `s d` open the right directory. |
