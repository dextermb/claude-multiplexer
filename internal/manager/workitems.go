package manager

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

// workItems builds the provider set from the current settings file, so a change
// to the settings takes effect on the next call. See docs/work-items.md.
func (m *Manager) workItems() *workitem.Set {
	cfg, err := config.Load(config.Target(m.opts.ConfigPaths...))
	if err != nil {
		return workitem.NewSet(nil)
	}
	return workitem.NewSet(cfg.WorkItems)
}

// WorkItemsEnabled reports whether a provider is configured.
func (m *Manager) WorkItemsEnabled() bool {
	return m.workItems().Enabled()
}

// ConfigureWorkItem writes one provider's token into the settings file, so the
// feature turns on. An email selects Jira Basic auth, and a url overrides the
// default endpoint. See docs/work-items.md.
func (m *Manager) ConfigureWorkItem(provider, token, email, url string) (string, error) {
	if !config.ValidWorkItemProvider(provider) {
		return "", ErrUnknownProvider
	}
	if strings.TrimSpace(token) == "" {
		return "", ErrNoWorkItemToken
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	entry := &config.WorkItemProvider{Token: token, Email: email, URL: url}
	current.WorkItems = current.WorkItems.SetProvider(provider, entry)
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// WorkItemProviders names the configured providers.
func (m *Manager) WorkItemProviders() []string {
	return m.workItems().Providers()
}

// WorkItem reads the work item a session links to, live or stopped.
func (m *Manager) WorkItem(name string) (mcp.WorkItem, error) {
	meta, err := m.anyMeta(name)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	return workItemView(meta), nil
}

// SetWorkItem links the calling session to an item. It resolves the item on the
// provider, then mirrors the title, url, and status into the metadata. See
// docs/work-items.md.
func (m *Manager) SetWorkItem(ctx context.Context, by, provider, key string) (mcp.WorkItem, error) {
	item, err := m.entry(by)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	resolved, err := m.workItems().Resolve(ctx, provider, key)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	if _, err := item.mutateMeta(func(meta *Meta) error {
		applyWorkItem(meta, resolved)
		return nil
	}); err != nil {
		return mcp.WorkItem{}, err
	}
	// Rename the session to the item, so its key names the row; see docs/work-items.md.
	_ = m.SetTitle(by, resolved.Key)
	return itemView(resolved), nil
}

// UnsetWorkItem clears the link of the calling session, and reports whether it
// held one. When the title still names the item, it clears the title too, so the
// rename does not outlive the link. See docs/work-items.md.
func (m *Manager) UnsetWorkItem(by string) (bool, error) {
	item, err := m.entry(by)
	if err != nil {
		return false, err
	}
	key := item.metaCopy().WorkItemKey
	if key == "" {
		return false, nil
	}
	if _, err := item.mutateMeta(func(meta *Meta) error {
		clearWorkItem(meta)
		return nil
	}); err != nil {
		return false, err
	}
	if item.sess.Snapshot().Title == key {
		_ = m.SetTitle(by, "")
	}
	return true, nil
}

// WorkItemStatuses reads the statuses the linked item may move to.
func (m *Manager) WorkItemStatuses(ctx context.Context, by string) ([]mcp.WorkItemStatus, error) {
	meta, err := m.anyMeta(by)
	if err != nil {
		return nil, err
	}
	if meta.WorkItemKey == "" {
		return nil, ErrNoWorkItem
	}
	statuses, err := m.workItems().Statuses(ctx, meta.WorkItemProvider, meta.WorkItemKey)
	if err != nil {
		return nil, err
	}
	out := make([]mcp.WorkItemStatus, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, mcp.WorkItemStatus{ID: s.ID, Name: s.Name})
	}
	return out, nil
}

// SetWorkItemStatus moves the linked item to the target status, then mirrors the
// new status into the metadata. See docs/work-items.md.
func (m *Manager) SetWorkItemStatus(ctx context.Context, by, target string) (mcp.WorkItem, error) {
	item, err := m.entry(by)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	meta := item.metaCopy()
	if meta.WorkItemKey == "" {
		return mcp.WorkItem{}, ErrNoWorkItem
	}
	updated, err := m.workItems().SetStatus(ctx, meta.WorkItemProvider, meta.WorkItemKey, target)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	if _, err := item.mutateMeta(func(meta *Meta) error {
		applyWorkItem(meta, updated)
		return nil
	}); err != nil {
		return mcp.WorkItem{}, err
	}
	return itemView(updated), nil
}

func applyWorkItem(meta *Meta, item workitem.Item) {
	meta.WorkItemProvider = item.Provider
	meta.WorkItemKey = item.Key
	meta.WorkItemURL = item.URL
	meta.WorkItemStatus = item.Status
	meta.WorkItemStatusID = item.StatusID
	meta.WorkItemSyncedAt = time.Now()
}

func clearWorkItem(meta *Meta) {
	meta.WorkItemProvider = ""
	meta.WorkItemKey = ""
	meta.WorkItemURL = ""
	meta.WorkItemStatus = ""
	meta.WorkItemStatusID = ""
	meta.WorkItemSyncedAt = time.Time{}
}

func workItemView(meta Meta) mcp.WorkItem {
	return mcp.WorkItem{
		Provider: meta.WorkItemProvider,
		Key:      meta.WorkItemKey,
		URL:      meta.WorkItemURL,
		Status:   meta.WorkItemStatus,
		StatusID: meta.WorkItemStatusID,
		Linked:   meta.WorkItemKey != "",
	}
}

func itemView(item workitem.Item) mcp.WorkItem {
	return mcp.WorkItem{
		Provider: item.Provider,
		Key:      item.Key,
		Title:    item.Title,
		URL:      item.URL,
		Status:   item.Status,
		StatusID: item.StatusID,
		Linked:   item.Key != "",
	}
}
