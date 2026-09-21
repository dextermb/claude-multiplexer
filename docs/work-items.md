# Work items: a session links to Jira or Linear

A session links to one work item in Jira or Linear. The multiplexer shows the
item status in the interface, and a session can change the status. The
multiplexer is the client to the provider, so the token stays in the
multiplexer and the agent needs no provider tools.

The feature is off until a provider token is set. With no provider, the
work-item tools do not appear.

## Set up Linear

Linear uses a personal API key, sent as a bearer token.

1. Open Linear. Select your profile icon, then select Settings.
2. In the sidebar, select Security & access.
3. Find the Personal API keys section. Select New API key.
4. Name the key. Give it write access, so a status change is allowed.
5. If you want to limit the key, restrict it to the teams you work in.
6. Copy the key now, because Linear shows it one time only.
7. Give the key to the `configure_linear` tool.

Linear needs no email, because it uses a bearer token.

## Set up Jira

Jira supports two token types. Both need one prerequisite.

An organisation admin must first turn on API token authentication. The admin
does this in Atlassian Administration, under Rovo, then Rovo MCP server, then
Authentication. Without this step, a token is refused.

**A personal API token (Basic auth).**

1. Open `https://id.atlassian.com/manage-profile/security/api-tokens`.
2. Select Create API token. Name the token, then create it.
3. Copy the token now, because Atlassian shows it one time only.
4. Give the token and your email to the `configure_jira` tool.

The email selects Basic auth, so the multiplexer sends `base64(email:token)`.

**A service-account API key (bearer).**

1. Ask your admin for a service-account API key, with the scopes the tools need.
2. Give the key to the `configure_jira` tool, with no email.
3. With no email, the multiplexer sends a bearer token.

A token needs read and write Jira access, so the multiplexer can read an item
and change its status. Both token types use the default endpoint, the Rovo MCP
server.

### The Jira REST API, when the MCP server refuses the token

The Rovo MCP server refuses a personal API token until the admin turns on API
token authentication. The refusal arrives on the first product call, not on the
handshake:

```
getJiraIssue: {"error":true,
 "message":"You don't have permission to connect via API token.
            Please ask your organization admin for access."}
```

The Jira REST API has no such gate, and the same personal token reads and moves
the same issue there. To use it, give the site base as the url:

```
configure_jira  token: …  email: you@corp.com  url: https://acme.atlassian.net
```

The url selects the transport. A url whose path ends in `/mcp` is an MCP
server. Any other url is a Jira site base, and the multiplexer calls
`/rest/api/3/issue/…` under it. An empty url stays the Rovo MCP default, so an
installation that already works does not move.

The REST transport resolves no site, because the site base is the url. So it
makes no `getAccessibleAtlassianResources` call and carries no cloudId.

## When the tools appear

A session takes the work-item tools when it starts. So set the token first, then
start a session. A session that started before the token was set does not carry
the tools, and a restart gives them.

## The settings

One block, `workItems`, holds a provider entry per platform. Both may be set at
once. A provider with no token is off. See [config.md](config.md).

```json
{
  "workItems": {
    "jira":   { "token": "…", "email": "you@corp.com" },
    "linear": { "token": "…" }
  }
}
```

| Key | What it holds |
|---|---|
| `workItems.jira.token` | The Jira API token or service-account key |
| `workItems.jira.email` | The account email, set only for a Jira personal token |
| `workItems.jira.url` | A Jira site base for the REST API, an MCP endpoint ending in `/mcp`, or empty for the default |
| `workItems.linear.token` | The Linear API key |
| `workItems.linear.url` | The MCP endpoint, or empty for the default |

The `configure_jira` and `configure_linear` tools write these keys, so setup
needs no dot path. The generic `set_config` tool still writes a key, for example
`set_config workItems.jira.url …` to change one endpoint, and `unset_config
workItems.jira` removes a provider. The settings file is `0o600`, so the token
sits with the other private settings.

### Auth

