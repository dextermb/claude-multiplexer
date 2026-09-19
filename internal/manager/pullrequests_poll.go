package manager

import (
	"context"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/git"
	"github.com/dextermb/claude-multiplexer/internal/pullrequest"
)

// pullRequestPollTick is how often the manager reads the pull request of every
// live session. One sweep batches the branches into a few requests. See
// docs/pull-requests.md.
const pullRequestPollTick = time.Minute

// StartPullRequestPoll starts the clock that mirrors the pull request of each
// live session. Call it once, after StartConfigWatch. See docs/pull-requests.md.
func (m *Manager) StartPullRequestPoll() {
	if m.prStop != nil {
		return
	}
	m.prStop = make(chan struct{})
	m.prWG.Add(1)
	go m.prPollLoop()
}

func (m *Manager) prPollLoop() {
	defer m.prWG.Done()
	ticker := time.NewTicker(pullRequestPollTick)
	defer ticker.Stop()
	m.sweepPullRequests(context.Background())
	for {
		select {
		case <-m.prStop:
			return
		case <-ticker.C:
			m.sweepPullRequests(context.Background())
		}
	}
}

// sweepPullRequests reads the branch and remote of every live session, then runs
// one batched lookup and mirrors each result. It reads the settings on each
// sweep, so a change takes effect on the next tick. See docs/pull-requests.md.
func (m *Manager) sweepPullRequests(ctx context.Context) {
	set := m.pullRequests()
	if !set.Enabled() {
		return
	}

	type job struct {
		name string
		dir  string
	}
	m.mu.Lock()
	jobs := make([]job, 0, len(m.entries))
	for name, item := range m.entries {
		if dir := effectiveDir(item.metaCopy()); dir != "" {
			jobs = append(jobs, job{name: name, dir: dir})
		}
	}
	m.mu.Unlock()
	if len(jobs) == 0 {
		return
	}

	refs := make([]pullrequest.BranchRef, len(jobs))
	branches := make([]string, len(jobs))
	for i, j := range jobs {
		branches[i] = git.Branch(j.dir)
		refs[i] = pullrequest.BranchRef{RemoteURL: git.RemoteURL(j.dir, "origin"), Branch: branches[i]}
	}

	results := set.Lookup(ctx, refs)
	for i, res := range results {
		m.applyPullRequest(jobs[i].name, branches[i], res)
	}
}
