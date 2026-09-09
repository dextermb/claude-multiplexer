# Hoisted sessions — what is still ahead

**Status:** shipped. The feature is built and tested on branch
`hoisted-sessions`. What it does is described in
[../peers/hoisted.md](../peers/hoisted.md). This file holds only the work that is
not built. Delete it when nothing is left.

The build landed: the credential type in `internal/config`, the environment
builder in `internal/session` (`env.go`), the lent-key metadata in
`internal/api`, the `create_api_key` / `revoke_api_key` tools and the peer
credential in `internal/mcp`, the hoisted spawn and the seed in
`internal/manager` (`hoist.go`), and the peer-mode select in the new-session
form.

---

## Still ahead

1. **Confirm the seed layout against a real Claude Code version.** The seed
   copies the borrower's `commands`, `skills`, and `rules`, and symlinks
   `claude.json`, into the per-lender `CLAUDE_CONFIG_DIR`. The exact location of
   `claude.json` and of the seeded sub-directories, relative to
   `CLAUDE_CONFIG_DIR`, depends on the installed Claude Code version. The seed is
   defensive (a missing source is skipped), so a hoisted session starts either
   way, but the seed is not verified against a running tool. Confirm the paths,
   then fix `seedHoistDir` in `internal/manager/hoist.go` if they differ.

2. **A sidebar badge for the lender.** A hoisted session sits in the local band,
   and its lender is on the session record that `list_sessions` returns
   (`mcp.Session.Lender`). The sidebar does not yet draw a visible badge for it.
   The sidebar marks remote sessions by grouping them under a peer header, and a
   hoisted session stays local by design, so a badge would be a new inline
   convention. Add one only after use shows it is wanted.
