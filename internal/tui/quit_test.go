package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuitStopsEverySession(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 96, Height: 22})
	m, _ = step(t, m, key("esc"))
	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.quitting || cmd != nil {
		t.Fatal("one ctrl+c must not quit")
	}
	m, cmd = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.quitting || cmd == nil {
		t.Fatal("a second ctrl+c must start the shutdown")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if _, ok := msg.(shutdownDoneMsg); !ok {
			t.Fatalf("msg = %T", msg)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("the shutdown never finished")
	}
	for _, snap := range mgr.Snapshots() {
		if snap.State.Live() {
			t.Errorf("session %s is still live: %v", snap.Name, snap.State)
		}
	}
}

func TestCtrlCClearsThePromptBeforeItQuits(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.prompt.SetValue("a prompt I no longer want")
	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.prompt.Value() != "" {
		t.Fatalf("the prompt was not cleared, it holds %q", m.prompt.Value())
	}
	if m.quitting || cmd != nil {
		t.Fatal("the first ctrl+c must not quit")
	}
	if !strings.Contains(m.statusView(), "ctrl+c again") {
		t.Errorf("the status does not warn about the next press:\n%s", m.statusView())
	}

	m, cmd = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.quitting || cmd == nil {
		t.Fatal("the second ctrl+c must quit")
	}
}

func TestAnyOtherKeyDisarmsTheQuit(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.armedQuit {
		t.Fatal("the first ctrl+c must arm the quit")
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.armedQuit {
		t.Fatal("another key must disarm the quit")
	}
	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.quitting || cmd != nil {
		t.Fatal("ctrl+c after another key must clear, not quit")
	}
}

func TestCtrlCClosesTheFormAndTheConfirmation(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	if m.form == nil {
		t.Fatal("the form must be open")
	}
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.form != nil {
		t.Fatal("ctrl+c must close the form")
	}
	if m.quitting {
		t.Fatal("closing the form must not quit")
	}

	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = chord(t, m, "s", "x")
	if m.confirm == "" {
		t.Fatal("x must ask for a confirmation")
	}
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.confirm != "" {
		t.Fatal("ctrl+c must drop the confirmation")
	}
	if m.quitting {
		t.Fatal("dropping the confirmation must not quit")
	}
}

func TestQStillQuitsAtOnce(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	m, cmd := step(t, m, key("q"))
	if !m.quitting || cmd == nil {
		t.Fatal("q must quit with one press")
	}
}
