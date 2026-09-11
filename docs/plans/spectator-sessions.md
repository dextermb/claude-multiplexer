# Plan: spectator sessions — read-only shares of a session

Status: **in progress**. The decisions below are settled with the user. The build
runs in the `spectate` worktree, one commit per phase. Phase 1 (the share store
and the host `/api/shares/{id}` surface) is in progress.

# Goal

Let a peer watch a session on another host, read-only. The host mints a share for
one session. The peer pastes the share and reads the session in its own output
pane, live, but cannot type, stop, or interrupt it. The share is a bearer
credential: whoever holds it can watch until it expires or the host revokes it.

# The key insight

The transport is already built. The peer listener serves
`GET /api/sessions/{name}/stream` as server-sent events of `wire.Event` (the
rendered lines, the snapshot, the partial text, the todos, the questions). The
manager remote pump reads that stream and republishes each event to the local
bus, so a remote session draws like a local one (see docs/peers.md, `remote.go`).

So a spectator session is a **streamed session that the viewer did not create**,
and that carries no input path:

- **Read-only is the absence of the write routes.** `message`, `stop`,
  `interrupt`, and `archive` are separate POST routes. A share surface that
  serves only the stream (and the transcript) cannot drive the session. The
  boundary is server-side, not a client courtesy.
- **The real gap is authorization and discovery.** Today `ownedSessions.guard`
  rejects any session a client does not own (`owner.go`). A spectator wants a
  session it did not create. The share is the capability that opens exactly one
  session's read routes, and the link is how the peer receives it.

Nothing below the manager bus changes. The pump, the buffer, and the pane already
treat a streamed session as local.

# Decisions

From the user:

1. **Server-stored, revocable token.** The host stores each share as a file. A
   share is a bearer token: no prior peer provisioning, and the host revokes it
   at any time. Rejected: a stateless signed capability (not revocable without
   key rotation), and a viewers ACL bound to a provisioned API client (heavier
   setup, no bearer link).
2. **One session per share.** Each share names one session. Rejected for now: a
   host-wide read-only mirror (a bigger grant; can follow later as a second
   scope).
3. **One bundled link.** The share is a single pasteable string that carries the
   host address and the token together. Rejected: separate token and address
   fields.
4. **The link is one base64url blob**, `cmux://spectate/<blob>`, not a legible
   query string. It reads as one opaque credential. Rejected: the
   `?u=…&t=…` query form (legible, but looks fiddly to hand to a person).
5. **A share expires 24 hours after it is minted, by default.** A forgotten link
   does not stay open for ever. `share_session` takes an `expires_in` to shorten
   or lengthen it, and a value that turns expiry off for a long-lived share.

Derived, to confirm in the plan review:

6. **The share is its own bearer credential — no OAuth grant.** A spectator does
   not run the client-credentials grant and holds no `client_id`. The share
   token authenticates the read routes directly. So a laptop watches without
   exposing anything: the viewer binds no listener and provisions no client.
7. **The token is a two-part credential, `<share_id>.<secret>`**, like a client
   id and secret. The host stores only a hash of the secret, and reuses
   `internal/api/hash.go`. The id indexes the store; the secret is checked
   against the stored hash.
8. **The share references the session by its host-local name, and the peer never
   learns that name.** The share surface maps the share id to the session inside
   the host, so the viewer streams by share id. No session name crosses the
   boundary, the same property owner scoping already holds.

# The share model

## The store

A share is a file under the API store, next to the clients (see docs/mcp/api.md):

```
~/.claude-multiplexer/
  api/
    clients/<client_id>.json      unchanged
    shares/<share_id>.json        one share, the secret hashed
```

The record:

```json
{
  "id": "shr_A1B2C3",
  "session": "brave-otter",
  "secret_hash": "…",
  "scope": "view",
  "created": "2026-09-11T10:00:00Z",
  "expires": "2026-09-12T10:00:00Z",
  "revoked": false
}
```

- `session` — the host-local session name the share opens.
- `scope` — `view` only, for now. The field leaves room for a wider scope later.
- `expires` — the mint time plus 24 hours by default. `share_session` takes an
  `expires_in` to change it, or to turn expiry off for a long-lived share.
- `revoked` — a revoke sets this true (or deletes the file). Either ends the
  share at once, before the expiry.

The manager scans `api/shares/` at start, holds the records in memory, and writes
one file per change — the same shape as the client store.

## The token and the link

The token is `<share_id>.<secret>`, the id and a random secret joined by a dot.
The link bundles the host address and the token in one base64url blob:

