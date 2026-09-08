# Plan: peers — usage sharing and remote sessions

Status: **in progress**. Phases 1, 2, and 3 are built and green in the
`peer-usage` worktree. Phase 1 is the usage poll, the peer listener, the
`/api/usage` route, and the `get_usage` / `peer_usage` / peer-config tools.
Phase 2 is the remote-session transport: the `internal/wire` event, the
owner-scoped session REST and the SSE stream on the peer listener, and the
`peer.Client` methods (`CreateSession`, `Stream`, `Send`, `Stop`). Phase 3 is
remote sessions in the manager: the `remotes` map, `AttachRemote`, the pump that
republishes a peer's stream to the local bus, reconnect and re-attach on start,
the routing of `Send`/`Stop`/`Interrupt`/`Lines`/`Snapshot`/`Messages`/`List`
through the remotes map, and the `Host`/`Hosted` section data on `mcp.Session`.
The durable Phase 3 content now lives in docs/peers.md. Phase 4 is the TUI: the
new-session `host` field (local or a peer) and the sidebar section bands (local
sessions, and a remote-sessions parent over hosted and streamed), which show only
with peering on; its durable content lives in docs/tui/sessions.md and
docs/peers.md. Still ahead: Phase 1b (the reserve gate, which the hosted-session
concept from Phase 3 unblocks). The one integration seam left open is
`Options.UsageFetch`: the exact usage endpoint and credential, to settle against
a real account. See docs/peers.md.

# Goal

Let multiplexer hosts on the same network work together. Three capabilities:

1. **Usage sharing.** A host serves its Claude usage. A tool reads the usage of
   every configured peer.
2. **Remote sessions.** A user starts a session on a peer's host, and that
   session streams into the local output pane and takes input, like a normal
   (non-control) session.
3. **A usage reserve.** A host keeps a share of its own Claude usage. When the
   remaining usage drops below the reserve, the host pauses its hosted sessions
   after the current turn, and refuses a new session from a peer.

The sidebar splits into sections that name where a session runs (below).

# Decisions (from the user)

1. **Discovery: a static peer list in the config.** No auto-discovery, no new
   dependency. The user lists each peer with its address and the credentials to
   reach it.
2. **Trust: reuse the client-credentials grant.** A peer authenticates with the
   same OAuth flow an external client uses. A session a host starts on a peer is
   owned by that host's client id, so owner scoping already limits it to its own
   remote sessions.
3. **Binding: a separate, opt-in peer listener.** The loopback API stays
   unchanged. A second listener binds a network interface and serves the
   owner-scoped surface: `POST /token`, the `/api/sessions` REST, the stream,
   and `GET /api/usage`. The `/admin` surface never reaches the network. The
   peer listener starts only when the config names a listen address.

# The key insight

`render.Line` is width-independent and serializable, and `manager.Event`
already carries all the output pane consumes (lines, snapshot, partial, todos,
questions, closed). So:

- The **wire format for a stream is a serialized `manager.Event`**.
- A **remote session splices in at the manager bus**: a goroutine reads
  `manager.Event`s from a peer over the network and republishes them to the
  local bus under a local name. The TUI, keyed by session name, treats it as
  local.

Nothing below the bus needs to know a session runs on another host.

# Resolved

- **Reconnect after a restart: reconnect.** Host A records the peer and remote
  name, and re-attaches the stream on start. A session survives an A restart,
  because B keeps running it.
- **Stream transport: server-sent events (SSE).** One-way server-to-client over
  `net/http`. Input goes back over the existing REST `message`/`stop` routes. No
  new dependency.

# The usage source

The usage figure is the Claude usage-limit stat, the same one Claude Code shows
in `/usage`. It does not come from the session stream. A session `Result` event
carries `total_cost_usd` and token counts, but not the plan percentage or the
reset time. See docs/peers.md.

The multiplexer reads the stat from the `anthropic-ratelimit-unified-*` response
headers the Anthropic API returns for a request made with the account's
credential:

- `anthropic-ratelimit-unified-5h-status` / `-remaining` / `-reset` — the
  5-hour rolling window.
- The 7-day (weekly) window carries the same three headers.

The multiplexer polls the API on an interval with a small request, reads the
headers, and caches the result. So a read of usage does not make a network call,
and the poll cost stays low.

For a subscription account, the `remaining` header is the percent left. For a
spend-capped account, the usage is the cost so far against a configured cap.

There is no official standalone usage endpoint yet, so the header parse is
defensive: a missing or renamed header reads as "unknown", never as zero.

# Config shape

A new `peers` block in `config.json`. No block, or an empty `listen`, keeps the
peer listener off.

