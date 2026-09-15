package tui

import (
	"testing"
)

func TestTheTotalCountsAStoredSessionAndAnArchivedOne(t *testing.T) {
	m, mgr := newTestModel(t, "")
	dir := t.TempDir()
	storeSession(t, mgr, "kept", dir, "hello")
	storeSession(t, mgr, "filed", dir, "hello")
	if err := mgr.Archive("filed", true); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	m = start(t, m, 100, 24)
	m, _ = step(t, m, storedMsg{metas: mgr.Stored()})

	if got := m.totalCost(); got != 0.5 {
		t.Fatalf("total = %v, want the two stored sessions summed to 0.5", got)
	}
}

func TestTheTotalIgnoresTheSearchFilter(t *testing.T) {
	m, mgr := newTestModel(t, "")
	dir := t.TempDir()
	storeSession(t, mgr, "kept", dir, "hello")
	storeSession(t, mgr, "other", dir, "hello")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, storedMsg{metas: mgr.Stored()})

	before := m.totalCost()
	m.search.SetValue("kept")
	m.refresh()
	if len(m.rows) != 1 {
		t.Fatalf("rows = %d, want the search to keep one", len(m.rows))
	}
	if got := m.totalCost(); got != before {
		t.Fatalf("total = %v with a search, want %v", got, before)
	}
}

func TestTheTotalCountsALiveSessionOnce(t *testing.T) {
	m, mgr := newTestModel(t, "")
	storeSession(t, mgr, "back", t.TempDir(), "hello")
	stored := mgr.Stored()
	if len(stored) != 1 {
		t.Fatalf("stored = %+v, want one", stored)
	}
	if _, err := mgr.Resume(t.Context(), stored[0]); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	m = start(t, m, 100, 24)
	m, _ = step(t, m, storedMsg{metas: mgr.Stored()})

	if got := m.totalCost(); got != 0.25 {
		t.Fatalf("total = %v, want the one session counted once at 0.25", got)
	}
}
