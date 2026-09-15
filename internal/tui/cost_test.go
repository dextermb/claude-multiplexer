package tui

import (
	"strings"
	"testing"
)

func TestTheStatusBarNamesTheWindow(t *testing.T) {
	m := Model{cost: 12.3456, costWindow: "1d"}
	if got := m.costSeg(); got != "$12.3456 1d" {
		t.Fatalf("costSeg = %q, want the window named", got)
	}
}

func TestTheStatusBarNamesNoWindowForTheWholeHistory(t *testing.T) {
	m := Model{cost: 12.3456}
	if got := m.costSeg(); got != "$12.3456" {
		t.Fatalf("costSeg = %q, want no suffix", got)
	}
}

func TestTheStatusBarDrawsTheWindowedTotal(t *testing.T) {
	m, mgr := newTestModel(t, "")
	storeSession(t, mgr, "kept", t.TempDir(), "hello")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, reloadStored(mgr)())

	if m.costWindow != "1d" {
		t.Fatalf("window = %q, want the 1d default", m.costWindow)
	}
	if m.cost != 0.25 {
		t.Fatalf("cost = %v, want the one turn of today", m.cost)
	}
	if view := m.View(); !strings.Contains(view, "$0.2500 1d") {
		t.Fatalf("the status bar does not draw the windowed total:\n%s", view)
	}
}
