package manager

import (
	"context"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func (b *bridge) WorkItemsEnabled() bool { return b.m.WorkItemsEnabled() }

func (b *bridge) WorkItemProviders() []string { return b.m.WorkItemProviders() }

func (b *bridge) ConfigureWorkItem(provider, token, email, url, by string) (string, error) {
	path, err := b.m.ConfigureWorkItem(provider, token, email, url)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" configured the "+provider+" work-item provider", true)
	return path, nil
}

func (b *bridge) WorkItem(session string) (mcp.WorkItem, error) { return b.m.WorkItem(session) }

func (b *bridge) SetWorkItem(ctx context.Context, provider, key, by string) (mcp.WorkItem, error) {
	item, err := b.m.SetWorkItem(ctx, by, provider, key)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	b.m.notify(by, by+" links to "+item.Key+" ("+item.Status+")", true)
	return item, nil
}

func (b *bridge) UnsetWorkItem(by string) (bool, error) {
	changed, err := b.m.UnsetWorkItem(by)
	if err != nil {
		return false, err
	}
	if changed {
		b.m.notify(by, by+" no longer links to a work item", true)
	}
	return changed, nil
}

func (b *bridge) WorkItemStatuses(ctx context.Context, by string) ([]mcp.WorkItemStatus, error) {
	return b.m.WorkItemStatuses(ctx, by)
}

func (b *bridge) SetWorkItemStatus(ctx context.Context, target, by string) (mcp.WorkItem, error) {
	item, err := b.m.SetWorkItemStatus(ctx, by, target)
	if err != nil {
		return mcp.WorkItem{}, err
	}
	b.m.notify(by, item.Key+" is now "+item.Status, true)
	return item, nil
}
