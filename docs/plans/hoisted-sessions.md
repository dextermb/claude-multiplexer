# Hoisted sessions — what is still ahead

**Status:** shipped. The feature is built and tested on branch
`hoisted-sessions`. What it does is described in
[../peers/hoisted.md](../peers/hoisted.md). This file holds only the work that is
not built. Delete it when nothing is left.

The build landed: the credential type in `internal/config`, the environment
builder in `internal/session` (`env.go`), the lent-key metadata in
`internal/api`, the `create_api_key` / `revoke_api_key` tools and the peer
credential in `internal/mcp`, the hoisted spawn and the seed in
`internal/manager` (`hoist.go`), the peer-mode select in the new-session form,
and the muted `H` flag in the sidebar (with the `C`/`S`/`H` flags standardised,
see [../tui/sessions.md](../tui/sessions.md)).

---

## Still ahead

1. **Confirm the seed skip-list against a real Claude Code version.** The seed
   reads the borrower's Claude directory and symlinks every entry into the
   per-lender `CLAUDE_CONFIG_DIR`, except a fixed set of session-state entries
   (`hoistSkip` in `internal/manager/hoist.go`): `projects`, `todos`,
   `history.jsonl`, `.credentials.json`, and the caches. The symlink approach
   mirrors whatever the borrower has, so a new kind of setup entry needs no code
   change. The remaining risk is a new state entry a later Claude Code version
   adds: it would be symlinked, and the lender's writes to it would reach the
   borrower's directory. Confirm the state entries against a running tool, and add
   any new ones to `hoistSkip`.
