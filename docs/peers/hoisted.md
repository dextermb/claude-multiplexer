# Hoisted sessions

A hoisted session runs Claude Code on the borrower's own machine, but it
authenticates with a lender's Claude credential. It sits beside the hosted
session in [../peers.md](../peers.md), where the lender runs Claude Code and the
borrower streams it. Hoisting is opt-in: a peer runs a hoisted session only when
it holds a lent credential.

## The two roles

The words match the peer roles in [../peers.md](../peers.md):

- **lender** — the host. It owns a Claude credential and provisions the API
  client. It creates the lent credential and gives it to the borrower.
- **borrower** — the peer. It runs Claude Code on its own machine with the lent
  credential.

The credential owner is the same party in both models. Hosting keeps the run on
the host. Hoisting moves the run to the peer, and the credential with it.

## Hosted beside hoisted

| | Hosted | Hoisted |
|---|---|---|
| Where Claude Code runs | On the lender | On the borrower |
| Whose credential | Lender's | Lender's |
| Transport | The borrower streams the session | The borrower runs it locally |
| Sidebar band | streamed (viewer), hosted (runner) | local |
| Needs the peer listener | Yes | No — the credential travels by copy |
| The lender's reserve gates it | Yes | No (see Cautions) |

The last two rows are the key differences. Hoisting needs no live link after the
operator copies the credential. And the lender gives up reserve control, because
the lender does not run the session.

## The credential

A lent credential has a type and a value:

- `token` — a Claude access token, from `claude setup-token`. It runs as the
  environment variable `CLAUDE_CODE_OAUTH_TOKEN`.
- `key` — an Anthropic API key. It runs as the environment variable
  `ANTHROPIC_API_KEY`.

The operator gives the type, because the multiplexer does not read the value to
guess it. The multiplexer does not mint a credential — the operator supplies one
from Anthropic.

## The tooling

### On the lender

A control session lends a credential to an API client it already provisioned
(see [../mcp/api.md](../mcp/api.md)):

- `create_api_key` — give the client (id or name), the type (`token` or `key`),
  and the value. It stores only the type and the last four characters, and
  returns the value once to share with the peer.
- `revoke_api_key` — remove the lent-credential record from a client.
- `list_api_clients` — shows the type and the last four for a client that lent a
  credential. It never shows the value.

The lender stores metadata only. The borrower holds the only full copy.

### On the borrower

A control session adds the credential to the peer host block:

- `add_peer` — takes an optional `credential_type` and `credential`. A peer used
  only to hoist needs no `url`, `client_id`, or `client_secret`.
- `update_peer` — takes the same fields to add or replace a credential, and
  `clear_credential` to remove it.
- `list_peers` — shows the type and the last four for a peer that holds a
  credential. It never shows the value.

## The config

The borrower's config holds the full value, because the borrower injects it. The
`peers.hosts[]` entry gains a `credential`:

```json
{
  "peers": {
    "hosts": [
      {
        "name": "studio",
        "url": "http://192.168.1.20:51900",
        "client_id": "cid_...",
        "client_secret": "secret_...",
        "credential": { "type": "token", "value": "sk-ant-oat01-..." }
      }
    ]
  }
}
```

The `credential` present means the borrower may hoist a session from this peer.
The `url`, `client_id`, and `client_secret` stay optional: a hoist-only peer
needs `name` and `credential` alone.

## The run

A hoisted session is a local session with an injected environment. The manager:

1. Reads the peer credential from the config.
2. Builds the environment: it scrubs `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`,
   and `CLAUDE_CODE_OAUTH_TOKEN` from the inherited environment, then sets the one
   variable the credential type needs. So the session carries exactly one Claude
   credential, and the borrower's own credential does not leak in.
3. Sets `CLAUDE_CONFIG_DIR` to a per-lender directory under the state root
   (`<root>/hoisted/<lender>`), so the lender's projects and history stay out of
   the borrower's personal `~/.claude`.
4. Spawns Claude Code locally, and marks the session with its lender.

The scrub matters. A plain append of the injected variable would leave an
inherited `ANTHROPIC_API_KEY` in place, and that key could win over an injected
token. The environment builder drops the scrubbed keys first, so the injected
credential is the only one. See `internal/session` (`env.go`).

### The seed

The per-lender `CLAUDE_CONFIG_DIR` is seeded from the borrower's own Claude Code
setup, so the borrower's customisation applies to the run:

- The borrower's `commands`, `skills`, and `rules` are copied in.
- `claude.json` is symlinked into the directory, so the lender's projects and
  history persist per lender across runs.

A missing source is skipped, so a hoisted session still starts. See
`internal/manager` (`hoist.go`).

Note: the exact location of `claude.json` and of the seeded sub-directories,
relative to `CLAUDE_CONFIG_DIR`, depends on the installed Claude Code version.

## The new-session select

When the borrower chooses a peer that holds a credential, the new-session form
shows a peer-mode select. It defaults to `hoist`, with `stream` as the other
option. A peer with no credential shows no select and streams as before.

On `hoist`, the form keeps the directory path, because the session runs on the
borrower's machine and the local path is valid. On `stream`, the directory is on
the peer, so the field clears. A hoisted session runs locally, so it sits in the
local band of the sidebar, with a muted `H` flag to the right of the name (see
[../tui/sessions.md](../tui/sessions.md)). Its lender is on the session record
that `list_sessions` returns.

## Cautions

Warning: a lent credential is a live key. The borrower stores the full value in
plaintext `config.json`. Any reader of that file can act as the lender's Claude
account. Protect the file, and share the value over a safe channel.

Caution: the reserve does not gate a hoisted session. The lender's reserve pauses
hosted sessions, but it cannot pause a session that runs on the borrower. A
lender that lends a credential gives up this control.

Caution: revoke at Anthropic, not only here. `revoke_api_key` clears the lender's
record only. It does not retract the borrower's copy, and it does not stop the
credential at Anthropic. To stop a lent credential, regenerate it at Anthropic (a
new Console key, or a new `setup-token`), which makes the old one fail
everywhere.

Note: a consumer subscription is for one person. To lend a subscription access
token to another party may break Anthropic's Consumer Terms. A Console API key
bills the lender's account, and is the clearer path. This is the lender's
decision to confirm.

## Turn hoisting on

1. On the lender: run `create_api_client` for the peer (if it has none), then
   `create_api_key` with the type and the value. Copy the value out.
2. Share the value with the borrower over a safe channel.
3. On the borrower: run `add_peer` (or `update_peer`) with the `credential_type`
   and the `credential`, and restart.
4. On the borrower: start a session, choose the peer, and keep the select on
   `hoist`.
