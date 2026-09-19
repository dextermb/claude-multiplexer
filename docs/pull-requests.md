# Pull requests: a branch shows its PR in the output bar

The multiplexer reads the pull request of a session's branch on GitHub or
GitLab, and shows it in the top bar of the output pane. The bar shows the number
and the count of unresolved review threads, for example `#1045 (3)` for a GitHub
pull request, or `!1045 (3)` for a GitLab merge request. The prefix names the
provider: `#` for GitHub, `!` for GitLab.

The branch is the key. There is no link step. The multiplexer reads the branch
and the `origin` remote of the session directory, infers the provider from the
remote host, then finds the most recent pull request whose source branch is that
branch.

The feature is off until a provider is configured. With no provider, the
`get_pr` tool does not appear.

## Set up GitHub

GitHub works two ways. Choose one with the `mode` field.

- **The API.** Give a token with `repo` read scope to the `configure_github`
  tool. The multiplexer sends the token as a bearer token to the GraphQL API.
- **The CLI.** Set the mode to `cli` and give no token. The multiplexer runs
  `gh api graphql`, which carries your `gh` login. The `gh` CLI must be on the
  `PATH`.

The default mode is `auto`: a token selects the API, and with no token the
multiplexer runs the `gh` CLI when it is on the `PATH`.

## Set up GitLab

GitLab works the same two ways.

- **The API.** Give a token with `read_api` scope to the `configure_gitlab`
  tool. The multiplexer sends the token as a bearer token to the GraphQL API.
- **The CLI.** Set the mode to `cli` and give no token. The multiplexer runs
  `glab api graphql`, which carries your `glab` login. The `glab` CLI must be on
  the `PATH`.

## The two transports

Each provider runs one GraphQL query, and the transport changes only how the
query is sent.

- **api** sends the query over HTTP to the GraphQL endpoint, with the token.
- **cli** sends the same query through `gh api graphql` or `glab api graphql`.
- **auto** takes the API when a token is set, else the CLI when the tool is on
  the `PATH`.

## Provider inference

The multiplexer reads the remote host to pick the provider. `github.com` selects
GitHub, and `gitlab.com` selects GitLab. For a self-hosted server, add the host
to the provider's `hosts` list: a GitHub Enterprise host to `github`, and a
self-hosted GitLab host to `gitlab`.

## When the tools appear

A session takes the `get_pr` tool when it starts, so configure a provider first,
then start a session. The `configure_github` and `configure_gitlab` tools are
always available, so a session can turn the feature on.

## The settings

One block, `pullRequests`, holds a provider entry per platform. Both may be set
at once. See [config.md](config.md).

```json
{
  "pullRequests": {
    "github": { "token": "…", "mode": "auto", "hosts": ["github.mycorp.com"] },
    "gitlab": { "mode": "cli", "hosts": ["gitlab.mycorp.com"] }
  }
}
```

| Key | What it holds |
|---|---|
| `pullRequests.github.token` | A GitHub token with `repo` read scope, for the API |
| `pullRequests.github.mode` | `auto`, `api`, or `cli`; empty is `auto` |
| `pullRequests.github.url` | The GraphQL endpoint, or empty for the default |
| `pullRequests.github.hosts` | Extra hosts that select GitHub (Enterprise) |
| `pullRequests.gitlab.token` | A GitLab token with `read_api` scope, for the API |
| `pullRequests.gitlab.mode` | `auto`, `api`, or `cli` |
| `pullRequests.gitlab.url` | The GraphQL endpoint, or empty for the default |
| `pullRequests.gitlab.hosts` | Extra hosts that select GitLab (self-hosted) |

The `configure_github` and `configure_gitlab` tools write these keys, so setup
needs no dot path. The generic `set_config` tool still writes a key, for example
`set_config pullRequests.github.hosts …`. The settings file is `0o600`, so the
token sits with the other private settings.

A provider is on when its block holds a token (for the API), or its mode allows
the CLI and the tool is on the `PATH`. A block with no token and no CLI is off.

### The endpoints

| Provider | Default endpoint |
|---|---|
| GitHub | `https://api.github.com/graphql` |
| GitLab | `https://gitlab.com/api/graphql` |

Override the endpoint with the `url` key, for a GitHub Enterprise or a
self-hosted GitLab server.

## The tools

| Tool | What it does |
|---|---|
| `configure_github` | Set the GitHub token and mode. Always available. |
| `configure_gitlab` | Set the GitLab token and mode. Always available. |
| `get_pr` | The pull request of a session's branch, with its number, state, url, and unresolved count. |

There is no `set_pr`, because the branch is the key. The `get_pr` tool reads the
branch and the remote of the session directory, runs one live lookup, mirrors
the result, and returns it.

## The poll

The unresolved-thread count of a pull request changes over time, so the mirror
needs a background poll. A manager worker sweeps every 60 seconds. It reads the
setting on each sweep, so a change takes effect on the next tick.

One sweep reads the branch and remote of every live session, groups the branches
by provider, host, and credential, then sends one batched GraphQL request per
group. Two sessions on the same repository and branch cost one lookup. So a
fleet costs a few requests a sweep, not one per session.

The poll reads every live session, but the interface draws the badge in the top
bar of the selected session only.

## What moves through the change

The multiplexer stores a mirror of the pull request in the session `meta.json`:
`pr_provider`, `pr_number`, `pr_url`, `pr_state`, `pr_title`, `pr_unresolved`,
`pr_branch`, and `pr_synced_at`. See [manager.md](manager.md).

- **Poll.** The sweep reads each branch's pull request, then writes the mirror.
  It writes a session only when a field changes, to limit disk churn.
- **Branch switch.** `pr_branch` records the branch the mirror belongs to. A
  branch switch makes the mirror stale, and the next sweep overwrites or clears
  it.
- **Read.** The top bar and the `get_pr` tool read the mirror. A transient error
  keeps the last known mirror. A merged or closed pull request keeps its mirror,
  so the bar shows the final state.

## In the interface

The top bar of the output pane shows the badge for the selected session, next to
the diff summary.

| The pull request | The badge |
|---|---|
| Open, no unresolved threads | `#1045` (GitHub), `!1045` (GitLab) |
| Open, three unresolved threads | `#1045 (3)` |
| Draft | `#1045 draft (3)` |
| Merged | `#1045 merged` |
| Closed | `#1045 closed` |
| No pull request for the branch | no segment |

An unresolved count above zero takes a warning colour, so a review that waits
stands out.

## The two query models

The two platforms name a pull request differently, so one provider interface has
two implementations. See `internal/pullrequest`.

- **GitHub.** The query reads `repository(owner, name).pullRequests(headRefName:
  …)`, and the unresolved count is the number of `reviewThreads` that are not
  resolved.
- **GitLab.** The query reads `project(fullPath: …).mergeRequests(sourceBranches:
  …)`, and the unresolved count is `resolvableDiscussionsCount` minus
  `resolvedDiscussionsCount`.

## What is verified, and what is not

The remote-url parser, the provider selection, the grouping and chunking, and
the two query models are covered by tests against a fake GraphQL server and a
recording CLI, in `internal/pullrequest`, and by the manager and MCP tests. A
real run against a live GitHub repository and a live GitLab repository, in both
the API mode and the CLI mode, is the remaining check. It is the one place to
correct a wire field name.
