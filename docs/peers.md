# Peers

The multiplexer shares its Claude usage with other multiplexer hosts on the same
network, and reads theirs. It also serves a session's event stream to a peer, so
a host can start a session on a peer and drive it. This page covers the usage
poll, the peer listener, the API routes, and the tools.

The transport a remote session rides on is built: the peer listener serves the
owner-scoped session REST and an event stream, and the `peer.Client` drives
them. What is still ahead is the local end — a remote session that appears in the
sidebar and the output pane of the host that started it (the manager `remotes`
map and the TUI sections).

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
loopback API stays on `127.0.0.1` and is unchanged. The peer listener binds a
network interface, and serves only the owner-scoped surface a peer reaches:

- `POST /token` — the client-credentials grant, the same one an external client
  uses.
- `GET /api/usage` — this host's usage, guarded by an access token.
- The `/api/sessions` REST — list, create, message, stop, archive — scoped to
  the client's own sessions.
- `GET /api/sessions/{name}/stream` — the session's event stream as server-sent
  events: the replay of the current lines first, then the live events.

The `/admin` surface never reaches the network. A peer reaches only the sessions
it owns, because every session it creates is owned by its client. The peer
listener starts only when the config names a listen address. See `internal/mcp`
(`StartPeer`).

## The stream

The stream carries a serialized session event (`internal/wire`): the rendered
lines, the snapshot, the partial text, the todos, and the questions — everything
the output pane reads. The host that views the session decodes each event and
republishes it to its own bus under a local name, so the pane treats a remote
session as local. `render.Line` is width-independent, so the viewing host draws
the lines at its own width.

# Auth

A peer authenticates with the client-credentials grant. Each host provisions an
API client for each peer (`create_api_client`), and the peer stores that
`client_id` and `client_secret` in its config. The peer's `peer.Client` runs the
grant, caches the access token until it expires, and retries once with a fresh
token when the host answers `401`. See `internal/peer`.

# The config

The `peers` block in `config.json`. No block, or an empty `listen`, keeps the
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
- `reserve.window` — the window a usage reserve guards: `5h` or `7d`.
- `reserve.min_percent` — the percent-remaining floor. The reserve is stored and
  read, but its enforcement (pause hosted sessions, refuse a new peer session)
  arrives with the remote-session capability. Omit `reserve` for no floor.
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

Manage the config (control tools, next to the API-client tools, because a peer
entry holds a credential and the listen address changes the network exposure):

- `list_peers` — the listen address, the reserve, and each peer host (the name,
  the url, and the client id). No secret is shown.
- `set_peer_listen` / `unset_peer_listen` — set or clear the listen address.
- `add_peer` / `remove_peer` — add a peer host (name, url, client id, secret),
  or remove one by name.
- `set_reserve` / `unset_reserve` — set the window and the floor, or clear it.

A change to the listen address or a peer host takes effect on the next restart,
the same as a hand edit of the file. Each tool writes the config and returns the
path it wrote.

# Turn on peering

1. Add `peers.listen`, e.g. `0.0.0.0:51900` (or run `set_peer_listen`).
2. For each host that will reach this host, run `create_api_client`, and give
   the `client_id` and `client_secret` to that host's `peers.hosts` block (or
   run `add_peer` there).
3. Restart. The peer listener binds the address.