The multiplexer sends the token in the `Authorization` header. An `email`
selects Basic auth, `base64(email:token)`, the Jira personal token. With no
`email` the client sends a bearer token, which is the Jira service-account key
and the only Linear path. The multiplexer does not run the OAuth browser flow,
because it is a daemon.

### The endpoints

| Provider | Default endpoint |
|---|---|
| Jira | `https://mcp.atlassian.com/v2/mcp` |
| Linear | `https://mcp.linear.app/mcp` |

Both defaults are remote, cloud-hosted MCP servers over Streamable HTTP. A Jira
url that is not an MCP endpoint selects the REST transport instead.

| Jira url | Transport | Calls |
|---|---|---|
| empty, or a path that ends in `/mcp` | MCP | `getJiraIssue`, `getTransitionsForJiraIssue`, `transitionJiraIssue` |
| a site base, such as `https://acme.atlassian.net` | REST | `GET`/`POST` under `/rest/api/3/issue/{key}` |

## The tools

The `configure_jira` and `configure_linear` tools are always available, so a
session turns the feature on. The rest appear only when a provider is
configured. See [mcp/tools.md](mcp/tools.md).

| Tool | What it does |
|---|---|
| `configure_jira` | Set the Jira token, an email for a personal token, and a url to choose the transport. Always available. |
| `configure_linear` | Set the Linear API key. Always available. |
| `set_workitem` | Link this session to an item by key. Names a `provider` when more than one is configured, else takes the sole one. |
| `set_workitem_status` | Move the linked item to a status. The status must be one the platform offers. |
| `list_workitem_statuses` | The statuses the linked item may move to now. |
| `get_workitem` | The item a session links to, and its last known status. |
| `unset_workitem` | Clear the link. |

The status name is the platform's own. The multiplexer holds no status
vocabulary of its own, and a name that is not on the platform list is an error
that reports the valid names.

For an example rule that tells an agent when to call these tools, see
[example-rules/work-item-progress.md](example-rules/work-item-progress.md).

## What moves through the change

The multiplexer stores the link and a mirror of the status in the session
`meta.json`: `workitem_provider`, `workitem_key`, `workitem_url`,
`workitem_status`, `workitem_status_id`, and `workitem_synced_at`. See
[manager.md](manager.md).

- **Link.** `set_workitem` reads the item on the provider, then writes the
  title, url, and status into the mirror. It renames the session to the item
  key, so the key names the row.
- **Set status.** `set_workitem_status` changes the status on the provider, then
  writes the new status into the mirror.
- **Read.** The mirror refreshes when a work-item tool runs. There is no
  background poll, and no status changes on a session start or stop. The only
  writer is `set_workitem_status`.

## In the interface

The mirror shows in three places:

- **The output pane status bar** shows the status of the selected session, next
  to the effort.
- **The sidebar groups** a session that links to a work item by its status. The
  status groups form a cluster above the directory groups, in status order, and
  each header names the status. A session with a link, but no status yet, sits
  in the "no status" group.
- **The session name** becomes the item key on the link, so a row names its item
  inside the status group. The unlink clears the rename, unless you changed the
  name yourself.

## The two status models

The two platforms name and change a status differently, so one provider
interface has two implementations. See `internal/workitem`.

- **Linear.** A status is a workflow state, and it is team-scoped. The
  multiplexer reads the team of the issue, lists the team statuses, then calls
  `save_issue` with the target state.
- **Jira.** A status changes through a workflow transition, not by name. The
  multiplexer reads the valid transitions with `getTransitionsForJiraIssue`,
  matches the target to the status a transition moves to, then calls
  `transitionJiraIssue` with that transition. It resolves the site `cloudId`
  once with `getAccessibleAtlassianResources`.

## What is verified, and what is not

The client, the provider selection, the name resolution, and the tool and
interface wiring are covered by tests against a fake MCP server, in
`internal/workitem` and `internal/mcp`. The exact wire shapes the live servers
return are read by tolerant parsers in `internal/workitem/parse.go`,
`linear.go`, and `jira.go`. A real run against a live provider is the remaining
check, and it is the one place to adjust a field name.
