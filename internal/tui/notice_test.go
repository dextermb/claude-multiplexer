package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/manager"
)

func TestANoticeKeepsTheStreamingText(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.lastSeq = 5
	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     6,
		Session: m.sel,
		Partial: "half a thought",
	}))
	if m.partials[m.sel] != "half a thought" {
		t.Fatalf("partial = %q", m.partials[m.sel])
	}
	name := m.sel

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     7,
		Session: "landing",
		Notice:  "alpha archived landing",
		Reload:  true,
	}))
	if m.partials[name] != "half a thought" {
		t.Fatalf("the notice cleared the streaming text: %q", m.partials[name])
	}
	if m.status != "alpha archived landing" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestANoticeAsksForTheStoredListAgain(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))

	m.lastSeq = 1
	_, cmd := step(t, m, eventMsg(manager.Event{
		Seq:     2,
		Session: "landing",
		Notice:  "docs archived landing",
		Reload:  true,
	}))
	if !yieldsStored(cmd) {
		t.Fatal("a notice with Reload did not read the stored list again")
	}
}

// yieldsStored runs a command tree and reports whether any branch of it asks
// the manager for the stored list. Each command runs with a deadline, because
// the batch also holds the command that waits on the bus for ever.
func yieldsStored(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		switch msg := msg.(type) {
		case storedMsg:
			return true
		case tea.BatchMsg:
			for _, inner := range msg {
				if yieldsStored(inner) {
					return true
				}
			}
		}
	case <-time.After(500 * time.Millisecond):
	}
	return false
}
