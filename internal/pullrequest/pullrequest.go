// Package pullrequest finds the pull request of a branch on GitHub or GitLab,
// through the HTTP API or the provider CLI. It groups many branches into one
// batched GraphQL request per provider, so a fleet costs a few requests a
// sweep. See docs/pull-requests.md.
package pullrequest

import (
	"context"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// batchCap is the number of branch lookups in one GraphQL request. It stays
// under the platform node and complexity limits. See docs/pull-requests.md.
const batchCap = 20

// PR is one pull request: the fields the interface shows. See
// docs/pull-requests.md.
type PR struct {
	Provider   string `json:"provider"`
	Number     int    `json:"number"`
	URL        string `json:"url,omitempty"`
	State      string `json:"state,omitempty"`
	Title      string `json:"title,omitempty"`
	Unresolved int    `json:"unresolved,omitempty"`
}

// The states a PR shows. A draft is an open PR marked draft. See
// docs/pull-requests.md.
const (
	StateOpen   = "open"
	StateDraft  = "draft"
	StateMerged = "merged"
	StateClosed = "closed"
)

// Repo names the repository a branch lives in, parsed from the git remote.
type Repo struct {
	Host  string
	Owner string
	Name  string
}

// FullPath joins the owner and the name, the form GitLab and GitHub name a
// repository by.
func (r Repo) FullPath() string { return r.Owner + "/" + r.Name }

// BranchRef is one lookup: the git remote url of a repository, and a branch.
type BranchRef struct {
	RemoteURL string
	Branch    string
}

// Result is the outcome of one BranchRef, aligned by index to the input.
type Result struct {
	PR    PR
	Found bool
	Err   error
}

// Query is one branch lookup inside a batch, on a known provider.
type Query struct {
	Repo   Repo
	Branch string
}

// Set holds the configured providers. The manager keeps one, built from the
// settings file. See docs/pull-requests.md.
type Set struct {
	cfg *config.PullRequests
}

// NewSet builds the provider set from the pull-request settings.
func NewSet(cfg *config.PullRequests) *Set { return &Set{cfg: cfg} }

// Enabled reports whether at least one provider has a usable transport: a token
// for the API, or the CLI on the PATH.
func (s *Set) Enabled() bool {
	if s == nil {
		return false
	}
	return s.usable(config.PullRequestGitHub) || s.usable(config.PullRequestGitLab)
}

// Lookup finds the PR of each ref. The result aligns by index to refs. A ref on
// an unknown host, or a provider with no usable transport, gives Found false and
// no error. Two refs that share a repository and branch cost one lookup.
func (s *Set) Lookup(ctx context.Context, refs []BranchRef) []Result {
	out := make([]Result, len(refs))
	targets := make([]*target, len(refs))
	groups := map[string][]int{}
	for i, ref := range refs {
		repo, ok := ParseRepo(ref.RemoteURL)
		branch := strings.TrimSpace(ref.Branch)
		if !ok || branch == "" {
			continue
		}
		provider := s.cfg.ProviderForHost(repo.Host)
		if provider == "" || !s.usable(provider) {
			continue
		}
		targets[i] = &target{provider: provider, host: repo.Host, query: Query{Repo: repo, Branch: branch}}
		key := provider + "\x00" + repo.Host
		groups[key] = append(groups[key], i)
	}
	for _, idxs := range groups {
		s.lookupGroup(ctx, targets, idxs, out)
	}
	return out
}

// lookupGroup runs one provider and host group, deduped and chunked.
func (s *Set) lookupGroup(ctx context.Context, targets []*target, idxs []int, out []Result) {
	if len(idxs) == 0 {
		return
	}
	first := targets[idxs[0]]
	prov := providerFor(first.provider)
	t, ok := s.transportFor(first.provider, first.host)
	if !ok {
		return
	}

	type slot struct {
		query Query
		refs  []int
	}
	var slots []*slot
	seen := map[string]*slot{}
	for _, i := range idxs {
		q := targets[i].query
		key := q.Repo.FullPath() + "\x00" + q.Branch
		if sl, ok := seen[key]; ok {
			sl.refs = append(sl.refs, i)
			continue
		}
		sl := &slot{query: q, refs: []int{i}}
		seen[key] = sl
		slots = append(slots, sl)
	}

	for start := 0; start < len(slots); start += batchCap {
		end := min(start+batchCap, len(slots))
		chunk := slots[start:end]
		queries := make([]Query, len(chunk))
		for j, sl := range chunk {
			queries[j] = sl.query
		}
		results := prov.fetch(ctx, t, queries)
		for j, res := range results {
			for _, i := range chunk[j].refs {
				out[i] = res
			}
		}
	}
}

type target struct {
	provider string
	host     string
	query    Query
}
