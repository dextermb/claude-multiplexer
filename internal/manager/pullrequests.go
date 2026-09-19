package manager

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/git"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/pullrequest"
)

// PRBadge is the pull request of a session, in the short form the output bar
// draws: the provider, the number, the state, and the unresolved count. See
// docs/pull-requests.md.
type PRBadge struct {
	Provider   string
	Number     int
	State      string
	Unresolved int
}

// pullRequests builds the provider set from the current settings file, so a
// change to the settings takes effect on the next call. See docs/pull-requests.md.
func (m *Manager) pullRequests() *pullrequest.Set {
	cfg, err := config.Load(config.Target(m.opts.ConfigPaths...))
	if err != nil {
		return pullrequest.NewSet(nil)
	}
	return pullrequest.NewSet(cfg.PullRequests)
}

// PullRequestsEnabled reports whether a provider has a usable transport.
func (m *Manager) PullRequestsEnabled() bool {
	return m.pullRequests().Enabled()
}

// ConfigurePullRequest writes one provider's token, mode, and endpoint into the
// settings file, so the feature turns on. An empty token, with mode cli, uses
// the provider CLI login. See docs/pull-requests.md.
func (m *Manager) ConfigurePullRequest(provider, token, mode, url string) (string, error) {
	if !config.ValidPullRequestProvider(provider) {
		return "", ErrUnknownPRProvider
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	entry := &config.PRProvider{Token: token, Mode: mode, URL: url}
	current.PullRequests = current.PullRequests.SetProvider(provider, entry)
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// PullRequestsFor reads the pull request of every code base of the calling
// session. It runs a live lookup, mirrors the result, and returns the list. A
// code base with no branch or no remote keeps its last known mirror. See
// docs/pull-requests.md.
func (m *Manager) PullRequestsFor(ctx context.Context, by string) ([]mcp.PullRequest, error) {
	meta, err := m.anyMeta(by)
	if err != nil {
		return nil, err
	}
	dirs := codebaseDirs(meta)
	if len(dirs) == 0 {
		return prViews(meta), nil
	}
	perDir, refs, idx := buildDirRefs(dirs)
	if len(refs) > 0 {
		results := m.pullRequests().Lookup(ctx, refs)
		for j, res := range results {
			perDir[idx[j]].Res = res
		}
	}
	m.applyPullRequests(by, perDir)
	updated, err := m.anyMeta(by)
	if err != nil {
		return nil, err
	}
	return prViews(updated), nil
}

// PullRequests reports the pull requests each live session tracks, keyed by
// session name. A session with no tracked PR is absent. See docs/pull-requests.md.
func (m *Manager) PullRequests() map[string][]PRBadge {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string][]PRBadge, len(m.entries))
	for name, item := range m.entries {
		if badges := PRBadges(item.metaCopy().PRs); len(badges) > 0 {
			out[name] = badges
		}
	}
	return out
}

// dirResult is one code base's lookup outcome. Keep marks a code base with no
// branch or no remote this sweep, so its prior mirror stays.
type dirResult struct {
	Dir    string
	Branch string
	Res    pullrequest.Result
	Keep   bool
}

// buildDirRefs reads the branch and remote of each code base, and returns the
// per-directory slots, the refs to look up, and the index from each ref back to
// its slot. A code base with no branch or no remote is marked Keep.
func buildDirRefs(dirs []string) ([]dirResult, []pullrequest.BranchRef, []int) {
	perDir := make([]dirResult, len(dirs))
	refs := make([]pullrequest.BranchRef, 0, len(dirs))
	idx := make([]int, 0, len(dirs))
	for i, dir := range dirs {
		branch := git.Branch(dir)
		remote := git.RemoteURL(dir, "origin")
		perDir[i] = dirResult{Dir: dir, Branch: branch}
		if branch == "" || remote == "" {
			perDir[i].Keep = true
			continue
		}
		refs = append(refs, pullrequest.BranchRef{RemoteURL: remote, Branch: branch})
		idx = append(idx, i)
	}
	return perDir, refs, idx
}

// applyPullRequests mirrors the per-directory results into a session's metadata.
// It keeps the last known mirror of a code base on a transient error or a
// missing branch, drops a code base with no PR, and writes only on a change.
func (m *Manager) applyPullRequests(name string, results []dirResult) {
	item, err := m.entry(name)
	if err != nil {
		return
	}
	cur := item.metaCopy()
	prev := make(map[string]PRMirror, len(cur.PRs))
	for _, pr := range cur.PRs {
		prev[pr.Dir] = pr
	}
	next := nextPRs(results, prev)
	if samePRs(cur.PRs, next) {
		return
	}
	_, _ = item.mutateMeta(func(meta *Meta) error {
		meta.PRs = next
		return nil
	})
}

func nextPRs(results []dirResult, prev map[string]PRMirror) []PRMirror {
	var out []PRMirror
	for _, r := range results {
		if r.Keep || r.Res.Err != nil {
			if p, ok := prev[r.Dir]; ok {
				out = append(out, p)
			}
			continue
		}
		if !r.Res.Found {
			continue
		}
		out = append(out, prMirror(r.Dir, r.Branch, r.Res.PR))
	}
	return out
}

func prMirror(dir, branch string, pr pullrequest.PR) PRMirror {
	return PRMirror{
		Dir:        dir,
		Branch:     branch,
		Provider:   pr.Provider,
		Number:     pr.Number,
		URL:        pr.URL,
		State:      pr.State,
		Title:      pr.Title,
		Unresolved: pr.Unresolved,
		SyncedAt:   time.Now(),
	}
}

// PRBadges is the short form of each mirrored pull request that has a number,
// for the interface. See docs/pull-requests.md.
func PRBadges(prs []PRMirror) []PRBadge {
	var out []PRBadge
	for _, pr := range prs {
		if pr.Number == 0 {
			continue
		}
		out = append(out, PRBadge{
			Provider:   pr.Provider,
			Number:     pr.Number,
			State:      pr.State,
			Unresolved: pr.Unresolved,
		})
	}
	return out
}

func prViews(meta Meta) []mcp.PullRequest {
	out := make([]mcp.PullRequest, 0, len(meta.PRs))
	for _, pr := range meta.PRs {
		out = append(out, mcp.PullRequest{
			Provider:   pr.Provider,
			Number:     pr.Number,
			URL:        pr.URL,
			State:      pr.State,
			Title:      pr.Title,
			Unresolved: pr.Unresolved,
			Branch:     pr.Branch,
			Dir:        pr.Dir,
			Found:      pr.Number != 0,
		})
	}
	return out
}

// codebaseDirs is the ordered set of code bases the session tracks: its project
// directories, or its one working directory when it has no project. It matches
// the set the diff panel groups by, in Projects. See docs/pull-requests.md.
func codebaseDirs(meta Meta) []string {
	if len(meta.WorkingDirs) > 0 {
		return meta.WorkingDirs
	}
	if dir := effectiveDir(meta); dir != "" {
		return []string{dir}
	}
	return nil
}

// effectiveDir is the directory a session works in: its worktree override, or
// its configured directory.
func effectiveDir(meta Meta) string {
	if dir := strings.TrimSpace(meta.WorkingDir); dir != "" {
		return dir
	}
	return strings.TrimSpace(meta.Dir)
}
