package manager

import (
	"context"
	"time"

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

// sweepPullRequests reads the branch and remote of every code base of every live
// session, then runs one batched lookup and mirrors each result. It reads the
// settings on each sweep, so a change takes effect on the next tick. One lookup
// covers the whole fleet, so a shared repository and branch cost one request.
// See docs/pull-requests.md.
func (m *Manager) sweepPullRequests(ctx context.Context) {
	set := m.pullRequests()
	if !set.Enabled() {
		return
	}

	type job struct {
		name string
		dirs []string
	}
	m.mu.Lock()
	jobs := make([]job, 0, len(m.entries))
	for name, item := range m.entries {
		if dirs := codebaseDirs(item.metaCopy()); len(dirs) > 0 {
			jobs = append(jobs, job{name: name, dirs: dirs})
		}
	}
	m.mu.Unlock()
	if len(jobs) == 0 {
		return
	}

	perJob := make([][]dirResult, len(jobs))
	var refs []pullrequest.BranchRef
	type at struct{ job, dir int }
	var index []at
	for ji, j := range jobs {
		perDir, jobRefs, jobIdx := buildDirRefs(j.dirs)
		perJob[ji] = perDir
		for _, di := range jobIdx {
			index = append(index, at{job: ji, dir: di})
		}
		refs = append(refs, jobRefs...)
	}

	if len(refs) > 0 {
		results := set.Lookup(ctx, refs)
		for k, res := range results {
			a := index[k]
			perJob[a.job][a.dir].Res = res
		}
	}
	for ji, j := range jobs {
		m.applyPullRequests(j.name, perJob[ji])
	}
}
