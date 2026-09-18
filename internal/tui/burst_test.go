package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTheFirstKeyIsNeverABurstKey(t *testing.T) {
	var b burst
	if b.key(time.Unix(0, 0)) {
		t.Fatal("the first key was marked as a burst key")
	}
}

func TestTwoCloseKeysMarkABurst(t *testing.T) {
	var b burst
	at := time.Unix(0, 0)
	b.key(at)
	if !b.key(at.Add(time.Millisecond)) {
		t.Fatal("a key 1ms after the last was not marked as a burst key")
	}
}

func TestTwoDistantKeysDoNotMarkABurst(t *testing.T) {
	var b burst
	at := time.Unix(0, 0)
	b.key(at)
	if b.key(at.Add(100 * time.Millisecond)) {
		t.Fatal("a key 100ms after the last was marked as a burst key")
	}
}

func burstModel(t *testing.T) Model {
	t.Helper()
	m, _ := newTestModel(t, "")
	m.burstAware = true
	m.focus = focusPrompt
	m.prompt.Focus()
	return m
}

func TestAPastedEnterInsertsANewline(t *testing.T) {
	m := burstModel(t)
	for _, msg := range []tea.KeyMsg{key("a"), key("enter"), key("b")} {
		m, _ = step(t, m, msg)
	}
	if got := m.prompt.Value(); got != "a\nb" {
		t.Fatalf("prompt holds %q, want %q", got, "a\nb")
	}
	if m.errText != "" {
		t.Fatalf("the prompt was sent: %q", m.errText)
	}
}

func TestATypedEnterStillSends(t *testing.T) {
	m := burstModel(t)
	m, _ = step(t, m, key("a"))
	time.Sleep(2 * burstGap)
	m, _ = step(t, m, key("enter"))
	if m.errText != "no session is selected" {
		t.Fatalf("the prompt was not sent, errText is %q", m.errText)
	}
}

func TestAPastedRuneOffThePromptMovesTheFocus(t *testing.T) {
	m, _ := newTestModel(t, "")
	m.burstAware = true
	m.focus = focusSidebar
	m, _ = step(t, m, key("a"))
	m, _ = step(t, m, key("s"))
	if m.focus != focusPrompt {
		t.Fatalf("the focus is %v, want the prompt", m.focus)
	}
	if m.seq != nil {
		t.Fatal("a pasted rune started a key sequence")
	}
	if got := m.prompt.Value(); got != "s" {
		t.Fatalf("prompt holds %q, want %q", got, "s")
	}
}
