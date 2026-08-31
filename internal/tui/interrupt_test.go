package tui

import (
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func waitBusy(t *testing.T, mgr *manager.Manager, name string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if snap, err := mgr.Snapshot(name); err == nil && snap.State == session.StateBusy {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q to go busy", name)
}

func busyModel(t *testing.T) (Model, *manager.Manager) {
	t.Helper()
	t.Setenv("FAKECLAUDE_MODE", "interruptible")
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.SetValue("a long task")
	m, _ = step(t, m, key("enter"))
	waitBusy(t, mgr, "alpha")
	m.refresh()
	return m, mgr
}

func TestEnterFlushesTheQueuedPromptWhileBusy(t *testing.T) {
	m, _ := busyModel(t)
	if len(m.queued["alpha"]) != 1 {
		t.Fatalf("the sent prompt was not held: %v", m.queued["alpha"])
	}
	m.prompt.Reset()
	next, cmd := step(t, m, key("enter"))
	nm := next
	if cmd == nil {
		t.Fatal("empty enter while busy must return an interrupt command")
	}
	if nm.status != "sending now…" {
		t.Fatalf("status = %q, want the send-now hint", nm.status)
	}
	if len(nm.queued["alpha"]) != 1 {
		t.Fatalf("the queued prompt must stay to flush, got %v", nm.queued["alpha"])
	}
}

func TestEmptyEnterWhileBusyWithNothingQueuedDoesNothing(t *testing.T) {
	m, _ := busyModel(t)
	m.dropQueued("alpha")
	m.prompt.Reset()
	next, cmd := step(t, m, key("enter"))
	nm := next
	if cmd != nil {
		t.Fatal("empty enter with nothing queued must not interrupt")
	}
	if nm.status == "sending now…" {
		t.Fatal("the send-now hint must not show with an empty queue")
	}
}

func TestEscStopsAndDropsTheQueueWhileBusy(t *testing.T) {
	m, _ := busyModel(t)
	next, cmd := step(t, m, key("esc"))
	nm := next
	if cmd == nil {
		t.Fatal("esc while busy must return an interrupt command")
	}
	if nm.status != "interrupted" {
		t.Fatalf("status = %q, want interrupted", nm.status)
	}
	if len(nm.queued["alpha"]) != 0 {
		t.Fatalf("esc must drop the queue, got %v", nm.queued["alpha"])
	}
	if nm.focus != focusPrompt {
		t.Fatalf("esc while busy must keep the prompt focus, got %v", nm.focus)
	}
}

func TestEscBlursWhenTheSessionIsIdle(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusPrompt
	m.prompt.Focus()

	next, _ := step(t, m, key("esc"))
	nm := next
	if nm.focus != focusSidebar {
		t.Fatalf("esc on an idle session must blur to the sidebar, got %v", nm.focus)
	}
}
