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

1. **Confirm the seed layout against a real Claude Code version.** The seed
   copies the borrower's `commands`, `skills`, and `rules`, and symlinks
   `claude.json`, into the per-lender `CLAUDE_CONFIG_DIR`. The exact location of
   `claude.json` and of the seeded sub-directories, relative to
   `CLAUDE_CONFIG_DIR`, depends on the installed Claude Code version. The seed is
   defensive (a missing source is skipped), so a hoisted session starts either
   way, but the seed is not verified against a running tool. Confirm the paths,
   then fix `seedHoistDir` in `internal/manager/hoist.go` if they differ.
