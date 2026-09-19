package manager

import (
	"context"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

func TestApplyAndClearWorkItem(t *testing.T) {
	var meta Meta
	applyWorkItem(&meta, workitem.Item{
		Provider: "linear",
		Key:      "LIN-1",
		URL:      "https://linear.app/acme/issue/LIN-1",
		Status:   "In Review",
		StatusID: "s3",
	})
	if meta.WorkItemKey != "LIN-1" || meta.WorkItemStatus != "In Review" {
		t.Fatalf("apply did not mirror: %+v", meta)
	}
	if meta.WorkItemSyncedAt.IsZero() {
		t.Fatal("apply did not stamp the sync time")
	}

	view := workItemView(meta)
	if !view.Linked || view.Provider != "linear" || view.Status != "In Review" {
		t.Fatalf("view = %+v", view)
	}

	clearWorkItem(&meta)
	if meta.WorkItemKey != "" || meta.WorkItemStatus != "" || !meta.WorkItemSyncedAt.IsZero() {
		t.Fatalf("clear left data: %+v", meta)
	}
	if workItemView(meta).Linked {
		t.Fatal("a cleared item must not read as linked")
	}
}

// linkForTest mirrors a work item onto a spawned session and renames it, the way
// SetWorkItem does, but without a live provider.
func linkForTest(t *testing.T, m *Manager, name, key string) *entry {
	t.Helper()
	item, err := m.entry(name)
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	if _, err := item.mutateMeta(func(meta *Meta) error {
		applyWorkItem(meta, workitem.Item{Provider: "linear", Key: key, Status: "In Review"})
		return nil
	}); err != nil {
		t.Fatalf("mutateMeta: %v", err)
	}
	if err := m.SetTitle(name, key); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	return item
}

func TestUnsetWorkItemRevertsTheRename(t *testing.T) {
	m := newBridgeManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: "s"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	item := linkForTest(t, m, name, "GIM-1")

	changed, err := m.UnsetWorkItem(name)
	if err != nil || !changed {
		t.Fatalf("UnsetWorkItem: changed=%v err=%v", changed, err)
	}
	if title := item.sess.Snapshot().Title; title != "" {
		t.Fatalf("title = %q, want the rename cleared", title)
	}
	if item.metaCopy().WorkItemKey != "" {
		t.Fatal("the link must be cleared")
	}
}

func TestUnsetWorkItemKeepsAManualTitle(t *testing.T) {
	m := newBridgeManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: "s"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	linkForTest(t, m, name, "GIM-1")
	if err := m.SetTitle(name, "my own title"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}

	if _, err := m.UnsetWorkItem(name); err != nil {
		t.Fatalf("UnsetWorkItem: %v", err)
	}
	item, _ := m.entry(name)
	if title := item.sess.Snapshot().Title; title != "my own title" {
		t.Fatalf("title = %q, want the manual title kept", title)
	}
}