```json
{
  "peers": {
    "listen": "0.0.0.0:51900",
    "reserve": { "window": "5h", "min_percent": 20 },
    "hosts": [
      {
        "name": "workstation",
        "url": "http://192.168.1.20:51900",
        "client_id": "cid_...",
        "client_secret": "secret_..."
      }
    ]
  }
}
```

- `listen` — the address the peer listener binds. Empty keeps it off.
- `reserve.window` — which window the reserve guards: `5h` or `7d`.
- `reserve.min_percent` — the floor. The gate trips when the percent remaining
  in the window drops below this value. Omit `reserve` to keep hosting on with
  no floor.
- `hosts[].name` — the label for the peer in the sidebar and tool results.
- `hosts[].url` — the base URL of the peer's peer listener.
- `hosts[].client_id` / `client_secret` — the credentials the peer provisioned
  for this host, used in the grant against `hosts[].url`.

For a spend-capped account, `reserve` names a cost cap and a floor in dollars
instead of a window and a percent. The build settles that shape from the poll
result, once the header set for a spend-capped account is confirmed.

# MCP tools for the peer config

The peer config gets MCP tools, so a session manages it without a hand edit of
`config.json`. This follows the pattern the block-cap, editor, and API-client
tools already use: a tool writes the config and returns the path it wrote.

The tools are control tools, next to the API-client tools, because a peer entry
holds a credential and the listen address changes the network exposure.

- `list_peers` — the listen address, the reserve, and each peer host (the name,
  the url, and the client id — never the secret).
- `set_peer_listen` / `unset_peer_listen` — set or clear the listen address.
- `add_peer` / `remove_peer` — add a peer host (name, url, client id, secret) or
  remove one by name.
- `set_reserve` / `unset_reserve` — set the window and the floor, or clear the
  reserve.

A change to the listen address or a peer host takes effect on the next restart,
the same as a hand edit. A change to the reserve takes effect on the next poll,
because the gate reads the reserve each time it runs.

# Sidebar sections

The sidebar gains a section layer above the current groups. A section is drawn
with a horizontal-border header. The sections show only when the user opts in to
peering. Without peering, the sidebar is unchanged (no section headers).

Three sections, in order:

1. **local sessions** — sessions this host runs for itself. Today's sidebar.
2. **remote sessions** — a parent header, shown only with peering on. Two
   sub-sections sit under it:
   - **hosted** — sessions this host runs on behalf of a peer. A peer started
     them through the peer listener.
   - **streamed** — sessions that run on a peer, streamed into this host. The
     `remotes` entries.

The existing directory and creator groups keep working inside each section.

How a session sorts into a section:

- **streamed** — the session is a `remotes` entry (it runs on a peer).
- **hosted** — the session was created through the peer listener. A flag set at
  create time marks it, so a session a loopback API client created stays local.
- **local** — everything else.

# The reserve

The reserve protects a share of a host's Claude usage for its own work. The host
sets a floor (`reserve.min_percent`) on a window (`reserve.window`). A gate
compares the polled percent remaining against the floor.

The gate trips when `remaining < min_percent`. While the gate is tripped:

- The peer listener refuses a new session from a peer. `POST /api/sessions`
  answers `403` with a reason. No new hosted session starts.
- Every hosted session pauses after its current turn. The session finishes the
  turn it runs, and then holds the queue. A queued prompt waits, it does not
  run. The session stays live, it does not stop.

The gate clears on its own. The 5-hour window is rolling, so the percent climbs
back as the window rolls. The next poll reads the higher percent, the gate
clears, the hosted sessions resume, and the peer listener accepts a new session
again.

The gate acts only on hosted sessions. A local session and a streamed session
are never paused by this host's reserve. A streamed session obeys the reserve of
the host that runs it, not the host that views it.

The pause is a per-session flag the session `writeLoop` reads. When the flag is
on, the loop finishes the running turn and then does not take the next queued
prompt, even at idle. The manager sets the flag on every hosted session when the
gate trips, and clears it when the gate clears.

# Data flows

Usage (no network on the read side of `get_usage`):

```
poll Anthropic API (account credential)  ──read unified headers──▶  cached Usage{5h, 7d, ...}
get_usage  ──▶  cached Usage (no network)
peer B  ──POST /token, GET /api/usage──▶  host A peer listener  ──▶  Usage of A
```

Reserve gate (host B hosts for host A):

```
poll ──▶ remaining%  ──▶  gate = remaining% < reserve.min_percent
gate trips ──▶ B sets pause on every hosted session; B refuses POST /api/sessions with 403
window rolls ──▶ next poll ──▶ remaining% rises ──▶ gate clears ──▶ B clears pause; B accepts create
```

