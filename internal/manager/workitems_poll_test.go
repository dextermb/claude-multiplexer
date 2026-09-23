package manager

import (
	"context"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

func spawnLinked(t *testing.T, m *Manager, name, key string) *entry {
	t.Helper()
	if _, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: name}); err != nil {
		t.Fatalf("Spawn %s: %v", name, err)
	}
	item, err := m.entry(name)
	if err != nil {
		t.Fatalf("entry %s: %v", name, err)
	}
	if _, err := item.mutateMeta(func(meta *Meta) error {
		applyWorkItem(meta, workitem.Item{Provider: "linear", Key: key, Status: "In Review"})
		return nil
	}); err != nil {
		t.Fatalf("mutateMeta %s: %v", name, err)
	}
	return item
}

func TestWorkItemGroupsGroupsSharedItems(t *testing.T) {
	m := newBridgeManager(t)
	spawnLinked(t, m, "a", "LIN-1")
	spawnLinked(t, m, "b", "LIN-1")
	spawnLinked(t, m, "c", "LIN-2")
	if _, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: "d"}); err != nil {
		t.Fatalf("Spawn d: %v", err)
	}

	order, names := m.workItemGroups()
	if len(order) != 2 {
		t.Fatalf("distinct items = %d, want 2 (one per key)", len(order))
	}
	one := workItemLink{provider: "linear", key: "LIN-1"}
	if got := names[one]; len(got) != 2 {
		t.Fatalf("LIN-1 sessions = %v, want two", got)
	}
	two := workItemLink{provider: "linear", key: "LIN-2"}
	if got := names[two]; len(got) != 1 || got[0] != "c" {
		t.Fatalf("LIN-2 sessions = %v, want [c]", got)
	}
}

func TestApplyWorkItemStatusWritesOnlyOnChange(t *testing.T) {
	m := newBridgeManager(t)
	item := spawnLinked(t, m, "a", "LIN-1")

	before := item.metaCopy().WorkItemSyncedAt
	m.applyWorkItemStatus("a", workitem.Item{Provider: "linear", Key: "LIN-1", Status: "In Review"})
	if got := item.metaCopy().WorkItemSyncedAt; !got.Equal(before) {
		t.Fatal("an unchanged status must not restamp the sync time")
	}

	m.applyWorkItemStatus("a", workitem.Item{Provider: "linear", Key: "LIN-1", Status: "Done", StatusID: "s9"})
	got := item.metaCopy()
	if got.WorkItemStatus != "Done" || got.WorkItemStatusID != "s9" {
		t.Fatalf("changed status not mirrored: %+v", got)
	}
	if !got.WorkItemSyncedAt.After(before) {
		t.Fatal("a changed status must restamp the sync time")
	}
}

func TestApplyWorkItemStatusSkipsWhenLinkChanged(t *testing.T) {
	m := newBridgeManager(t)
	item := spawnLinked(t, m, "a", "LIN-1")

	m.applyWorkItemStatus("a", workitem.Item{Provider: "linear", Key: "LIN-2", Status: "Done"})
	if got := item.metaCopy(); got.WorkItemKey != "LIN-1" || got.WorkItemStatus != "In Review" {
		t.Fatalf("a status for another key must not land: %+v", got)
	}
}

func TestApplyWorkItemStatusIgnoresUnknownSession(t *testing.T) {
	m := newBridgeManager(t)
	m.applyWorkItemStatus("gone", workitem.Item{Provider: "linear", Key: "LIN-1", Status: "Done"})
}

func TestApplyWorkItemStatusDoesNotRename(t *testing.T) {
	m := newBridgeManager(t)
	item := spawnLinked(t, m, "a", "LIN-1")
	if err := m.SetTitle("a", "LIN-1"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	m.applyWorkItemStatus("a", workitem.Item{Provider: "linear", Key: "LIN-1", Status: "Done"})
	if title := item.sess.Snapshot().Title; title != "LIN-1" {
		t.Fatalf("title = %q, a poll must not rename the session", title)
	}
}
