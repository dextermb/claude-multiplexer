package tui

import (
	"strings"
	"testing"
)

func TestARebuildDoesNotDrawAWaitingEventTwice(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 120, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	for len(m.sub.C) > 0 {
		m, _ = step(t, m, eventMsg(<-m.sub.C))
	}

	if _, err := mgr.SendFrom("alpha", "boss", "hello"); err != nil {
		t.Fatalf("SendFrom: %v", err)
	}
	m.rebuildOutput()
	for len(m.sub.C) > 0 {
		m, _ = step(t, m, eventMsg(<-m.sub.C))
	}

	if n := strings.Count(m.outputText, "prompt from boss"); n != 1 {
		t.Fatalf("the pane draws the line %d times:\n%s", n, m.outputText)
	}
}