Start and stream a remote session:

```
host A TUI (new session, host = "workstation")
  A ──POST /token──────────────────────────▶ B  access token (cached until expiry)
  A ──POST /api/sessions {dir,model,...}────▶ B  starts a session owned by A's client, returns name
  A ──GET  /api/sessions/{name}/stream (SSE)▶ B  initial event (buffered lines) then live manager.Events
  for each event: A republishes to its local bus under the local name
  A ──POST /api/sessions/{name}/message─────▶ B  when the user submits a prompt
  A ──POST /api/sessions/{name}/stop────────▶ B  when the user stops the session
```

The peer session is a normal, non-control session on B. Owner scoping means A
reaches only the sessions A started on B.

# Workflows

Turn on peering on a host:

1. Add `peers.listen` to the config, e.g. `0.0.0.0:51900`.
2. For each host that will reach this host, run `create_api_client`, and give
   the `client_id` and `client_secret` to that host's `peers.hosts` block.
3. Restart. The peer listener binds the address.

Start a remote session (host A):

1. Open the new-session form. The `host` field lists `local` and each peer.
2. Choose a peer, and a directory that exists on that peer.
3. Submit. A starts the session on the peer and attaches its stream.
4. The session appears under `remote sessions` → `streamed`, marked with the
   peer name, and streams into the output pane. The user types prompts and stops
   it, as normal. On the peer, the same session appears under `remote sessions`
   → `hosted`.

A peer that is off or unreachable shows an error, and never blocks the TUI.

Keep a usage reserve (host B):

1. Add `peers.reserve`, e.g. `{"window": "5h", "min_percent": 20}`.
2. B polls its Claude usage. While the 5-hour window stays at or above 20%, B
   hosts sessions for its peers as normal.
3. The window drops below 20%. B pauses every hosted session after its current
   turn, and refuses a new session from a peer with a `403`.
4. The 5-hour window rolls and the percent climbs back above 20%. B resumes the
   hosted sessions and accepts a new session again.

# The build

The build is large. It lands in phases, each green on its own.

Phase 1 — usage polling and sharing:

- `internal/config`: add the `Peers` struct (listen, reserve, hosts) and field.
- `internal/usage` (new): poll the Anthropic API with the account credential,
  parse the `anthropic-ratelimit-unified-*` headers into a `Usage` (5h and 7d
  windows: status, percent remaining, reset), cache it, and re-poll on an
  interval. Parse defensively — a missing header reads as "unknown".
- `internal/peer` (new): `Peer` and `Client` (grant, token cache, `Usage(ctx)`).
- `internal/mcp`: `Usage`/`PeerReport` mirror types; `Usage()` and
  `PeerUsage(ctx)` on the `Sessions` interface; `StartPeer(listen)`;
  `handleUsage`; a `usage` case in `handleAPI`; tools `get_usage` and
  `peer_usage` in `tools_usage.go`; the `usage` route in `get_api_docs`.
- `internal/mcp`: the peer-config tools in `tools_peers.go` — `list_peers`,
  `set_peer_listen`, `unset_peer_listen`, `add_peer`, `remove_peer`,
  `set_reserve`, `unset_reserve`; their methods on the `Sessions` interface;
  the tool constants added to `ControlTools`.
- `internal/manager`: `Usage()` reads the cached poll result; `PeerUsage(ctx)`
  fan-out; the peer-config methods write `config.json` and return the path;
  `StartMCP` starts the poll and the peer listener when configured.

Phase 1b — the reserve gate:

- `internal/session`: a pause flag and `SetPaused(bool)`; the `writeLoop`
  finishes the running turn and then holds the queue while paused.
- `internal/manager`: a gate that reads the cached usage against
  `reserve.min_percent`; on a change, set or clear the pause on every hosted
  session. Expose the gate state so the peer listener reads it.
- `internal/mcp`: `restCreate` on the peer listener answers `403` with a reason
  when the gate is tripped.

Phase 2 — remote-session transport:

- `internal/peer` `Client`: `CreateSession(ctx, spec)`, `Stream(ctx, name)`
  returning a channel of decoded `manager.Event`, `Send`, `Stop`.
- `internal/mcp`: extend `restCreate` to accept `model`, `permission_mode`,
  `effort`; add `GET /api/sessions/{name}/stream` (SSE) that replays the line
  buffer then tails the bus for that session, owner-scoped; mount the sessions
  REST and the stream on the peer listener.
- Serialize `manager.Event` for the wire (a wire struct with `Snapshot.Err` as
  a string, so the error marshals).

Phase 3 — remote sessions in the manager:

