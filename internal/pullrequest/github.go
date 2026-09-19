package pullrequest

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// github reads a pull request through the GitHub GraphQL API. The query reads
// the most recent PR of a branch, and its unresolved review threads. See
// docs/pull-requests.md.
type github struct{}

func (github) fetch(ctx context.Context, t transport, queries []Query) []Result {
	return runFetch(ctx, t, queries, buildGitHub, parseGitHub)
}

func buildGitHub(queries []Query) (string, map[string]string) {
	vars := map[string]string{}
	var decl, body strings.Builder
	for i, q := range queries {
		o, n, b := "o"+alias(i), "n"+alias(i), "b"+alias(i)
		vars[o] = q.Repo.Owner
		vars[n] = q.Repo.Name
		vars[b] = q.Branch
		if i > 0 {
			decl.WriteString(", ")
		}
		decl.WriteString("$" + o + ":String!, $" + n + ":String!, $" + b + ":String!")
		body.WriteString(aliasOf(i) + ": repository(owner: $" + o + ", name: $" + n + ") { " +
			"pullRequests(headRefName: $" + b + ", first: 1, orderBy: {field: CREATED_AT, direction: DESC}) { " +
			"nodes { number url state isDraft title reviewThreads(first: 100) { nodes { isResolved } } } } } ")
	}
	query := "query(" + decl.String() + ") { " + body.String() + "}"
	return query, vars
}

type ghNode struct {
	Number        int    `json:"number"`
	URL           string `json:"url"`
	State         string `json:"state"`
	IsDraft       bool   `json:"isDraft"`
	Title         string `json:"title"`
	ReviewThreads struct {
		Nodes []struct {
			IsResolved bool `json:"isResolved"`
		} `json:"nodes"`
	} `json:"reviewThreads"`
}

func parseGitHub(data json.RawMessage, queries []Query) []Result {
	byAlias := map[string]json.RawMessage{}
	_ = json.Unmarshal(data, &byAlias)
	out := make([]Result, len(queries))
	for i := range queries {
		var repo struct {
			PullRequests struct {
				Nodes []ghNode `json:"nodes"`
			} `json:"pullRequests"`
		}
		raw := byAlias[aliasOf(i)]
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		if err := json.Unmarshal(raw, &repo); err != nil || len(repo.PullRequests.Nodes) == 0 {
			continue
		}
		node := repo.PullRequests.Nodes[0]
		pr := PR{
			Provider: config.PullRequestGitHub,
			Number:   node.Number,
			URL:      node.URL,
			Title:    node.Title,
			State:    githubState(node.State, node.IsDraft),
		}
		if pr.State == StateOpen || pr.State == StateDraft {
			for _, th := range node.ReviewThreads.Nodes {
				if !th.IsResolved {
					pr.Unresolved++
				}
			}
		}
		out[i] = Result{PR: pr, Found: true}
	}
	return out
}

func githubState(state string, draft bool) string {
	switch strings.ToUpper(state) {
	case "MERGED":
		return StateMerged
	case "CLOSED":
		return StateClosed
	default:
		if draft {
			return StateDraft
		}
		return StateOpen
	}
}

func alias(i int) string { return strings.TrimPrefix(aliasOf(i), "a") }
