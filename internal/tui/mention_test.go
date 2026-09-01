package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func mentionModel(t *testing.T, dir string) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "api", dir)
	if m.focus != focusPrompt {
		t.Fatal("the prompt must have the focus")
	}
	return m
}

func typed(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestMentionReadsTheWordThePromptEndsOn(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{"nothing typed", "", "", false},
		{"no mark", "read the file", "", false},
		{"the mark alone", "@", "", true},
		{"a relative path", "read @internal/tu", "internal/tu", true},
		{"an absolute path", "@/tmp/wo", "/tmp/wo", true},
		{"a home path", "@~/pro", "~/pro", true},
		{"after a new line", "one\n@two", "two", true},
		{"the mark is not first", "mail@example.com", "", false},
		{"a word follows the path", "@a b", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, ok := mention(tc.value)
			if ok != tc.ok {
				t.Fatalf("mention(%q) ok = %v, want %v", tc.value, ok, tc.ok)
			}
			if ok && token.text != tc.want {
				t.Fatalf("mention(%q) = %q, want %q", tc.value, token.text, tc.want)
			}
		})
	}
}

func TestAMentionSuggestsFilesAndDirectories(t *testing.T) {
	dir := tree(t, "api/", "app.go", "assets/", "readme.md")
	m := mentionModel(t, dir)

	m = typed(t, m, "read @a")
	names := matchNames(m.pathMatches)
	want := []string{"api", "app.go", "assets"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("matches = %v, want %v", names, want)
	}
	view := visible(m.View())
	for _, name := range want {
		if !strings.Contains(view, name) {
			t.Errorf("the view does not offer %q:\n%s", name, view)
		}
	}
}

func TestAMentionReadsTheSessionDirectoryNotOurs(t *testing.T) {
	dir := tree(t, "session-only/")
	m := mentionModel(t, dir)
	m.opts.DefaultDir = tree(t, "somewhere-else/")

	m = typed(t, m, "@s")
	names := matchNames(m.pathMatches)
	if len(names) != 1 || names[0] != "session-only" {
		t.Fatalf("matches = %v, want the session's own directory", names)
	}
}

func TestTabFinishesAUniqueMention(t *testing.T) {
	dir := tree(t, "internal/", "readme.md")
	m := mentionModel(t, dir)

	m = typed(t, m, "@int")
	m, _ = step(t, m, key("tab"))
	want := "@internal" + string(filepath.Separator)
	if m.prompt.Value() != want {
		t.Fatalf("prompt = %q, want %q", m.prompt.Value(), want)
	}
	if m.focus != focusPrompt {
		t.Fatal("a completion must not move the focus")
	}

	m.prompt.SetValue("")
	m = typed(t, m, "@rea")
	m, _ = step(t, m, key("tab"))
	if m.prompt.Value() != "@readme.md " {
		t.Fatalf("prompt = %q, want a file and a space", m.prompt.Value())
	}
}

func TestTabGrowsAMentionToTheCommonPrefix(t *testing.T) {
	dir := tree(t, "workshop/", "workspace/", "other.md")
	m := mentionModel(t, dir)

	m = typed(t, m, "@w")
	m, _ = step(t, m, key("tab"))
	if m.prompt.Value() != "@works" {
		t.Fatalf("prompt = %q, want @works", m.prompt.Value())
	}
	if m.focus != focusPrompt {
		t.Fatal("a completion must not move the focus")
	}
}

func TestAMentionCompletesAnAbsolutePath(t *testing.T) {
	dir := tree(t)
	other := tree(t, "notes/")
	m := mentionModel(t, dir)

	m = typed(t, m, "@"+filepath.Join(other, "no"))
	m, _ = step(t, m, key("tab"))
	want := "@" + filepath.Join(other, "notes") + string(filepath.Separator)
	if m.prompt.Value() != want {
		t.Fatalf("prompt = %q, want %q", m.prompt.Value(), want)
	}
}

