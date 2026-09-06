package tui

import "testing"

func TestTheSendRecordsThePromptInHistory(t *testing.T) {
	m, _ := busyModel(t)
	if len(m.history) != 1 || m.history[0] != "a long task" {
		t.Fatalf("history = %v, want the sent prompt", m.history)
	}
	if m.histIdx != len(m.history) {
		t.Fatalf("histIdx = %d, want %d after a send", m.histIdx, len(m.history))
	}
}

func TestTheArrowKeysWalkThePromptHistory(t *testing.T) {
	m, _ := busyModel(t)
	m.history = []string{"first", "second"}
	m.histIdx = len(m.history)
	m.histDraft = ""
	m.prompt.SetValue("draft")

	steps := []struct {
		press string
		want  string
	}{
		{"up", "second"},
		{"up", "first"},
		{"up", "first"},
		{"down", "second"},
		{"down", "draft"},
		{"down", "draft"},
	}
	for i, s := range steps {
		m, _ = step(t, m, key(s.press))
		if got := m.prompt.Value(); got != s.want {
			t.Fatalf("step %d (%s): prompt = %q, want %q", i, s.press, got, s.want)
		}
	}
}

func TestAnArrowInAMultiRowPromptMovesTheCursor(t *testing.T) {
	m, _ := busyModel(t)
	m.history = []string{"first"}
	m.histIdx = len(m.history)
	m.prompt.SetValue("line1\nline2")

	m, _ = step(t, m, key("up"))
	if got := m.prompt.Value(); got != "line1\nline2" {
		t.Fatalf("prompt = %q, want the draft left alone", got)
	}
	if m.histIdx != len(m.history) {
		t.Fatalf("histIdx = %d, want %d — the cursor moved, not the history", m.histIdx, len(m.history))
	}
}

func TestRecordHistoryDropsAConsecutiveDuplicate(t *testing.T) {
	var m Model
	m.recordHistory("same")
	m.recordHistory("same")
	m.recordHistory("other")
	m.recordHistory("same")
	want := []string{"same", "other", "same"}
	if len(m.history) != len(want) {
		t.Fatalf("history = %v, want %v", m.history, want)
	}
	for i := range want {
		if m.history[i] != want[i] {
			t.Fatalf("history = %v, want %v", m.history, want)
		}
	}
}
