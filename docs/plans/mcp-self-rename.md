# MCP self-rename — let a session's Claude rename itself

**Status:** draft, not started. This is a follow-up to the manual rename, which
is shipped. The manual rename behaviour lives in [manager.md](../manager.md),
[tui/sessions.md](../tui/sessions.md), and [tui/keys.md](../tui/keys.md).

Another agent can pick this up. Read the "Open questions" first, and settle the
transport with the user before you build.

---

## Goal

Let the Claude inside a session set its own title. A session that starts on a
vague name (for example the directory) can name itself for the task it takes,
without a key press from the user.

The title machinery already exists. This plan only adds the inbound channel that
carries a title from the child to the multiplexer.

---

## What already exists (do not rebuild)

- `Session.SetTitle(title string)` — sets a mutable title and emits a state
  event, so the interface refreshes.
- `Snapshot.Title` and `Meta.Title` — carry the title. `Meta.sameAs` includes
  it, so a change is persisted by `rememberSession`.
- `Manager.SetTitle(name, title string) error` — renames a live session (through
  the session) or a stored session (straight to its meta). **This is the one
  call the MCP tool handler needs.**
- `session.Config` already has the fields to launch an MCP server per session:
  `AllowedTools`, `ExtraArgs`, and `Env`. `Args()` joins `AllowedTools` and
  appends `ExtraArgs`, so the manager can inject the flags at spawn.

---

## Decision

**Use an MCP tool, not a hook.** The roadmap already commits to an MCP server
inside the multiplexer for the permission-prompt tool `mcp__mux__approve`; see
[multiplexier.md](multiplexier.md). One server can carry both tools. A hook would
need its own IPC path and would not share that work.

The tool is `mcp__mux__rename_session`, with one argument `title` (a string). An
empty title clears the title, the same as the manual dialog.

---

## Open questions — settle these before you build

### 1. The transport, and how the tool finds its session

Each session is a separate Claude child. The tool handler must know **which**
session called it. Two shapes:

- **One in-process HTTP or SSE server (preferred).** The multiplexer serves one
  MCP endpoint on a local port. At spawn, it generates a per-session secret
  token, maps `token -> session name`, and writes a per-session `--mcp-config`
  that carries the token (in a header or the URL). The handler reads the token,
  finds the session, and calls `Manager.SetTitle`. One server, shared with the
  future approve tool.
- **A stdio subprocess per session.** Claude spawns `multiplexier mcp --session
  <name>` per session. That subprocess talks back to the main process over a
  Unix socket. It needs a new subcommand and a socket server, and one process
  per session.

**Recommendation:** the in-process HTTP or SSE server with a per-session token.
It is one server, and it is the shape the approve tool needs too.

### 2. Probe the installed Claude Code first

The roadmap already warns that the `control_request` schema is not modelled and
must be probed against the installed version. The same caution applies here.
Before you build, confirm against the pinned Claude Code version:

- the `--mcp-config` format it accepts (file path, inline JSON, http or sse
  entries, headers),
- how it names an MCP tool on `--allowedTools` (expected `mcp__mux__…`),
- that a tool call reaches the server, and that the child continues after the
  tool returns.

Record what the installed version does in [protocol.md](../protocol.md).

### 3. Does the tool need confirmation from the user?

A rename is low risk and reversible, so the first version can apply it at once
and show a status line. If a self-rename should ask the user first, that reuses
the same input-required state the approve tool needs, so decide the two together.

---

## Dataflow

```
Claude (child)                multiplexer
   |                              |
   | tool call rename_session     |
   |  title="Billing rewrite"     |
   |  + per-session token         |
   |----------------------------->|  MCP server resolves token -> name
   |                              |  Manager.SetTitle(name, title)
   |                              |    live: Session.SetTitle -> event -> bus -> TUI
   |                              |    (persist on next turn via rememberSession)
   |<-----------------------------|  tool result: ok
```

The write path after `Manager.SetTitle` is the one the manual rename already
uses. This plan adds only the left half.

---

## The build

1. **MCP server package** (for example `internal/mcp`). Serve one tool,
   `rename_session(title: string)`. Keep the transport behind a small interface,
   so the approve tool can join later.
2. **Per-session identity.** At `Manager.Spawn`:
   - generate a token, store `token -> name` in the manager (guard with `mu`),
   - write a per-session MCP config under `sessionDir(root, name)`,
   - set `cfg.ExtraArgs` with `--mcp-config <path>` and add
     `mcp__mux__rename_session` to `cfg.AllowedTools`.
   Remove the token when the session ends (in `pump`, next to the zero-turn
   cleanup).
3. **Handler.** Resolve the token to a name, then call
   `Manager.SetTitle(name, title)`. Return a short result. Reject an unknown
   token.
4. **Lifecycle.** Start the server when the manager starts, and stop it in
   `Manager.Shutdown`. One server for the whole run.
5. **Docs.** Update [protocol.md](../protocol.md) for the MCP server and the
   tool, [manager.md](../manager.md) for the token map and the launch flags, and
   remove the "still ahead" note about self-rename from
   [manager.md](../manager.md) once it lands.

---

## Verification

- Unit test the handler: a known token renames the right session, an unknown
  token is rejected.
- Unit test that `Spawn` writes the MCP config and adds the tool to the allowed
  list.
- Integration test: the fake Claude in `internal/testutil/fakeclaude` must call
  the tool, so it needs a mode that emits a `rename_session` tool call. **This is
  real work — budget for it.** Then assert the session title changes end to end.
- Manual check: start a session, have Claude call the tool, confirm the sidebar
  and the bar show the new title, and that it survives a restart.

---

## When this lands

Follow [plans.md](../../.claude/rules/plans.md): move the durable parts into
`docs/*.md`, repoint anything that named this plan, and delete this file once
nothing is ahead. The git history keeps it.
