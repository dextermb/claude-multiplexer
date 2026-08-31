# Session self-rename

**Status:** in progress. The manual rename is built and shipped. Its behaviour
now lives in [manager.md](../manager.md), [tui/sessions.md](../tui/sessions.md),
and [tui/keys.md](../tui/keys.md). Only the self-rename is still ahead, and its
channel is not decided.

A user can rename a session with the `R` key. The name stays the identity key,
and a new `Title` carries the display text. This plan holds the one part that is
not built: letting the Claude inside a session rename itself.

---

## What is already built

- `Session.SetTitle` sets a mutable title and emits a state event.
- `Snapshot.Title` and `Meta.Title` carry the title. `Meta.sameAs` includes it,
  so a title change is persisted.
- `Manager.SetTitle(name, title)` renames a live session (via the session) or a
  stored session (straight to its meta).
- The TUI shows `row.displayName()` (title or name) in the sidebar and the bar.
  The `R` key opens a one-field dialog.

The one gap: a live session renamed before its first turn keeps the title only
in memory, because `rememberSession` skips a session with zero turns. The title
persists on the first turn. This is acceptable, because a zero-turn session is
removed when it ends.

---

## Open question — the inbound channel

Claude has no way to send a rename to the multiplexer today. Only model and
permission-mode changes flow the other way, to Claude. Three options:

1. **An MCP tool** `mcp__mux__rename_session`. The multiplexer serves an
   in-process MCP server. It launches Claude with `--mcp-config` and the tool on
   `--allowedTools`. The tool handler resolves the calling session and calls
   `Manager.SetTitle`. This matches the roadmap, which plans `mcp__mux__approve`
   for permission prompts; see [multiplexier.md](multiplexier.md). The same
   server can carry both tools.
2. **A hook.** A Claude Code hook shells back to the multiplexer. It needs an IPC
   path (for example a Unix socket) from the spawned command into the running
   process, keyed by session.
3. **Defer.** Keep the manual rename only.

**Recommendation:** the MCP tool, because the roadmap already commits to an MCP
server. It should ride with, or after, the permission-prompt work rather than
alone.

---

## The build — self-rename (after the channel is decided)

If the MCP option is chosen:

- Add an in-process MCP server that exposes `rename_session(title)`.
- Launch Claude with `--mcp-config` and the tool allowed.
- The tool handler resolves the session and calls `Manager.SetTitle`.
- Share the server with the future `mcp__mux__approve` tool.

This section stays a sketch until the channel is decided.
