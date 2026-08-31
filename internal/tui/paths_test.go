package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func tree(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range names {
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCompletePathOffersEveryDirectoryThatMatches(t *testing.T) {
	root := tree(t, "api/", "api-docs/", "assets/", "readme.md")

	completed, matches := completePath(filepath.Join(root, "a"))
	if len(matches) != 3 {
		t.Fatalf("matches = %v, want three directories", matches)
	}
	if completed != filepath.Join(root, "a") {
		t.Fatalf("completed = %q, want no change while three still match", completed)
	}
	for _, name := range matches {
		if name == "readme.md" {
			t.Error("a file must never be offered")
		}
	}
}

func TestCompletePathFinishesAUniqueName(t *testing.T) {
	root := tree(t, "api/", "api-docs/", "assets/")

	completed, matches := completePath(filepath.Join(root, "as"))
	if len(matches) != 1 || matches[0] != "assets" {
		t.Fatalf("matches = %v", matches)
	}
	want := filepath.Join(root, "assets") + string(filepath.Separator)
	if completed != want {
		t.Fatalf("completed = %q, want %q", completed, want)
	}
}

func TestCompletePathGrowsToTheCommonPrefix(t *testing.T) {
	root := tree(t, "workshop/", "workspace/", "other/")

	completed, matches := completePath(filepath.Join(root, "w"))
	if len(matches) != 2 {
		t.Fatalf("matches = %v", matches)
	}
	if completed != filepath.Join(root, "works") {
		t.Fatalf("completed = %q, want the common prefix", completed)
	}
}

func TestCompletePathListsTheChildrenOfAFinishedDirectory(t *testing.T) {
	root := tree(t, "api/", "api/handlers/", "api/store/")

	_, matches := completePath(filepath.Join(root, "api") + string(filepath.Separator))
	if len(matches) != 2 {
		t.Fatalf("matches = %v, want the two children", matches)
	}
}

func TestCompletePathHidesDotDirectoriesUntilYouAskForThem(t *testing.T) {
	root := tree(t, ".git/", ".multiplexier/", "app/")

	sep := string(filepath.Separator)
	_, matches := completePath(root + sep)
	if len(matches) != 1 || matches[0] != "app" {
		t.Fatalf("matches = %v, want only app", matches)
	}

	_, matches = completePath(root + sep + ".")
	if len(matches) != 2 {
		t.Fatalf("matches = %v, want both dot directories", matches)
	}
}

func TestCompletePathExpandsTheHomeMark(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	completed, _ := completePath("~/")
	if !strings.HasPrefix(completed, home) && completed != "~/" {
		t.Fatalf("completed = %q, want it under %q", completed, home)
	}
}

func TestCompletePathSurvivesADirectoryThatIsNotThere(t *testing.T) {
	input := filepath.Join(t.TempDir(), "nope", "deeper")
	completed, matches := completePath(input)
	if completed != input || matches != nil {
		t.Fatalf("completed = %q, matches = %v", completed, matches)
	}
}

func TestTabCompletesTheDirectoryFieldBeforeItMoves(t *testing.T) {
	root := tree(t, "workshop/", "workspace/")
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	if m.form == nil {
		t.Fatal("the form must be open")
	}

	m.form.inputs[fieldDir].SetValue(filepath.Join(root, "works"))
	m.form.inputs[fieldDir].CursorEnd()
	m.form.suggest()
	if !strings.Contains(visible(m.View()), "workshop") {
		t.Errorf("the form does not suggest the directories:\n%s", visible(m.View()))
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m, _ = step(t, m, key("tab"))
	want := filepath.Join(root, "workshop") + string(filepath.Separator)
	if got := m.form.inputs[fieldDir].Value(); got != want {
		t.Fatalf("directory = %q, want %q", got, want)
	}
	if m.form.focus != fieldDir {
		t.Fatal("tab must stay on the field while it completes")
	}

	m, _ = step(t, m, key("tab"))
	if m.form.focus != fieldName {
		t.Fatal("tab must move on when there is nothing left to complete")
	}
}

func TestShiftTabWalksBackwardsThroughTheSuggestions(t *testing.T) {
	root := tree(t, "alpha/", "beta/", "gamma/")
	sep := string(filepath.Separator)

	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form.inputs[fieldDir].SetValue(root + sep)
	m.form.inputs[fieldDir].CursorEnd()
	m.form.suggest()
	if len(m.form.matches) != 3 {
		t.Fatalf("matches = %v", m.form.matches)
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := m.form.inputs[fieldDir].Value(); got != root+sep+"gamma"+sep {
		t.Fatalf("first shift+tab gave %q, want the last suggestion", got)
	}
	if m.form.focus != fieldDir {
		t.Fatal("shift+tab must stay on the path line while it walks")
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := m.form.inputs[fieldDir].Value(); got != root+sep+"beta"+sep {
		t.Fatalf("second shift+tab gave %q", got)
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := m.form.inputs[fieldDir].Value(); got != root+sep+"gamma"+sep {
		t.Fatalf("the walk must wrap, got %q", got)
	}
}

func TestTabWalksForwardOnceTheWalkHasStarted(t *testing.T) {
	root := tree(t, "alpha/", "beta/", "gamma/")
	sep := string(filepath.Separator)

	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form.inputs[fieldDir].SetValue(root + sep)
	m.form.inputs[fieldDir].CursorEnd()
	m.form.suggest()

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	m, _ = step(t, m, key("tab"))
	if got := m.form.inputs[fieldDir].Value(); got != root+sep+"alpha"+sep {
		t.Fatalf("tab gave %q, want the walk to wrap forward", got)
	}
	if m.form.focus != fieldDir {
		t.Fatal("tab must not leave the field mid-walk")
	}
}

func TestTypingEndsTheWalk(t *testing.T) {
	root := tree(t, "alpha/", "beta/")
	sep := string(filepath.Separator)

	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form.inputs[fieldDir].SetValue(root + sep)
	m.form.inputs[fieldDir].CursorEnd()
	m.form.suggest()

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.form.picked < 0 {
		t.Fatal("the walk did not start")
	}
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.form.picked >= 0 {
		t.Fatal("typing must end the walk")
	}
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.form.focus == fieldDir {
		t.Fatal("with no match left, shift+tab should have moved the focus")
	}
}

func TestShiftTabMovesFieldsWhenThereIsNothingToWalk(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form.move(1)
	if m.form.focus != fieldName {
		t.Fatalf("focus = %d", m.form.focus)
	}
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.form.focus != fieldDir {
		t.Fatalf("shift+tab must move back a field, focus = %d", m.form.focus)
	}
}

func TestTheHintMarksTheChosenSuggestion(t *testing.T) {
	root := tree(t, "alpha/", "beta/")
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form.inputs[fieldDir].SetValue(root + string(filepath.Separator))
	m.form.inputs[fieldDir].CursorEnd()
	m.form.suggest()

	plain := m.form.View(100)
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	marked := m.form.View(100)
	if plain == marked {
		t.Fatal("the chosen suggestion must look different")
	}
	if !strings.Contains(visible(marked), "beta") {
		t.Fatalf("the suggestions vanished:\n%s", visible(marked))
	}
}
