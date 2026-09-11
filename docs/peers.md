# Peers

The multiplexer shares its Claude usage with other multiplexer hosts on the same
network, and reads theirs. It also starts a session on a peer and streams it into
the local output pane, like a local session. This page covers the usage poll, the
peer listener, the API routes, the remote-session flow, and the tools.

A remote session runs on a peer and streams into the host that started it. The
manager holds it in a `remotes` map, and a pump republishes the peer's stream to
the local bus under a local name, so the pane treats it as local. The TUI names
where a session runs: the new-session `host` field starts a session on a peer,
and the sidebar splits into bands (see docs/tui/sessions.md). A usage reserve
protects a share of this host's Claude usage for its own work (below).

A peer may instead lend its Claude credential. The borrower then runs the session
on its own machine as the peer — a hoisted session. See
[peers/hoisted.md](peers/hoisted.md).

A host may also share a session read-only. It mints a share link, and whoever
holds the link watches the session live but cannot drive it — a spectator
session. See [peers/spectate.md](peers/spectate.md).

## The pages

| Page | Read it for |
|---|---|
| [peers/connect.md](peers/connect.md) | Connect two hosts: the roles, the diagrams, and the step-by-step |
| [peers/hoisted.md](peers/hoisted.md) | Run a peer's session locally with the peer's credential: the opt-in, the tools, and the run |
| [peers/spectate.md](peers/spectate.md) | Share a session read-only, and watch a shared session: the link, the tools, and the run |

# The usage source

The usage figure is the Claude usage-limit stat, the same one Claude Code shows
in `/usage`. It does not come from the session stream: a session `Result` event
carries the cost and the token counts, but not the plan percent or the reset.

The multiplexer reads the stat from the `anthropic-ratelimit-unified-*` response
headers the Anthropic API returns for a request made with the account's
credential:

- `anthropic-ratelimit-unified-5h-status` / `-remaining` / `-reset` — the
  5-hour rolling window.
- The 7-day window carries the same three headers.

There is no official standalone usage endpoint yet, so the parse is defensive: a
header that is not there, or does not parse, reads as unknown, never as zero.
See `internal/usage`.

## The poll

A read of usage costs a little of the account's usage, because it is a real API
request. So the `usage.Poller` calls the fetch on an interval and caches the
result, and a read of usage never makes a network call. A failed fetch keeps the
last-known windows, so a transient failure does not lose the last good values.

The fetch is one injected function (`usage.Fetch`) — the manager wires it as
`Options.UsageFetch`. The exact endpoint and the credential the fetch uses are
the one integration seam still open, to settle against a real account. When no
fetch is wired, usage reads as unknown.

# The peer listener

A host that opts in binds a second listener, separate from the loopback API. The
loopback API stays on `127.0.0.1` and is unchanged. The peer listener binds
`0.0.0.0` on a fixed port (default `51900`), so a peer on the network reaches
this host at a stable address. It must be a separate listener on its own port,
because the loopback API binds `127.0.0.1` on a dynamic port and carries the
`/admin` surface, which never reaches the network. The peer listener serves only
the owner-scoped surface a peer reaches:

- `POST /token` — the client-credentials grant, the same one an external client
  uses.
- `GET /api/usage` — this host's usage, guarded by an access token.
- The `/api/sessions` REST — list, create, message, interrupt, stop, archive —
  scoped to the client's own sessions.
- `GET /api/sessions/{name}/stream` — the session's event stream as server-sent
  events: the replay of the current lines first, then the live events.
- The `/api/shares/{id}` surface — the read-only routes a spectator reaches with
  a share token: the label, the stream, and the transcript. See
  [peers/spectate.md](peers/spectate.md).

A peer reaches only the sessions it owns, because every session it creates is
owned by its client. A spectator reaches only the one session its share opens.
The peer listener starts only when `peers.enabled` is on. See `internal/mcp`
(`StartPeer`) and `config.Peers.ListenAddr`.

A session a peer creates through this listener is marked hosted, so the host that
runs it sorts it under the hosted section. The loopback API never marks a create
hosted. See `internal/mcp` (`handlePeerAPI`).

## The stream

The stream carries a serialized session event (`internal/wire`): the rendered
lines, the snapshot, the partial text, the todos, and the questions — everything
the output pane reads. The host that views the session decodes each event and
republishes it to its own bus under a local name, so the pane treats a remote
session as local. `render.Line` is width-independent, so the viewing host draws
the lines at its own width.

The stream sends the current lines first (the replay), then the live events. The
host must not send a line in both, or the viewer draws it twice. So the host
reads the replay lines and a watermark together, under one hold of the bus lock:
the watermark is the sequence of the last event the replay lines hold. The host
then drops the lines of a live event at or below the watermark, because the
replay holds them. The read is atomic, so no event falls in the gap between the
subscribe and the replay, and no line is lost or doubled. See
`internal/manager` (`stream.go`, `bus.go`).

# Remote sessions in the manager

`AttachRemote` starts a session on a peer, then streams it into this host under a
local name. It returns the local name. The manager holds the session in a
`remotes` map, and a pump reads the peer's stream and republishes each event to
the local bus. The first event of each connection carries the whole line buffer,
so the pump replaces the local buffer; a later event appends. See
`internal/manager` (`remote.go`).

The pump marks the first event of each connection with `Replace`, because its
lines are the whole buffer, not a delta. A viewer that reads the event rebuilds
its output from the buffer instead of appending. Without the mark, a reconnect
draws the whole transcript a second time, below the copy already on the pane.
See `internal/tui` (`events.go`).

The create carries a directory on the peer, or `temp_dir` for a fresh one the
host makes (`os.MkdirTemp`), so a client starts a session without knowing the
host's paths. The host removes a temporary directory when the session did no
work, or when the session is removed; a session that did work keeps it.

