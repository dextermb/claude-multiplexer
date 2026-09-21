# Jira over the REST API

Status: in progress. The transport choice is decided. The default endpoint
stays the Rovo MCP server.

## Why

The Rovo MCP server refuses a personal API token until an organisation admin
turns on API token authentication. The refusal arrives on the product call,
not on the handshake:

```
getJiraIssue: {"error":true,
 "message":"You don't have permission to connect via API token.
            Please ask your organization admin for access."}
```

The same token reads the same issue through the plain Jira REST API. So a
second transport gives a working path while the admin gate is shut.

## Decisions

**One key names the endpoint.** `workItems.jira.url` keeps its job. A URL whose
path ends in `/mcp` is an MCP endpoint. Any other URL is a Jira site base, and
selects the REST transport. An empty URL stays the Rovo MCP default.

Rejected: a second `api: mcp|rest` key. Two keys can disagree, and the URL
already says which server it names.

**The default does not move.** A new install still reaches the Rovo MCP server.
A human who wants REST sets `url` to the site base.

**The parsers are shared.** The REST bodies carry the same shapes the MCP
results carry, so `parseJiraIssue` and `parseJiraTransitions` read both.

## Data flow

```
Resolve(key)
  GET  {site}/rest/api/3/issue/{key}?fields=summary,status   -> parseJiraIssue
  URL  {site}/browse/{key}                                    (built, not read)

Statuses(key)
  GET  {site}/rest/api/3/issue/{key}/transitions             -> parseJiraTransitions

SetStatus(key, target)
  Statuses(key) -> matchStatus(target) -> transition id
  POST {site}/rest/api/3/issue/{key}/transitions  {"transition":{"id":id}}
  Resolve(key)                                                (re-read)
```

The REST transport resolves no site. The base url is the configured url, so
there is no `getAccessibleAtlassianResources` call and no cloudId.

## The build

- `internal/workitem/jira_rest.go` — `jiraREST`, implementing `Provider`.
- `internal/workitem/workitem.go` — pick `jiraREST` or `jira` from the url.
- `internal/config/workitems.go` — the url no longer means an MCP endpoint
  only.
- Tests — an `httptest` Jira that serves the three routes.
- Docs — `docs/work-items.md` gains the REST section.
