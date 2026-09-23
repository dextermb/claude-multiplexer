package manager

import (
	"context"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

// workItemPollTick is how often the manager reads the status of every linked
// session's work item. One sweep groups the sessions by item, so a shared item
// costs one request. See docs/work-items.md.
const workItemPollTick = time.Minute

// StartWorkItemPoll starts the clock that mirrors the work item status of each
// linked session. Call it once, after StartPullRequestPoll. See
// docs/work-items.md.
func (m *Manager) StartWorkItemPoll() {
	if m.wiStop != nil {
		return
	}
	m.wiStop = make(chan struct{})
	m.wiWG.Add(1)
	go m.workItemPollLoop()
}

func (m *Manager) workItemPollLoop() {
	defer m.wiWG.Done()
	ticker := time.NewTicker(workItemPollTick)
	defer ticker.Stop()
	m.sweepWorkItems(context.Background())
	for {
		select {
		case <-m.wiStop:
			return
		case <-ticker.C:
			m.sweepWorkItems(context.Background())
		}
	}
}

// workItemLink is one distinct work item that live sessions link to.
type workItemLink struct {
	provider string
	key      string
}

// sweepWorkItems reads the status of every live session's linked work item, then
// mirrors each result. It reads the settings on each sweep, so a change takes
// effect on the next tick. It groups the sessions by item, so two sessions on
// one item cost one request, not two. A transient error keeps the last known
// status. See docs/work-items.md.
func (m *Manager) sweepWorkItems(ctx context.Context) {
	set := m.workItems()
	if !set.Enabled() {
		return
	}
	order, names := m.workItemGroups()
	for _, link := range order {
		item, err := set.Resolve(ctx, link.provider, link.key)
		if err != nil {
			continue
		}
		for _, name := range names[link] {
			m.applyWorkItemStatus(name, item)
		}
	}
}

// workItemGroups groups the live linked sessions by their work item. It returns
// the distinct items in a stable discovery order, and the session names under
// each item, so one Resolve serves every session on that item.
func (m *Manager) workItemGroups() ([]workItemLink, map[workItemLink][]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make(map[workItemLink][]string)
	var order []workItemLink
	for name, item := range m.entries {
		meta := item.metaCopy()
		if meta.WorkItemKey == "" {
			continue
		}
		link := workItemLink{provider: meta.WorkItemProvider, key: meta.WorkItemKey}
		if _, ok := names[link]; !ok {
			order = append(order, link)
		}
		names[link] = append(names[link], name)
	}
	return order, names
}

// applyWorkItemStatus mirrors a resolved item's status into a session's
// metadata. It writes only when the status changed, and it leaves the link and
// the title alone, because a poll must not rename the session. It skips a
// session that stopped or relinked during the sweep. See docs/work-items.md.
func (m *Manager) applyWorkItemStatus(name string, item workitem.Item) {
	entry, err := m.entry(name)
	if err != nil {
		return
	}
	cur := entry.metaCopy()
	if cur.WorkItemKey != item.Key || cur.WorkItemProvider != item.Provider {
		return
	}
	if cur.WorkItemStatus == item.Status && cur.WorkItemStatusID == item.StatusID {
		return
	}
	_, _ = entry.mutateMeta(func(meta *Meta) error {
		meta.WorkItemStatus = item.Status
		meta.WorkItemStatusID = item.StatusID
		meta.WorkItemURL = item.URL
		meta.WorkItemSyncedAt = time.Now()
		return nil
	})
}
