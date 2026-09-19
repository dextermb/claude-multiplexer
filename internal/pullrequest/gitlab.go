package pullrequest

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// gitlab reads a merge request through the GitLab GraphQL API. The query reads
// the most recent MR of a source branch, and the resolvable and resolved
// discussion counts. See docs/pull-requests.md.
type gitlab struct{}

func (gitlab) fetch(ctx context.Context, t transport, queries []Query) []Result {
	return runFetch(ctx, t, queries, buildGitLab, parseGitLab)
}

func buildGitLab(queries []Query) (string, map[string]string) {
	vars := map[string]string{}
	var decl, body strings.Builder
	for i, q := range queries {
		p, b := "p"+alias(i), "b"+alias(i)
		vars[p] = q.Repo.FullPath()
		vars[b] = q.Branch
		if i > 0 {
			decl.WriteString(", ")
		}
		decl.WriteString("$" + p + ":ID!, $" + b + ":String!")
		body.WriteString(aliasOf(i) + ": project(fullPath: $" + p + ") { " +
			"mergeRequests(sourceBranches: [$" + b + "], first: 1, sort: CREATED_DESC) { " +
			"nodes { iid webUrl state draft title resolvableDiscussionsCount resolvedDiscussionsCount } } } ")
	}
	query := "query(" + decl.String() + ") { " + body.String() + "}"
	return query, vars
}

type glNode struct {
	IID        string `json:"iid"`
	WebURL     string `json:"webUrl"`
	State      string `json:"state"`
	Draft      bool   `json:"draft"`
	Title      string `json:"title"`
	Resolvable int    `json:"resolvableDiscussionsCount"`
	Resolved   int    `json:"resolvedDiscussionsCount"`
}

func parseGitLab(data json.RawMessage, queries []Query) []Result {
	byAlias := map[string]json.RawMessage{}
	_ = json.Unmarshal(data, &byAlias)
	out := make([]Result, len(queries))
	for i := range queries {
		var project struct {
			MergeRequests struct {
				Nodes []glNode `json:"nodes"`
			} `json:"mergeRequests"`
		}
		raw := byAlias[aliasOf(i)]
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		if err := json.Unmarshal(raw, &project); err != nil || len(project.MergeRequests.Nodes) == 0 {
			continue
		}
		node := project.MergeRequests.Nodes[0]
		pr := PR{
			Provider: config.PullRequestGitLab,
			Number:   atoiSafe(node.IID),
			URL:      node.WebURL,
			Title:    node.Title,
			State:    gitlabState(node.State, node.Draft),
		}
		if pr.State == StateOpen || pr.State == StateDraft {
			if unresolved := node.Resolvable - node.Resolved; unresolved > 0 {
				pr.Unresolved = unresolved
			}
		}
		out[i] = Result{PR: pr, Found: true}
	}
	return out
}

func gitlabState(state string, draft bool) string {
	switch strings.ToLower(state) {
	case "merged":
		return StateMerged
	case "closed", "locked":
		return StateClosed
	default:
		if draft {
			return StateDraft
		}
		return StateOpen
	}
}

func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
