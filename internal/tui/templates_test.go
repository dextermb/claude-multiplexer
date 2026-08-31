package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func writeTemplate(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func typeInto(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func lastPrompt(t *testing.T, mgr *manager.Manager, name string) string {
	t.Helper()
	var out string
	waitUntil(t, 10*time.Second, func() bool {
		for _, line := range mgr.Lines(name) {
			if line.Class == render.ClassPrompt {
				out = line.Text
			}
		}
		return out != ""
	})
	return out
}

func TestThePickerFillsThePromptFromATemplate(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "---\ndescription: Work a Linear issue\n---\nLook up {{issue}} and focus on {{focus=correctness}}.")
	writeTemplate(t, mgr.Root(), "review", "Review the branch.")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, key("t"))
	if m.picker == nil {
		t.Fatal("t must open the picker")
	}
	view := visible(m.View())
	for _, want := range []string{"Preset prompts", "/linear", "Work a Linear issue", "/review"} {
		if !strings.Contains(view, want) {
			t.Errorf("the picker does not show %q:\n%s", want, view)
		}
	}

	m, _ = step(t, m, key("enter"))
	if m.picker != nil || m.fields == nil {
		t.Fatal("choosing a template must open the field form")
	}
	if got := visible(m.View()); !strings.Contains(got, "issue") || !strings.Contains(got, "focus") {
		t.Errorf("the form does not ask for the fields:\n%s", got)
	}

	m = typeInto(t, m, "ENG-123")
	m, _ = step(t, m, key("enter"))
	if m.fields != nil {
		t.Fatal("enter must close the form")
	}
	want := "Look up ENG-123 and focus on correctness."
	if m.prompt.Value() != want {
		t.Fatalf("prompt = %q, want %q", m.prompt.Value(), want)
	}
	if m.focus != focusPrompt {
		t.Error("the focus must follow the text into the prompt")
	}
	if snap, _ := mgr.Snapshot("alpha"); snap.Turns != 0 {
		t.Error("filling the prompt must not send it")
	}
}

func TestThePickerNarrowsAsYouType(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}}")
	writeTemplate(t, mgr.Root(), "lint", "Lint it")
	writeTemplate(t, mgr.Root(), "review", "Review it")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, key("t"))

	m = typeInto(t, m, "lin")
	if len(m.picker.matches) != 2 {
		t.Fatalf("matches = %d, want linear and lint", len(m.picker.matches))
	}
	m = typeInto(t, m, "t")
	if len(m.picker.matches) != 1 || m.picker.matches[0].Name != "lint" {
		t.Fatalf("matches = %+v, want only lint", m.picker.matches)
	}
	m, _ = step(t, m, key("esc"))
	if m.picker != nil {
		t.Fatal("esc must close the picker")
	}
}

func TestASlashNameExpandsAndSends(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}} and focus on {{focus=correctness}}.")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("/linear ENG-9 the retry path")
	m, _ = step(t, m, key("enter"))

	if m.fields != nil {
		t.Fatal("every field was given, so no form should open")
	}
	prompt := lastPrompt(t, mgr, "alpha")
	if !strings.Contains(prompt, "Look up ENG-9 and focus on the retry path.") {
		t.Fatalf("the session received %q", prompt)
	}
}

func TestASlashNameWithAMissingFieldOpensTheForm(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}} for {{owner}}.")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("/linear ENG-9")
	m, _ = step(t, m, key("enter"))

	if m.fields == nil {
		t.Fatal("a missing field must open the form")
	}
	if got := m.fields.inputs[0].Value(); got != "ENG-9" {
		t.Fatalf("the form lost the argument you gave, issue = %q", got)
	}
	m = typeInto(t, m, "dexter")
	m, _ = step(t, m, key("enter"))
	if m.prompt.Value() != "Look up ENG-9 for dexter." {
		t.Fatalf("prompt = %q", m.prompt.Value())
	}
}

func TestAnUnknownSlashNameGoesStraightToTheChild(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}}")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("/some-claude-code-command with args")
	m, _ = step(t, m, key("enter"))

	if m.fields != nil {
		t.Fatal("an unknown name must not open a form")
	}
	prompt := lastPrompt(t, mgr, "alpha")
	if !strings.Contains(prompt, "/some-claude-code-command with args") {
		t.Fatalf("the child received %q, want the text unchanged", prompt)
	}
}

func TestTabCompletesASlashName(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}}")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("/lin")
	m.reloadTemplates()
	if !strings.Contains(visible(m.promptView()), "/linear") {
		t.Errorf("the completion is not offered:\n%s", visible(m.promptView()))
	}

	m, _ = step(t, m, key("tab"))
	if m.prompt.Value() != "/linear " {
		t.Fatalf("prompt = %q, want the completed name", m.prompt.Value())
	}
	if m.focus != focusPrompt {
		t.Fatal("tab must not move the focus while it completes")
	}
}

func TestTabStillMovesTheFocusWithNothingToComplete(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("plain text")
	m, _ = step(t, m, key("tab"))
	if m.focus == focusPrompt {
		t.Fatal("tab must move the focus when there is nothing to complete")
	}
}

func TestANewSessionCanStartWithAPreset(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}}.")

	m = start(t, m, 100, 24)
	if m.form == nil {
		t.Fatal("the form must be open")
	}
	dir := t.TempDir()
	m.form.inputs[fieldDir].SetValue(dir)
	m.form.inputs[fieldName].SetValue("api")
	m.form.inputs[fieldFirst].SetValue("/linear ENG-42")
	m, cmd := step(t, m, key("enter"))
	if cmd == nil {
		t.Fatal("the form must spawn a session")
	}
	msg, ok := cmd().(spawnedMsg)
	if !ok || msg.err != nil {
		t.Fatalf("spawn gave %+v", msg)
	}
	m, _ = step(t, m, msg)

	prompt := lastPrompt(t, mgr, msg.name)
	if !strings.Contains(prompt, "Look up ENG-42.") {
		t.Fatalf("the new session received %q", prompt)
	}
}

func TestAProjectTemplateWinsOverTheHomeOne(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "the home one {{issue}}")

	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".multiplexier", "templates")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "linear.md"), []byte("the project one {{issue}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", dir)

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("/linear ENG-1")
	m, _ = step(t, m, key("enter"))

	prompt := lastPrompt(t, mgr, "alpha")
	if !strings.Contains(prompt, "the project one ENG-1") {
		t.Fatalf("the session received %q, want the project template", prompt)
	}
}

func TestANamedArgumentFillsAFieldFromThePrompt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	writeTemplate(t, mgr.Root(), "linear", "Look up {{issue}} for {{owner}} focus {{focus=correctness}}.")

	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue(`/linear owner=dexter issue=ENG-4 focus="the retry path"`)
	m, _ = step(t, m, key("enter"))

	if m.fields != nil {
		t.Fatal("every field was named, so no form should open")
	}
	prompt := lastPrompt(t, mgr, "alpha")
	if !strings.Contains(prompt, "Look up ENG-4 for dexter focus the retry path.") {
		t.Fatalf("the session received %q", prompt)
	}
}
