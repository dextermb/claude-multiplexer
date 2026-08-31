package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func touch(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDropTextReadsOnePath(t *testing.T) {
	dir := t.TempDir()
	path := touch(t, dir, "notes.md")

	got, paths := dropText(path + "\n")
	if !paths || got != path {
		t.Fatalf("dropText = %q, %v", got, paths)
	}
}

func TestDropTextUnescapesASpace(t *testing.T) {
	dir := t.TempDir()
	path := touch(t, dir, "my notes.md")

	got, paths := dropText(strings.ReplaceAll(path, " ", `\ `))
	if !paths {
		t.Fatalf("dropText did not recognise the path: %q", got)
	}
	if got != `"`+path+`"` {
		t.Fatalf("dropText = %q, want the path in quotes", got)
	}
}

func TestDropTextReadsSeveralPaths(t *testing.T) {
	dir := t.TempDir()
	one := touch(t, dir, "one.go")
	two := touch(t, dir, "two.go")

	got, paths := dropText(one + " " + two)
	if !paths || got != one+" "+two {
		t.Fatalf("dropText = %q, %v", got, paths)
	}
}

func TestDropTextAcceptsAQuotedPath(t *testing.T) {
	dir := t.TempDir()
	path := touch(t, dir, "my notes.md")

	got, paths := dropText(`"` + path + `"`)
	if !paths || got != `"`+path+`"` {
		t.Fatalf("dropText = %q, %v", got, paths)
	}
}

func TestDropTextAcceptsAFileURL(t *testing.T) {
	dir := t.TempDir()
	path := touch(t, dir, "my notes.md")

	got, paths := dropText("file://" + strings.ReplaceAll(path, " ", "%20"))
	if !paths || got != `"`+path+`"` {
		t.Fatalf("dropText = %q, %v", got, paths)
	}
}

func TestDropTextLeavesOrdinaryTextAlone(t *testing.T) {
	for _, raw := range []string{
		"  please refactor this  ",
		"/no/such/file/anywhere.txt",
		"one two three",
	} {
		got, paths := dropText(raw)
		if paths {
			t.Errorf("%q was read as a path list", raw)
		}
		if got != strings.TrimSpace(raw) {
			t.Errorf("dropText(%q) = %q", raw, got)
		}
	}
}

func TestDropTextIgnoresEmptyInput(t *testing.T) {
	if got, paths := dropText("   \n "); got != "" || paths {
		t.Fatalf("dropText = %q, %v", got, paths)
	}
}

func TestADropOnTheSidebarGoesToThePrompt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	path := touch(t, t.TempDir(), "dropped.go")

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(path), Paste: true})
	if m.focus != focusPrompt {
		t.Fatalf("focus = %v, want the prompt", m.focus)
	}
	if m.prompt.Value() != path+" " {
		t.Fatalf("prompt = %q, want the path and a space", m.prompt.Value())
	}
	if len(m.rows) != 1 {
		t.Fatal("the drop must not reach the sidebar keys")
	}
}

func TestADropNeverTriggersASidebarKey(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	dir := t.TempDir()
	path := touch(t, dir, "quit-and-archive-nx.go")

	m.focus = focusSidebar
	m.prompt.Blur()
	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(path), Paste: true})
	if m.quitting {
		t.Fatal("a path holding q must not quit")
	}
	if m.confirm != "" {
		t.Fatal("a path holding x must not ask to stop")
	}
	if m.form != nil {
		t.Fatal("a path holding n must not open the form")
	}
	if cmd == nil {
		t.Fatal("the drop must still return the cursor command")
	}
}

func TestADropJoinsWhatYouAlreadyTyped(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	path := touch(t, t.TempDir(), "main.go")
	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("explain")

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(path), Paste: true})
	if m.prompt.Value() != "explain "+path+" " {
		t.Fatalf("prompt = %q", m.prompt.Value())
	}

	second := touch(t, t.TempDir(), "other.go")
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(second), Paste: true})
	if m.prompt.Value() != "explain "+path+" "+second+" " {
		t.Fatalf("prompt = %q", m.prompt.Value())
	}
}

func TestAnOrdinaryPasteGoesInAsItIs(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("check the login flow"), Paste: true})
	if m.prompt.Value() != "check the login flow" {
		t.Fatalf("prompt = %q", m.prompt.Value())
	}
}

func TestADropIntoTheFormSetsTheDirectory(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	if m.form == nil {
		t.Fatal("the form must be open")
	}

	dir := t.TempDir()
	file := touch(t, dir, "readme.md")
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(file), Paste: true})
	if got := m.form.inputs[fieldDir].Value(); got != dir {
		t.Fatalf("directory = %q, want the folder that holds the file", got)
	}

	other := t.TempDir()
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(other), Paste: true})
	if got := m.form.inputs[fieldDir].Value(); got != other {
		t.Fatalf("directory = %q, want the dropped folder", got)
	}
}