func TestAMentionExpandsTheHomeDirectory(t *testing.T) {
	home := tree(t, "projects/")
	t.Setenv("HOME", home)
	m := mentionModel(t, tree(t))

	m = typed(t, m, "@~/pro")
	names := matchNames(m.pathMatches)
	if len(names) != 1 || names[0] != "projects" {
		t.Fatalf("matches = %v, want projects", names)
	}
}

func TestTabStillMovesTheFocusWithoutAMention(t *testing.T) {
	m := mentionModel(t, tree(t, "api/"))

	m = typed(t, m, "hello")
	m, _ = step(t, m, key("tab"))
	if m.focus != focusOutput {
		t.Fatalf("focus = %v, want the output pane", m.focus)
	}
}

func TestTabMovesTheFocusWhenNothingMatches(t *testing.T) {
	m := mentionModel(t, tree(t, "api/"))

	m = typed(t, m, "@zzz")
	m, _ = step(t, m, key("tab"))
	if m.focus != focusOutput {
		t.Fatalf("focus = %v, want the output pane", m.focus)
	}
}

func TestShiftTabWalksTheMentionMatches(t *testing.T) {
	dir := tree(t, "api/", "app.go", "assets/")
	m := mentionModel(t, dir)
	m = typed(t, m, "@a")

	m, _ = step(t, m, key("shift+tab"))
	if m.prompt.Value() != "@assets"+string(filepath.Separator) {
		t.Fatalf("prompt = %q, want the last match first", m.prompt.Value())
	}
	if !strings.Contains(visible(m.View()), "assets") {
		t.Fatalf("the view does not mark the walked name:\n%s", visible(m.View()))
	}

	m, _ = step(t, m, key("shift+tab"))
	if m.prompt.Value() != "@app.go" {
		t.Fatalf("prompt = %q, want the second match and no separator", m.prompt.Value())
	}

	m, _ = step(t, m, key("shift+tab"))
	if m.prompt.Value() != "@api"+string(filepath.Separator) {
		t.Fatalf("prompt = %q, want the first match", m.prompt.Value())
	}

	m, _ = step(t, m, key("shift+tab"))
	if m.prompt.Value() != "@assets"+string(filepath.Separator) {
		t.Fatalf("prompt = %q, want the walk to wrap", m.prompt.Value())
	}
}

func TestTabWalksTheOtherWay(t *testing.T) {
	dir := tree(t, "api/", "app.go", "assets/")
	m := mentionModel(t, dir)
	m = typed(t, m, "@a")

	m, _ = step(t, m, key("shift+tab"))
	m, _ = step(t, m, key("tab"))
	if m.prompt.Value() != "@api"+string(filepath.Separator) {
		t.Fatalf("prompt = %q, want tab to step the other way", m.prompt.Value())
	}
}

func TestTypingStopsTheWalk(t *testing.T) {
	dir := tree(t, "api/", "assets/")
	m := mentionModel(t, dir)
	m = typed(t, m, "@a")
	m, _ = step(t, m, key("shift+tab"))
	if m.pathPicked < 0 {
		t.Fatal("shift+tab must start a walk")
	}

	m = typed(t, m, "x")
	if m.pathPicked >= 0 {
		t.Fatal("a key press must stop the walk")
	}
}

func TestTheMentionSurvivesIntoTheSentPrompt(t *testing.T) {
	dir := tree(t, "notes.md")
	m := mentionModel(t, dir)

	m = typed(t, m, "read @not")
	m, _ = step(t, m, key("tab"))
	if m.prompt.Value() != "read @notes.md " {
		t.Fatalf("prompt = %q", m.prompt.Value())
	}
}

func TestAMentionIsNotCompletedAwayFromTheEnd(t *testing.T) {
	dir := tree(t, "api/")
	m := mentionModel(t, dir)

	m = typed(t, m, "@ap")
	m.prompt.SetCursor(2)
	m, _ = step(t, m, key("tab"))
	if m.prompt.Value() != "@ap" {
		t.Fatalf("prompt = %q, want no change while the cursor sits inside", m.prompt.Value())
	}
}