A streamed session routes its input and reads back through the peer: `Send`,
`Stop`, and `Interrupt` post to the peer; `Lines`, `Snapshot`, `Messages`, and
`Todos` read the local cache the pump fills (except `Messages`, which reads the
peer's transcript). `List` carries the streamed session with its peer as `Host`.

The pump reconnects when the stream drops, so a peer that restarts does not end
the session on this host. When the session closes on the peer, the pump detaches
the session and stops. The manager records each streamed session (the peer and
the remote name) in `remotes.json`, and re-attaches it on start, under the same
local name, so a restart of this host does not lose the session — the peer keeps
running it. A link whose peer is gone from the config is dropped.

## Sections

A session sorts into one of three sidebar sections from two fields on
`mcp.Session`:

- **streamed** — the session runs on a peer and streams in. `Host` names the
  peer; `Hosted` is false.
- **hosted** — this host runs the session on behalf of a peer, which created it
  through the peer listener. `Hosted` is true.
- **local** — everything else. Both fields are empty or false.

# The reserve

The reserve protects a share of a host's Claude usage for its own work. The host
sets a floor (`reserve.min_percent`) on a window (`reserve.window`, `5h` or
`7d`). A gate compares the polled percent remaining in that window against the
floor, and re-evaluates on every poll, so a change to the reserve takes effect on
the next poll. See `internal/manager` (`reserve.go`).

The gate trips when the percent remaining drops below the floor. While it is
tripped:

- The peer listener refuses a new session from a peer. `POST /api/sessions`
  answers `403` with a reason. No new hosted session starts.
- Every hosted session pauses after its current turn. The session finishes the
  turn it runs, then holds the queue: a queued prompt waits, and the session
  stays live. See `internal/session` (`SetPaused`).

The gate acts only on hosted sessions. A local session and a streamed session are
never paused by this host's reserve — a streamed session obeys the reserve of the
host that runs it.

The gate clears on its own. The window is rolling, so the percent climbs back as
it rolls. The next poll reads the higher percent, the gate clears, the hosted
sessions resume, and the peer listener accepts a new session again. An unknown
percent (a missing header) never trips the gate, and no reserve keeps hosting on
with no floor.

# Auth

A peer authenticates with the client-credentials grant. Each host provisions an
API client for each peer (`create_api_client`), and the peer stores that
`client_id` and `client_secret` in its config. The peer's `peer.Client` runs the
grant, caches the access token until it expires, and retries once with a fresh
token when the host answers `401`. See `internal/peer`.

# The config

The `peers` block in `config.json`. No block, or `enabled` off, keeps the peer
listener off.

```json
{
  "peers": {
    "enabled": true,
    "port": 51900,
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

- `enabled` — turns the peer listener on. It binds `0.0.0.0`, so a peer on the
  network reaches this host. Off (or absent) keeps the surface loopback-only.
- `port` — the port the peer listener binds. Omit for the default `51900`, which
  sits just past the loopback API range (`51890`–`51899`).
- `reserve.window` — the window a usage reserve guards: `5h` or `7d`.
- `reserve.min_percent` — the percent-remaining floor the gate trips below (the
  reserve). Omit `reserve` for no floor.
- `hosts[].name` — the label for the peer.
- `hosts[].url` — the base URL of the peer's peer listener.
- `hosts[].client_id` / `client_secret` — the credentials the peer provisioned
  for this host, used in the grant against `hosts[].url`.

# The tools

Read usage:

- `get_usage` — this host's usage from the poll cache. No network call.
- `peer_usage` — this host's usage, and a report for every configured peer. A
  peer that is off or unreachable reports `reachable: false` with the error, and
  never blocks the others.

Read the dial address (any session, because it holds no secret):

- `get_peer_url` — the url a peer on the network dials to reach this host, with
  the host and port inside it. The host is a routable LAN address, not the
  `0.0.0.0` the listener binds, so a peer uses the url as returned. It reports
  `enabled: false` with an empty url when the peer listener is off. Give this url
  to another host, which sets it as the peer `url` in `add_peer`.

Manage the config (control tools, next to the API-client tools, because a peer
entry holds a credential and turning peering on changes the network exposure):

- `list_peers` — whether peering is on and its port, the reserve, and each peer
  host (the name, the url, the client id, and whether it holds a lent
  credential). No secret is shown.
- `enable_peering` / `disable_peering` — turn the peer listener on (with an
  optional port) or off.
- `add_peer` / `remove_peer` — add a peer host (name, url, client id, secret, and
  an optional lent credential to hoist), or remove one by name.
- `update_peer` — change a peer host found by name, for example when its client
  secret is regenerated or its url changes. Give only the fields to change.
- `set_reserve` / `unset_reserve` — set the window and the floor, or clear it.

Share a session read-only (control tools):

- `share_session` — mint a read-only share link for one session. See
  [peers/spectate.md](peers/spectate.md).
- `list_shares` / `revoke_share` — list the active shares, or end one by id.
- `watch_share` — on the spectator, attach a read-only session from a link.

To lend a Claude credential, so a peer runs its session locally as this host, use
`create_api_key` and `revoke_api_key`, and give the peer entry a credential. See
[peers/hoisted.md](peers/hoisted.md).

A change to peering or a peer host takes effect on the next restart, the same as
a hand edit of the file. Each tool writes the config and returns the
path it wrote.

# Turn on peering

For the full walkthrough — the two roles, the diagrams, and the steps on each
host — see [peers/connect.md](peers/connect.md). In short:

1. On the host that runs the sessions: run `enable_peering`, then
   `create_api_client` for each peer, and restart.
2. On the host that starts the sessions: run `add_peer` with the address and
   those credentials, and restart.
