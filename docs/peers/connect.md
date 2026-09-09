# Connect two hosts

How to let one cmux host start and view sessions on another. See
[../peers.md](../peers.md) for the model, the config, and the tools.

## The two roles

- **Host** — the machine that runs the sessions. It turns peering on and issues
  the credentials. A session it runs for a peer shows under `hosted`.
- **Client** — the machine that starts and views the sessions. It adds the host
  as a peer. Those sessions show under `streamed`.

One host runs two listeners:

```
 one cmux host
 +-----------------------------------------------------------+
 |  loopback API   127.0.0.1:5189x   own sessions, /admin    |  never on the network
 |  peer listener  0.0.0.0:51900     /token, /api/..., stream|  peers on the LAN
 +-----------------------------------------------------------+
```

The client reaches the host over the peer listener:

```
  CLIENT (laptop)                        HOST (workstation)
  starts and views                       runs the session
 +----------------+                     +---------------------+
 |                |  POST /token        |  peer listener      |
 |  new session   | ------------------> |  0.0.0.0:51900      |
 |  host = ws     |  POST /api/sessions |                     |
 |                | ------------------> |  starts a session   |
 |  streamed      |  GET .../stream SSE |  owned by CLIENT,    |
 |  session   <---+---------------------|  shown as hosted    |
 |                |  POST message, stop |                     |
 |                | ------------------> |                     |
 +----------------+                     +---------------------+
  remote -> streamed                     remote -> hosted
```

## On the host (runs the sessions)

- Run `enable_peering`. Add a `port` to change the default `51900`.
- Open that port to the LAN in the firewall.
- Run `create_api_client` for the peer. Copy the `client_id` and the
  `client_secret` at once, because the secret shows one time.
- To keep a share of your own usage, run `set_reserve`.
- Restart cmux. The peer listener binds `0.0.0.0:51900`.
- Give the client your address, for example `http://192.168.1.20:51900`.

## On the client (starts the sessions)

- Run `add_peer` with a name, the host address, the `client_id`, and the
  `client_secret`.
- Restart cmux.
- Run `peer_usage` to confirm the host is reachable.
- Press `n` for a new session. Set the `Host` field to the peer.
- Choose a directory that exists on the host, or leave `Directory` blank for a
  temporary directory the host makes. Then submit.
- Read the session under `remote sessions` -> `streamed`. On the host it shows
  under `hosted`. See [../tui/sessions.md](../tui/sessions.md).

## Notes

- This guide connects one direction. For both directions, each host does both
  parts.
- A blank `Directory` for a peer makes a fresh temporary directory on the host,
  so you start a session without knowing the host's paths. The host removes it
  when the session did no work, or when the session is removed. A session that
  did work keeps its directory, like any other.
- A host that is off, or unreachable, shows an error and never blocks the
  client.
- To disconnect, run `disable_peering` on the host, or `remove_peer` on the
  client.
- When the host regenerates the client secret, or its address changes, run
  `update_peer` on the client with the name and the changed field.
