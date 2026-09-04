package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func openHelp(t *testing.T, m Model) Model {
	t.Helper()
	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, key("?"))
	if m.help == nil {
		t.Fatal("? must open the key list")
	}
	return m
}

func TestQuestionMarkShowsEveryKey(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	view := visible(m.View())
	for _, want := range []string{"Keys", "Quick keys", "Start a new session", "esc close"} {
		if !strings.Contains(view, want) {
			t.Errorf("the list is missing %q:\n%s", want, view)
		}
	}

	rows := strings.Join(visibleAll(m.help.rows(80)), "\n")
	for _, want := range []string{"The session (s)", "The list (l)", "Everywhere", "Show this list"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the list holds no %q:\n%s", want, rows)
		}
	}
	if len(m.help.rows(80)) < len(bindings) {
		t.Errorf("the list shows %d rows for %d bindings", len(m.help.rows(80)), len(bindings))
	}
}

func visibleAll(rows []string) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, visible(row))
	}
	return out
}

func TestTheKeyListSearches(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	for _, r := range "archive" {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	narrowed := strings.Join(visibleAll(m.help.rows(80)), "\n")
	if !strings.Contains(narrowed, "Archive the selected session") {
		t.Fatalf("the search lost the row it should keep:\n%s", narrowed)
	}
	if strings.Contains(narrowed, "Add a new line inside the prompt") {
		t.Fatalf("the search kept a row it should drop:\n%s", narrowed)
	}
	if len(m.help.rows(80)) >= len(bindings) {
		t.Error("the search narrowed nothing")
	}

	for i := 0; i < len("archive"); i++ {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	restored := strings.Join(visibleAll(m.help.rows(80)), "\n")
	if !strings.Contains(restored, "Add a new line inside the prompt") {
		t.Fatal("clearing the search must bring every key back")
	}
}

func TestTheKeyListSearchesTheKeysThemselves(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	for _, r := range "ctrl+j" {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	rows := strings.Join(visibleAll(m.help.rows(80)), "\n")
	if !strings.Contains(rows, "Add a new line inside the prompt") {
		t.Fatalf("a search for a key did not find it:\n%s", rows)
	}
}

func TestTheKeyListSaysWhenNothingMatches(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	for _, r := range "zzzz" {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if !strings.Contains(visible(m.View()), "No key matches that.") {
		t.Fatalf("view:\n%s", visible(m.View()))
	}
}

func TestTheKeyListScrolls(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 22)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	first := visible(m.View())
	m, _ = step(t, m, key("down"))
	if visible(m.View()) == first {
		t.Fatal("down must scroll the list")
	}
	m, _ = step(t, m, key("up"))
	if visible(m.View()) != first {
		t.Fatal("up must scroll back")
	}
	m, _ = step(t, m, key("up"))
	if visible(m.View()) != first {
		t.Fatal("the list must stop at the top")
	}
}

func TestEscAndEnterCloseTheKeyList(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m = openHelp(t, m)
	m, _ = step(t, m, key("esc"))
	if m.help != nil {
		t.Fatal("esc must close the list")
	}

	m = openHelp(t, m)
	m, _ = step(t, m, key("enter"))
	if m.help != nil {
		t.Fatal("enter must close the list")
	}

	m = openHelp(t, m)
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.help != nil {
		t.Fatal("ctrl+c must close the list")
	}
	if m.quitting {
		t.Fatal("closing the list must not quit")
	}
}

func TestAQuestionMarkInThePromptIsJustText(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m, _ = step(t, m, key("?"))
	if m.help != nil {
		t.Fatal("? in the prompt must not open the list")
	}
	if m.prompt.Value() != "?" {
		t.Fatalf("prompt = %q", m.prompt.Value())
	}
}

func TestTheStatusBarHintsComeFromTheSameTable(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 26)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	status := visible(m.statusView())
	for _, want := range []string{"n new", "t preset", "? keys", "q quit"} {
		if !strings.Contains(status, want) {
			t.Errorf("the status bar is missing %q:\n%s", want, status)
		}
	}
	if !strings.Contains(statusHints(), "? keys") {
		t.Error("statusHints must name the key list")
	}
}
