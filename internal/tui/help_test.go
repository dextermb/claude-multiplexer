package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/keys"
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

	view := visible(m.screen())
	for _, want := range []string{"KEYS", "QUICK KEYS", "start a new session", "esc close"} {
		if !strings.Contains(view, want) {
			t.Errorf("the list is missing %q:\n%s", want, view)
		}
	}

	rows := strings.Join(listText(m.help, m.keys, nil), "\n")
	for _, want := range []string{"THE SESSION (S)", "THE LIST (L)", "EVERYWHERE", "Show this list"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the list holds no %q:\n%s", want, rows)
		}
	}
	if len(listText(m.help, m.keys, nil)) < len(bindings) {
		t.Errorf("the list shows %d rows for %d bindings", len(listText(m.help, m.keys, nil)), len(bindings))
	}
}

func TestTheKeyListShowsTheCommands(t *testing.T) {
	h := newHelp()
	cmds := []config.Command{{Keys: "b o", Label: "browser", Script: "o.sh"}}
	rows := strings.Join(listText(h, defaultKeymap(), cmds), "\n")
	for _, want := range []string{"YOUR COMMANDS", "b o", "browser"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the list is missing %q:\n%s", want, rows)
		}
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
		m, _ = step(t, m, key(string(r)))
	}
	narrowed := strings.Join(listText(m.help, m.keys, nil), "\n")
	if !strings.Contains(narrowed, "Archive the selected session") {
		t.Fatalf("the search lost the row it should keep:\n%s", narrowed)
	}
	if strings.Contains(narrowed, "Add a new line inside the prompt") {
		t.Fatalf("the search kept a row it should drop:\n%s", narrowed)
	}
	if len(listText(m.help, m.keys, nil)) >= len(bindings) {
		t.Error("the search narrowed nothing")
	}

	for i := 0; i < len("archive"); i++ {
		m, _ = step(t, m, key("backspace"))
	}
	restored := strings.Join(listText(m.help, m.keys, nil), "\n")
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
		m, _ = step(t, m, key(string(r)))
	}
	rows := strings.Join(listText(m.help, m.keys, nil), "\n")
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
		m, _ = step(t, m, key(string(r)))
	}
	if !strings.Contains(visible(m.screen()), "No key matches that.") {
		t.Fatalf("view:\n%s", visible(m.screen()))
	}
}

func TestTheKeyListScrolls(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 22)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	first := visible(m.screen())
	m, _ = step(t, m, key("down"))
	if visible(m.screen()) == first {
		t.Fatal("down must scroll the list")
	}
	m, _ = step(t, m, key("up"))
	if visible(m.screen()) != first {
		t.Fatal("up must scroll back")
	}
	m, _ = step(t, m, key("up"))
	if visible(m.screen()) != first {
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
	if m.form == nil {
		t.Fatal("enter must run the key under the cursor, the first of which starts a new session")
	}
	m, _ = step(t, m, key("esc"))

	m = openHelp(t, m)
	m, _ = step(t, m, key("ctrl+c"))
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
	if !strings.Contains(m.statusHints(), "? keys") {
		t.Error("statusHints must name the key list")
	}
}

func TestFitHintsDropsWholeHints(t *testing.T) {
	plain := "m model · p mode · e effort"
	if got := fitHints(plain, 80); got != plain {
		t.Fatalf("a list that fits must stay whole, got %q", got)
	}
	if got := fitHints(plain, 18); got != "m model · p mode · …" {
		t.Fatalf("got %q, want whole hints and an ellipsis", got)
	}
	if got := fitHints(plain, 3); got != "…" {
		t.Fatalf("got %q, want only the ellipsis", got)
	}
}

// listText is the key list as text: each group name in capitals, then its keys
// and what they do.
func listText(h *help, km keys.Keymap, cmds []config.Command) []string {
	var out []string
	group := ""
	for _, e := range h.entries(km, cmds) {
		if e.group != group {
			group = e.group
			out = append(out, strings.ToUpper(group))
		}
		out = append(out, e.keys+" "+e.what)
	}
	return out
}

func TestTheKeyListRunsTheKeyUnderTheCursor(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 40)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	for _, r := range "rename" {
		m, _ = step(t, m, key(string(r)))
	}
	m, _ = step(t, m, key("enter"))
	if m.help != nil {
		t.Fatal("enter must close the list")
	}
	if renameOf(m) == nil {
		t.Fatal("enter on the rename key must open the rename dialog")
	}
}

func TestAPaneKeySaysWhereItWorks(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 40)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)

	for _, r := range "new line" {
		m, _ = step(t, m, key(string(r)))
	}
	m, _ = step(t, m, key("enter"))
	if !strings.Contains(m.status, "the prompt") {
		t.Fatalf("status = %q, want where the key works", m.status)
	}
}

func TestTheArrowsMoveBetweenColumns(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 200, 40)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = openHelp(t, m)
	_ = m.screen()

	if m.help.cols < 2 {
		t.Fatalf("a wide terminal must show more than one column, got %d", m.help.cols)
	}
	m, _ = step(t, m, key("right"))
	_ = m.screen()
	if col, _ := m.help.place(); col != 1 {
		t.Fatalf("right must move the cursor to the next column, got column %d", col)
	}
	m, _ = step(t, m, key("left"))
	if col, _ := m.help.place(); col != 0 {
		t.Fatalf("left must move the cursor back, got column %d", col)
	}
}
