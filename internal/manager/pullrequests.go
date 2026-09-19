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

// PullRequest reads the pull request of the calling session's branch. It runs a
// live lookup for the one session, mirrors the result, and returns it. With no
// branch or no remote, it returns the last known mirror. See docs/pull-requests.md.
func (m *Manager) PullRequest(ctx context.Context, by string) (mcp.PullRequest, error) {
	meta, err := m.anyMeta(by)
	if err != nil {
		return mcp.PullRequest{}, err
	}
	dir := effectiveDir(meta)
	branch := git.Branch(dir)
	remote := git.RemoteURL(dir, "origin")
	if branch == "" || remote == "" {
		return prView(meta), nil
	}
	results := m.pullRequests().Lookup(ctx, []pullrequest.BranchRef{{RemoteURL: remote, Branch: branch}})
	if len(results) != 1 || results[0].Err != nil {
		return prView(meta), nil
	}
	m.applyPullRequest(by, branch, results[0])
	if results[0].Found {
		return prItemView(results[0].PR, branch), nil
	}
	return mcp.PullRequest{Branch: branch, Found: false}, nil
}

// PullRequests reports the pull request each live session tracks, keyed by
// session name. A session with no tracked PR is absent. See docs/pull-requests.md.
func (m *Manager) PullRequests() map[string]PRBadge {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]PRBadge, len(m.entries))
	for name, item := range m.entries {
		meta := item.metaCopy()
		if meta.PRNumber == 0 {
			continue
		}
		out[name] = PRBadge{
			Provider:   meta.PRProvider,
			Number:     meta.PRNumber,
			State:      meta.PRState,
			Unresolved: meta.PRUnresolved,
		}
	}
	return out
}

// applyPullRequest mirrors one lookup result into a session's metadata. It keeps
// the last known mirror on a transient error, and writes only on a change.
func (m *Manager) applyPullRequest(name, branch string, res pullrequest.Result) {
	if res.Err != nil {
		return
	}
	item, err := m.entry(name)
	if err != nil {
		return
	}
	cur := item.metaCopy()
	next := cur
	if res.Found {
		applyPR(&next, res.PR, branch)
	} else {
		clearPR(&next)
	}
	if prSame(cur, next) {
		return
	}
	_, _ = item.mutateMeta(func(meta *Meta) error {
		if res.Found {
			applyPR(meta, res.PR, branch)
		} else {
			clearPR(meta)
		}
		return nil
	})
}

func applyPR(meta *Meta, pr pullrequest.PR, branch string) {
	meta.PRProvider = pr.Provider
	meta.PRNumber = pr.Number
	meta.PRURL = pr.URL
	meta.PRState = pr.State
	meta.PRTitle = pr.Title
	meta.PRUnresolved = pr.Unresolved
	meta.PRBranch = branch
	meta.PRSyncedAt = time.Now()
}

func clearPR(meta *Meta) {
	meta.PRProvider = ""
	meta.PRNumber = 0
	meta.PRURL = ""
	meta.PRState = ""
	meta.PRTitle = ""
	meta.PRUnresolved = 0
	meta.PRBranch = ""
	meta.PRSyncedAt = time.Time{}
}

func prSame(a, b Meta) bool {
	return a.PRProvider == b.PRProvider &&
		a.PRNumber == b.PRNumber &&
		a.PRState == b.PRState &&
		a.PRUnresolved == b.PRUnresolved &&
		a.PRBranch == b.PRBranch
}

func prView(meta Meta) mcp.PullRequest {
	return mcp.PullRequest{
		Provider:   meta.PRProvider,
		Number:     meta.PRNumber,
		URL:        meta.PRURL,
		State:      meta.PRState,
		Title:      meta.PRTitle,
		Unresolved: meta.PRUnresolved,
		Branch:     meta.PRBranch,
		Found:      meta.PRNumber != 0,
	}
}

func prItemView(pr pullrequest.PR, branch string) mcp.PullRequest {
	return mcp.PullRequest{
		Provider:   pr.Provider,
		Number:     pr.Number,
		URL:        pr.URL,
		State:      pr.State,
		Title:      pr.Title,
		Unresolved: pr.Unresolved,
		Branch:     branch,
		Found:      true,
	}
}

// effectiveDir is the directory a session works in: its worktree override, or
// its configured directory.
func effectiveDir(meta Meta) string {
	if dir := strings.TrimSpace(meta.WorkingDir); dir != "" {
		return dir
	}
	return strings.TrimSpace(meta.Dir)
}