func TestADropIntoAnotherFormFieldInsertsText(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form.move(1)
	m.form.inputs[fieldName].SetValue("api")
	m.form.inputs[fieldName].SetCursor(3)

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-two"), Paste: true})
	if got := m.form.inputs[fieldName].Value(); got != "api-two" {
		t.Fatalf("name = %q", got)
	}
}

func TestASplitDropStillReachesThePrompt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()

	fragments := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune(`/tmp/drop\`)},
		{Type: tea.KeySpace, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune(`test/notes\`)},
		{Type: tea.KeySpace, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune("file.md")},
	}
	for _, fragment := range fragments {
		m, _ = step(t, m, fragment)
	}

	if m.focus != focusPrompt {
		t.Fatalf("focus = %v, want the prompt", m.focus)
	}
	if got := m.prompt.Value(); got != `/tmp/drop\ test/notes\ file.md` {
		t.Fatalf("prompt = %q", got)
	}
	if m.quitting || m.confirm != "" || m.form != nil {
		t.Fatal("the fragments must not reach the sidebar keys")
	}
}

func TestFastTypingStillReachesTheSidebar(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("kk")})
	if m.focus != focusSidebar {
		t.Fatal("two quick keys must not be read as a dropped path")
	}
	if m.prompt.Value() != "" {
		t.Fatalf("prompt = %q, want it untouched", m.prompt.Value())
	}
}

func TestASingleRuneWithASlashIsStillAKey(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if m.focus != focusSidebar {
		t.Fatal("one slash is a key press, not a drop")
	}
}

func TestTheOutputRendersMarkdownAndTheKeyShowsTheRawText(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	if err := mgr.Send("alpha", "The loader has **three** problems."); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitUntil(t, 10*time.Second, func() bool {
		snap, err := mgr.Snapshot("alpha")
		return err == nil && snap.Turns >= 1
	})
	m.refresh()
	m.rebuildOutput()

	rendered := visible(m.outputText)
	if got := strings.Count(rendered, "**three**"); got != 1 {
		t.Fatalf("the asterisks appear %d times, want only the prompt line:\n%s", got, rendered)
	}
	if got := strings.Count(rendered, "three"); got != 2 {
		t.Fatalf("three appears %d times, want the prompt and the reply:\n%s", got, rendered)
	}
	if !strings.Contains(rendered, "problems") {
		t.Errorf("the words were lost:\n%s", rendered)
	}
	if !strings.Contains(m.outputText, ";1m") && !strings.Contains(m.outputText, "[1m") {
		t.Error("nothing was made bold")
	}

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, key("m"))
	if !m.showRaw {
		t.Fatal("m must turn the markdown off")
	}
	if got := strings.Count(visible(m.outputText), "**three**"); got != 2 {
		t.Fatalf("the raw text appears %d times, want the prompt and the reply:\n%s",
			got, visible(m.outputText))
	}
	if !strings.Contains(visible(m.barView()), "raw") {
		t.Errorf("the bar does not say the pane is raw:\n%s", visible(m.barView()))
	}

	m, _ = step(t, m, key("m"))
	if m.showRaw {
		t.Fatal("m must turn the markdown back on")
	}
	if got := strings.Count(visible(m.outputText), "**three**"); got != 1 {
		t.Fatalf("the markdown did not come back, the asterisks appear %d times", got)
	}
}

func TestOnlyAssistantTextIsTreatedAsMarkdown(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassPrompt, Text: "› read ## the notes"},
		{Class: render.ClassToolResult, Text: "← 3 lines of **plain** output"},
		{Class: render.ClassMeta, Text: "✓ success · 1 turn"},
	})

	got := visible(m.outputText)
	for _, want := range []string{"› read ## the notes", "← 3 lines of **plain** output", "✓ success · 1 turn"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q was changed:\n%s", want, got)
		}
	}
}

func TestDropTextKeepsAnApostropheInAPath(t *testing.T) {
	dir := t.TempDir()
	path := touch(t, dir, "don't panic.md")

	got, paths := dropText(strings.ReplaceAll(path, " ", `\ `))
	if !paths {
		t.Fatalf("the path was not recognised: %q", got)
	}
	if got != `"`+path+`"` {
		t.Fatalf("dropText = %q, want %q in quotes", got, path)
	}

	quoted, paths := dropText(`"` + path + `"`)
	if !paths || quoted != `"`+path+`"` {
		t.Fatalf("dropText = %q, %v", quoted, paths)
	}
}