- `internal/manager`: `remotes map[string]*remoteEntry`; `AttachRemote(peer,
  spec)`; a remote pump that reads the peer stream and republishes to the bus;
  route `Send`/`Stop`/`Interrupt`/`Lines`/`Snapshot`/`Messages`/`List` through
  the remotes map. Record the peer and remote name, and re-attach on start.
- Section data: add `Host` to `mcp.Session` (the peer name for a streamed
  session, empty for a local one) and a `Hosted` flag. Mark a session `Hosted`
  when the peer listener creates it (a `Hosted` field on `Spec` and `Meta`, set
  by the peer listener's create path, not by the loopback API).

Phase 4 — TUI:

- The new-session form gets a `host` field (local or a peer).
- The sidebar gains the section layer: a `section` above `group`, and a
  horizontal-border header for `local sessions` and `remote sessions`, with
  `hosted` and `streamed` under the second. The sections show only with peering
  on. A row sorts to a section from its `Host`/`Hosted` fields.
- Nothing else changes, because the output pane and the input path are keyed by
  session name and fed by the bus.

Docs (with the code that lands):

- New `docs/peers.md` — the peer model, the config, the auth, the routes, the
  remote-session flow, the usage poll (the unified headers), and the reserve.
- `docs/mcp/api.md` — the peer listener, the `usage` route, the stream route.
- `docs/mcp/tools.md` — the `get_usage`, `peer_usage`, and peer-config tools.
- `docs/config.md` — the `peers` block.

Verification:

- Unit tests: the header parse (a full header set, a missing header → unknown, a
  reset in the past); `/api/usage` with and without a token; the reserve gate
  (trips below the floor, clears above it); a hosted session pauses after its
  turn and resumes; `POST /api/sessions` answers `403` while the gate is
  tripped; `peer.Client` against a test server (reachable, refused, down); the
  SSE stream replay-then-tail; the manager remote pump republishing to the bus;
  the owner-scoped guard on a remote session; the sidebar section split (a
  hosted and a streamed session land in the right sections, and the sections
  hide when peering is off); the peer-config tools (add and remove a peer, set
  and unset the listen address and the reserve, and `list_peers` never returns a
  secret).
- `just check` green before each collapse.

Open, to settle in the build:

- The exact header set for a spend-capped account, and the `reserve` config
  shape for a cost cap. Confirm from a real poll before the gate reads it.
- The poll interval, and which session credential the poll uses.

# Review findings (Phase 1)

A review on 2026-09-08 checked the `peer-usage` worktree against this plan.
Phase 1 is complete, matches the plan, and is green (`go build ./...` and the
package tests pass). Phases 1b-4 are correctly not started. The notes below are
for the agent that lands the next phase or collapses the worktree.

Built and matches the plan:

- `config`: `Peers`, `Reserve`, `PeerHost`, `Window5h`/`Window7d`, `ValidWindow`.
- `internal/usage`: `Parse` (defensive header read), `Poller` (interval cache,
  keeps last-known values on a failed fetch).
- `internal/peer`: `Client` with the client-credentials grant, a cached token
  (30s slack), a 401 retry, and `Usage(ctx)`.
- `internal/mcp`: mirror types, the `Sessions` methods, `StartPeer`,
  `handleUsage`, the `get_usage` / `peer_usage` tools, the seven peer-config
  tools in `ControlTools`, and the `/api/usage` doc entry.
- `internal/manager`: `Usage()`, the `PeerUsage` fan-out, the config writers,
  and `StartMCP` starts the poll and the peer listener.
- Tests cover the Phase 1 verification list (header parse, missing header,
  unix reset, failed-fetch retention, `/api/usage` needs a token, grant and 401
  and rejected-grant, config round-trip, `list_peers` holds no secret, an
  unreachable peer, a bad window).
- `docs/peers.md` covers the usage source, the poll, the listener, the route,
  and the tools.

Deviations from this plan (both sound, the plan text is now stale):

1. `/api/usage` is a dedicated route, not a `case` in `handleAPI`. The code
   mounts `/api/usage` before the `/api/` catch-all in `api_http.go`. This is
   cleaner. `docs/peers.md` already describes the route correctly.
2. `Options.UsageFetch` is nil, so the poll is off and usage reads as unknown.
   This is the one deliberate open seam (the endpoint and credential).

Small cleanups for the next change:

- `config.go`: the `Hosts` field comment reads "keyed by no map so the order is
  stable". Reword to "a list, not a map, so the order in the file is stable".
- When Phase 1 collapses, this plan's status line still names the `handleAPI`
  `usage` case that did not ship. Repoint it, or drop it, per plans.md.
