# The external API

A program outside the multiplexer reaches the sessions over HTTP. The example
client is Bruno, an API client. The API runs on the same HTTP server as the MCP
endpoint, on loopback. It is off until a human turns it on from inside a session.

Two kinds of caller reach the server:

- A **session** is a Claude Code child. It holds a per-session bearer token, and
  it reaches the MCP endpoint. See [transport.md](transport.md).
- A **client** is a program outside the multiplexer. It holds a `client_id` and
  a `client_secret`. It exchanges the pair for a short-lived access token, and it
  reaches the REST surface and the MCP endpoint with that token.

## Two secrets

The API has two layers of secret:

1. The **admin secret** is the master key. It gates the client-management
   endpoint. A control session creates and rotates it through an MCP tool.
2. A **client secret** belongs to one client. A control session, or an operator
   with the admin secret, creates, rotates, and revokes it.

The API is off until the admin secret exists. No admin secret means no external
door. The server stores only a salted hash of each secret. A tool shows the
plaintext once, at create and at rotate. A lost secret is rotated, not recovered.

## What a client may do

A client reaches the session tools only, never the local configuration. The API
never exposes the config, editor, block-cap, working-directory, project, layout,
or schedule tools. The client tools are:

- Read: `list_sessions`, `get_messages`, `list_jobs`.
- Title: `rename_session`.
- Drive: `send_message`, `stop_session`, `archive_session`, `create_session`,
  `stop_job`.

There is no permission tier. Every client holds the whole session-only set. The
only boundary is ownership.

## Ownership

A client sees and touches only the sessions it owns. A session carries an owner,
which is the `client_id` that created it. A session a human, a schedule, or a
control session starts has no owner, so no client sees it. The human interface
still shows every session, because the human owns the host.

- A client's `list_sessions` returns only its own sessions.
- A client call on a session it does not own answers as if the session is not
  there: `404` on REST, and a not-found error on MCP. So the API leaks no
  session name across a client boundary.
- Ownership is set once, at create, and it lives in the session record
  (`sessions/<name>/meta.json`, the `owner` field). It survives a restart, so a
  client still sees its stored and archived sessions.

## Where the credentials live

The store sits under the state directory, next to the sessions and the
schedules:

```
~/.claude-multiplexer/
  sessions/<name>/meta.json     the owner field marks the client that created it
  api/
    admin.json                  the admin secret, hashed
    endpoint.json               the base URL, rewritten each start
    clients/
      <client_id>.json          one client record, the secret hashed
```

The manager scans `api/clients/` at start, holds the records in memory, and
writes one file per change. The access token does not persist. It lives in
memory with an expiry, so a client runs the grant again after a restart.

## The port range

The server binds the first free port in a range, on `127.0.0.1`. The default
range is `51890-51899`. The flags `--api-port-start` and `--api-port-end` change
it. The server tries each port in order, and it fails to start only when the
whole range is taken.

The port moves inside the range across restarts, so the manager writes the exact
base URL to `api/endpoint.json` each start. A client reads that file, or it scans
the range, or it calls the `get_api_endpoint` tool from a control session.

## The surfaces

The server mounts four paths on one loopback port:

| Path | Who reaches it | For |
|---|---|---|
| `/mcp` | a session token, or a client access token | the MCP tools |
| `/token` | a client id and secret | the client-credentials grant |
| `/admin/clients` | the admin secret | the client-management endpoint |
| `/api/...` | a client access token | the REST surface |

### The grant (`/token`)

A client runs the OAuth2 client-credentials grant:

```
POST /token
  grant_type=client_credentials, client_id=..., client_secret=...
->
  { "access_token": "...", "token_type": "Bearer", "expires_in": 3600 }
```

The server hashes the secret, compares it to the stored hash, and mints a random
access token. It builds the owner-scoped tool set, and it holds the token in
memory with an expiry. A wrong secret, or a disabled client, gets `401`. On
expiry, a call gets `401`, and the client runs the grant again.

### The REST surface (`/api/...`)

A client that does not speak MCP calls flat routes with the access token. Every
route reads the owner-scoped view, so a route on a session the client does not
own answers `404`.

| Method and route | Action |
|---|---|
| `GET /api/sessions` | list the client's sessions |
| `GET /api/sessions/{name}/messages` | read the transcript |
| `GET /api/sessions/{name}/jobs` | list the background jobs |
| `PATCH /api/sessions/{name}` | set the title |
| `POST /api/sessions` | create a session, owned by this client |
| `POST /api/sessions/{name}/message` | send a prompt |
| `POST /api/sessions/{name}/stop` | stop a session |
| `POST /api/sessions/{name}/archive` | archive or restore a session |
| `POST /api/sessions/{name}/jobs/{id}/stop` | stop a background job |

### The MCP endpoint (`/mcp`)

A client access token also reaches `/mcp`, the same JSON-RPC endpoint a session
uses. The access token drops into the same token map, so the MCP handler serves a
client with no change. The tool set is the session-only set, over the
owner-scoped view.

### The admin endpoint (`/admin/clients`)

An operator that works only over HTTP manages the clients with the admin secret
in the `Authorization` header:

| Method and route | Action |
|---|---|
| `GET /admin/clients` | list the clients |
| `POST /admin/clients` | create a client, return its secret once |
| `PATCH /admin/clients/{id}` | rename or disable a client |
| `POST /admin/clients/{id}/rotate` | issue a new secret, return it once |
| `DELETE /admin/clients/{id}` | remove a client |

The MCP tools cover the same actions, so the admin endpoint is for an operator
without a session.

## The tools a control session holds

A control session manages the API through these tools:

- `create_api_admin`, `rotate_api_admin`, `revoke_api_admin` — the admin secret.
- `create_api_client`, `update_api_client`, `rotate_api_client`,
  `revoke_api_client`, `list_api_clients` — the clients.
- `get_api_endpoint` — the base URL and the port range.

A rotate or a revoke drops the client's live access tokens at once, so a leaked
secret stops working immediately.

## Workflows

**Turn the API on.** A human asks a control session to enable the API. The
session calls `create_api_admin`, and the tool returns the admin secret once and
the base URL. The human copies the secret into the client.

**Create a client.** A control session calls `create_api_client` with a name, or
an operator POSTs to `/admin/clients` with the admin secret. Either way the
response holds the `client_id` and the `client_secret` once.

**Use the API.** The client runs the grant at `/token`, keeps the access token,
and calls `/api/...` or `/mcp`. It creates a session, then reads and drives only
the sessions it owns. On `401`, it runs the grant again.

## Security

Warning: the API widens the boundary that is a per-session token. Hold these
lines, because a client secret in the wrong hands drives sessions on the host:

- The API binds only `127.0.0.1`. A remote client reaches it through an SSH
  tunnel, not a public bind.
- The store holds only a hash of each secret. The files are `0600`.
- The API stays off until a human creates the admin secret in a session.
- Only a control session manages the credentials.
- A rotate or a revoke drops the client's live tokens at once.
- A client reaches only its own sessions, and any other session answers `404`.