```
cmux://spectate/<blob>

blob = base64url({
  "u": "http://192.168.1.20:51900",   the peer-listener base URL (get_peer_url)
  "t": "shr_A1B2C3.<secret>"          the share token
})
```

The blob reads as one opaque credential, so a person copies it as a single
string. The host shows the link once, at mint, the same way a client secret
shows once. `watch_share` on the viewer decodes the blob back to the URL and the
token.

# The share surface (host)

The peer listener gains a share-scoped surface, parallel to the owner-scoped
`/api/sessions` surface. Every route reads the share token from the
`Authorization: Bearer` header, loads `shares/<id>.json`, checks the secret hash,
and checks the share is not expired and not revoked. A bad token, an expired
share, or a revoked share gets `401`.

| Method and route | Action |
|---|---|
| `GET /api/shares/{id}` | the session label (title, host name, model) so the viewer draws a header — no session name |
| `GET /api/shares/{id}/stream` | the session's events (SSE), read-only — the same replay-then-tail the owner stream serves |
| `GET /api/shares/{id}/messages` | the transcript, read-only |

There is no `message`, `stop`, `interrupt`, or `archive` route under
`/api/shares/`. So the surface cannot drive the session, whatever the client
sends.

The stream reuses `streamSession` from the owner path. The only change is the
guard: a share guard in place of the owner guard, mapping the share id to the
session name.

# Read-only enforcement

Two lines hold read-only, and the server-side line is the one that matters:

1. **Server-side (the boundary).** The share surface serves only GET routes.
   Even a viewer that ignores the read-only flag has no route to post a prompt or
   a stop. This is the line that cannot be bypassed.
2. **Viewer-side (the courtesy).** A spectated session is marked read-only, so
   the TUI disables the input box and the stop and interrupt keys, and shows a
   read-only badge. This stops the viewer from trying, and explains why.

# The viewer side

A spectated session is a `remoteEntry` with two differences from a streamed one:

- It is created by attaching a share, not by `CreateSession`. A new
  `AttachSpectator(link)` parses the link, dials `GET /api/shares/{id}/stream`
  with the token, and registers a `remoteEntry` whose pump streams by share id.
- It is read-only. A `readOnly` flag on `remoteEntry` makes `Send`, `Stop`, and
  `Interrupt` refuse (the routing in `messages.go`/`lifecycle.go` checks the
  flag), and the TUI reads the flag to disable input.

`peer.Client` gains a share mode: it dials the `/api/shares/{id}` routes with the
bearer token, instead of the `/api/sessions` routes after a grant.

**The pump must end on a terminal auth failure.** Today `remotePump` retries
every stream error, so a peer restart does not drop the session. A revoked or
expired share returns `401`. The pump must tell a transient drop (retry) from a
terminal `401`/`403` (detach and mark the session ended). So the spectated
session leaves the sidebar when the host revokes the share, instead of retrying
for ever.

The viewer records the share link in `remotes.json` (next to the streamed
sessions), so a restart re-attaches a spectated session — until the share is
revoked, when the re-attach gets `401` and drops the link.

# The sidebar (viewer)

A spectated session sorts under `remote sessions` -> `streamed`, with a read-only
glyph on the row, so the viewer sees it is a watch, not a drive. (Alternative
considered: a new `spectating` band under `remote sessions`. Rejected for now: a
glyph is lighter than a new band, and the section plumbing stays as it is. Revisit
if a viewer watches many shares at once.)

The host side shows nothing new in v1: a shared session is still a normal local
or hosted session on the host. A "someone is watching" indicator is an open
question, below.

# The tools

Host (mint and manage — control tools, next to the API-client and peer tools,
because a share hands out a capability):

- `share_session` — mint a read-only share for one session. Takes the session
  name and an optional `expires_in`. Returns the `share_id` and the link once.
  Requires the peer listener on (`peers.enabled`), because the link points at it.
- `list_shares` — the active shares: id, session, created, expiry, revoked. No
  secret.
- `revoke_share` — end a share by id. The host drops any live stream on that
  share at once.

Viewer (attach — a control tool, next to `add_peer`, because it dials a network
host):

- `watch_share` — take a link, attach a read-only streamed session, and return
  the local name. The viewer needs no peer listener and no client for this.

Removing a spectated session (the TUI `x`) detaches the stream locally. It never
stops the session on the host, because the share is read-only.

# Data flows

Mint and watch a share:

