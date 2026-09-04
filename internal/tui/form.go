package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const (
	fieldDir = iota
	fieldName
	fieldModel
	fieldMode
	fieldEffort
	fieldControl
	fieldFirst
	fieldCount
)

var fieldLabels = [fieldCount]string{"Directory", "Name", "Model", "Permission mode", "Effort", "Control", "First prompt"}

type formResult int

const (
	formOpen formResult = iota
	formSubmitted
	formCancelled
)

type form struct {
	inputs  [fieldCount]textinput.Model
	focus   int
	err     string
	matches []string
	picked  int
	stem    string
}

func newForm(dir, model, mode string) *form {
	f := &form{}
	placeholders := [fieldCount]string{
		dir,
		"taken from the directory",
		"the Claude Code default",
		mode,
		"low, medium, high, xhigh, max",
		"no — yes lets it drive other sessions",
		"optional, and /preset works here",
	}
	values := [fieldCount]string{dir, "", model, mode, "", "", ""}
	for i := range f.inputs {
		input := textinput.New()
		input.Placeholder = placeholders[i]
		input.SetValue(values[i])
		input.CharLimit = 512
		input.Width = 40
		f.inputs[i] = input
	}
	f.inputs[fieldDir].CursorEnd()
	f.inputs[fieldDir].Focus()
	f.suggest()
	return f
}

func (f *form) suggest() {
	f.picked = -1
	f.stem = ""
	if f.focus != fieldDir {
		f.matches = nil
		return
	}
	_, f.matches = completePath(f.inputs[fieldDir].Value())
}

func (f *form) cycle(delta int) bool {
	if f.focus != fieldDir || len(f.matches) == 0 {
		return false
	}
	if f.picked < 0 {
		f.stem, _ = splitPath(expandHome(f.inputs[fieldDir].Value()))
		f.picked = 0
		if delta < 0 {
			f.picked = len(f.matches) - 1
		}
	} else {
		f.picked = (f.picked + delta + len(f.matches)) % len(f.matches)
	}
	f.inputs[fieldDir].SetValue(f.stem + f.matches[f.picked] + string(filepath.Separator))
	f.inputs[fieldDir].CursorEnd()
	return true
}

func (f *form) completeDir() bool {
	value := f.inputs[fieldDir].Value()
	completed, matches := completePath(value)
	f.matches = matches
	if completed == value || completed == "" {
		return false
	}
	f.inputs[fieldDir].SetValue(completed)
	f.inputs[fieldDir].CursorEnd()
	f.suggest()
	return true
}

func (f *form) Update(msg tea.Msg) (formResult, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return formCancelled, nil
		case "enter":
			if f.validate() {
				return formSubmitted, nil
			}
			return formOpen, nil
		case "tab":
			if f.focus == fieldDir {
				if f.picked >= 0 && f.cycle(1) {
					return formOpen, nil
				}
				if f.completeDir() {
					return formOpen, nil
				}
			}
			f.move(1)
			return formOpen, nil
		case "shift+tab":
			if f.cycle(-1) {
				return formOpen, nil
			}
			f.move(-1)
			return formOpen, nil
		case "down":
			f.move(1)
			return formOpen, nil
		case "up":
			f.move(-1)
			return formOpen, nil
		}
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	f.suggest()
	return formOpen, cmd
}

func (f *form) move(delta int) {
	f.inputs[f.focus].Blur()
	f.focus = (f.focus + delta + fieldCount) % fieldCount
	f.inputs[f.focus].Focus()
	f.inputs[f.focus].CursorEnd()
	f.suggest()
}

func (f *form) insert(text string, paths bool) {
	input := &f.inputs[f.focus]
	if paths && f.focus == fieldDir {
		input.SetValue(directoryOf(unquote(text)))
		input.CursorEnd()
		return
	}
	runes := []rune(input.Value())
	at := input.Position()
	if at < 0 {
		at = 0
	}
	if at > len(runes) {
		at = len(runes)
	}
	input.SetValue(string(runes[:at]) + text + string(runes[at:]))
	input.SetCursor(at + len([]rune(text)))
}

func unquote(text string) string {
	if tokens := splitTokens(text); len(tokens) > 0 {
		return tokens[0]
	}
	return text
}

func directoryOf(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return path
	}
	return filepath.Dir(path)
}

func (f *form) validate() bool {
	dir := strings.TrimSpace(f.inputs[fieldDir].Value())
	if dir == "" {
		f.err = "give a directory"
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		f.err = err.Error()
		return false
	}
	info, err := os.Stat(abs)
	if err != nil {
		f.err = "no such directory: " + abs
		return false
	}
	if !info.IsDir() {
		f.err = abs + " is not a directory"
		return false
	}
	if effort := strings.TrimSpace(f.inputs[fieldEffort].Value()); effort != "" && !knownEffort(effort) {
		f.err = "effort is one of " + strings.Join(session.EffortLevels, ", ")
		return false
	}
	f.inputs[fieldDir].SetValue(abs)
	f.err = ""
	return true
}

func knownEffort(level string) bool {
	for _, known := range session.EffortLevels {
		if known == level {
			return true
		}
	}
	return false
}

func (f *form) spec() manager.Spec {
	return manager.Spec{
		Dir:            strings.TrimSpace(f.inputs[fieldDir].Value()),
		Name:           strings.TrimSpace(f.inputs[fieldName].Value()),
		Model:          strings.TrimSpace(f.inputs[fieldModel].Value()),
		PermissionMode: strings.TrimSpace(f.inputs[fieldMode].Value()),
		Effort:         strings.TrimSpace(f.inputs[fieldEffort].Value()),
		Control:        affirmative(f.inputs[fieldControl].Value()),
	}
}

func affirmative(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "on", "1":
		return true
	}
	return false
}

func (f *form) firstPrompt() string {
	return strings.TrimSpace(f.inputs[fieldFirst].Value())
}

func (f *form) View(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("New session"))
	b.WriteString("\n\n")
	inner := modalInner(width)
	for i := range f.inputs {
		b.WriteString(fieldLabelStyle.Render(pad(fieldLabels[i], 16)))
		b.WriteString(f.inputs[i].View())
		b.WriteString("\n")
		if i == fieldDir && f.focus == fieldDir {
			if hint := pathHint(f.matches, f.picked, inner-18); hint != "" {
				b.WriteString(strings.Repeat(" ", 16) + hintStyle.Render(hint) + "\n")
			}
		}
	}
	if f.err != "" {
		b.WriteString("\n" + errorStyle.Render(f.err) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("tab complete · shift+tab walk · enter start · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

func centre(width, height int, content string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
