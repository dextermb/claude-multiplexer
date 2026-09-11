# Watch a session read-only

A host shares one of its sessions read-only. It mints a share link, and whoever
holds the link watches the session live, but cannot type, stop, or interrupt it.
This is a spectator session. See [../peers.md](../peers.md) for the peer model,
and [connect.md](connect.md) for the peer listener the share rides on.

## The two roles

- **Host** — the machine that runs the session. It turns peering on and mints
  the share. The shared session stays a normal session on the host.
- **Spectator** — the machine that watches. It needs no peer listener and no
  client: the share link is a self-contained bearer credential.

```
  HOST (runs the session)                 SPECTATOR (watches)
 +----------------------+                +----------------------+
 |  share_session       |                |                      |
 |  -> a spectate link  | --- link --->  |  watch_share <link>  |
 |                      |                |                      |
 |  peer listener       |  GET /api/     |  read-only streamed  |
 |  0.0.0.0:51900       |  shares/{id}/  |  session, no input    |
 |                      |  stream (SSE)  |                      |
 +----------------------+ -------------> +----------------------+
```

## The share

A share opens one session, read-only, until it expires or the host revokes it.
The host stores each share under `~/.claude-multiplexer/api/shares/<id>.json`,
with the secret hashed like a client secret. A share expires 24 hours after it
is minted, by default.

The share token is `<share_id>.<secret>`. The link bundles the peer-listener URL
and the token in one base64url blob, so a person copies it as one string:

```
cmux://spectate/<blob>

blob = base64url({
  "u": "http://192.168.1.20:51900",   the peer-listener base URL (get_peer_url)
  "t": "shr_A1B2C3.<secret>"          the share token
})
```

The host shows the link once, at mint, the same way a client secret shows once.

## The surface

The peer listener serves a read-only surface under `/api/shares/`. Every route
reads the share token from the `Authorization: Bearer` header, checks the secret,
and checks the share is not expired and not revoked. A bad, expired, or revoked
token gets `401`.

| Method and route | For |
|---|---|
| `GET /api/shares/{id}` | the session label (title, model) so the viewer draws a header |
| `GET /api/shares/{id}/stream` | the session's events (SSE), read-only |
| `GET /api/shares/{id}/messages` | the transcript, read-only |

The share maps its id to the session inside the host, so the spectator never
learns the host-local session name. Each event on the stream carries the share
id as its session field, not the host-local name.

## Read-only is the absence of the write routes

There is no `message`, `stop`, `interrupt`, or `archive` route under
`/api/shares/`. So the surface cannot drive the session, whatever a spectator
sends. This is the boundary that holds, and it is server-side.

On the spectator side, the interface also marks the session read-only: it shows
the `R` flag, it takes no prompt, and the stop and interrupt keys do nothing.
This explains the read-only state to the human; it is not the boundary.

A revoked or expired share ends the watch. The host closes the live stream, and
the spectator's stream returns `401`, which is terminal: the spectator session
leaves the sidebar instead of reconnecting. The spectator also stops watching at
any time — `s x` on a spectator session detaches the stream locally, and never
touches the session on the host.

## The host sees who watches

The host marks a session a spectator watches now with the `W` flag. The flag
follows the live stream: it shows when a spectator connects, and it clears when
the last spectator stops. A minted-but-unwatched share shows no flag, because
nobody watches yet. See [../tui/sessions.md](../tui/sessions.md).

## The tools

Who may share:

- A control session shares any session on the host.
- A plain session shares only itself. It omits the session argument, or names
  itself; naming another session fails.

Host (mint and manage):

- `share_session` — mint a read-only share for a session. It takes the session
  name and an optional `expires_hours` (omit for 24, or 0 for no expiry). It
  returns the share id and the link once. Peering must be on.
- `list_shares` — the active shares: the id, the session, and the expiry. No
  secret. Control only.
- `revoke_share` — end a share by id. A live spectator drops within a few
  seconds. Control only.

Spectator (watch — control only):

- `watch_share` — take a link, attach a read-only streamed session, and return
  its local name.

## Share a session (host)

1. Turn peering on, if it is off (`enable_peering`, then restart). The link
   points at the peer listener.
2. Run `share_session` with the session name. Copy the link it returns once.
3. Give the link to the person who watches.
4. To end the watch, run `revoke_share` with the id from `list_shares`.

## Watch a share (spectator)

1. Run `watch_share` with the link.
2. Read the session under `remote sessions` -> `streamed`, with the `R` flag.
   The prompt is off, and the stop and interrupt keys do nothing.
3. Press `s x` to stop watching. This detaches the stream; the session on the
   host is untouched.

## A note on privacy

Warning: a share hands the whole rendered session to whoever holds the link.
This includes every tool call, and any file content the session prints. Share a
link only with a person who may read all of that session, and revoke it when the
watch is over.