```
HOST (runs the session)                          VIEWER (watches)
  share_session "brave-otter"
    -> writes api/shares/shr_A1B2C3.json (expires in 24h)
    -> returns cmux://spectate/<blob>
                        the user copies the link to the viewer
                                                 watch_share "<link>"
  GET /api/shares/shr_A1B2C3         <---------- dial with Bearer <token>
    validate token, not expired, not revoked
    map shr_A1B2C3 -> "brave-otter"
  GET /api/shares/shr_A1B2C3/stream  <---------- SSE
    replay the lines, then tail the bus  --------> republish to the local bus
                                                 read-only streamed session
```

Revoke:

```
HOST                                             VIEWER
  revoke_share shr_A1B2C3
    -> revoked = true; close the live stream
                                       ---------> pump reads 401 (terminal)
                                                 detach; the session leaves the sidebar
                                                 drop the link from remotes.json
```

# Workflows

Share a session (host):

1. Turn peering on, if it is off (`enable_peering`, restart). The share link
   points at the peer listener.
2. Run `share_session` with the session name. Copy the link it returns once.
3. Give the link to the person who will watch.
4. To end the watch, run `revoke_share` with the id from `list_shares`.

Watch a share (viewer):

1. Run `watch_share` with the link.
2. Read the session under `remote sessions` -> `streamed`, with a read-only
   badge. The input box is off, and stop and interrupt do nothing.
3. Press `x` to close the watch. This detaches the stream; it does not touch the
   session on the host.

# The build

The build lands in phases, each green on its own, in one worktree per phase (see
.claude/rules/worktrees.md).

Phase 1 — the share store and the host surface:

- `internal/api` (or `internal/mcp`): a `Share` record and a share store that
  scans `api/shares/`, mints a share (`<id>.<secret>`, hash the secret), looks
  one up and validates it (hash, expiry, revoked), lists, and revokes. Reuse
  `internal/api/hash.go`.
- `internal/mcp`: the share-scoped routes on the peer listener —
  `GET /api/shares/{id}`, `/stream`, `/messages` — with a share guard that maps
  the id to the session name and reuses `streamSession`.
- `internal/manager`: a share guard view like `ownedSessions`, but keyed by the
  share's session, read-only (only the read methods).

Phase 2 — the mint and manage tools (host):

- `internal/mcp`: `share_session`, `list_shares`, `revoke_share` in
  `tools_shares.go`; their methods on the `Sessions` interface; the tool
  constants in `ControlTools`.
- `internal/manager`: the methods write the store and return the link.

Phase 3 — the viewer attach and read-only routing:

- `internal/peer` `Client`: a share mode that dials `/api/shares/{id}` with the
  bearer token; `Stream(ctx)` by share id.
- `internal/manager`: `AttachSpectator(link)`; a `readOnly` flag on
  `remoteEntry`; `Send`/`Stop`/`Interrupt` refuse when read-only; the pump ends
  on a terminal `401`/`403`; record and re-attach the link in `remotes.json`.
- `internal/mcp`: the `watch_share` tool.

Phase 4 — the TUI:

- The read-only badge on a spectated row.
- The input box, and the stop and interrupt keys, off for a read-only session.
- `x` detaches a spectated session locally.

Docs (with the code that lands):

- `docs/peers.md` — a spectator section: the share model, the surface, and the
  read-only boundary.
- New `docs/peers/spectate.md` — the mint-and-watch walkthrough (linked from the
  peers index), parallel to `peers/connect.md`.
- `docs/mcp/api.md` — the `/api/shares/` surface.
- `docs/mcp/tools.md` — the four new tools.

# Verification

- The share store: mint round-trips; a wrong secret, an expired share, and a
  revoked share each fail the guard; `list_shares` holds no secret.
- The host surface: `GET /api/shares/{id}/stream` replays then tails; a bad,
  expired, or revoked token gets `401`; there is no write route under
  `/api/shares/`.
- The share guard maps the id to the session and leaks no session name.
- The viewer: `AttachSpectator` streams into the bus; `Send`/`Stop`/`Interrupt`
  refuse on a read-only session; the pump detaches on `401` and retries on a
  transient drop; a re-attach after a revoke drops the link.
- The TUI: a read-only session shows the badge, the input is off, and `x`
  detaches without a call to the host stop route.
- `just check` green before each collapse.

# Open questions

1. **A "watched" indicator on the host.** Should the host show that a session is
   shared, and how many are watching? v1 shows nothing. Answer from use.
2. **A host-wide read-only mirror.** The second scope the user set aside. If it
   is wanted, `scope: "host"` opens the list and the streams of every session,
   read-only. Not in this plan.

# A note on privacy

Warning: a share hands the whole rendered session to whoever holds the link.
This includes every tool call, and any file content the session prints. Share a
link only with a person who may read all of that session, and revoke it when the
watch is over.
