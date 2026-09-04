package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/template"
)

type pickerResult int

const (
	pickerOpen pickerResult = iota
	pickerChosen
	pickerCancelled
)

type picker struct {
	all     []template.Template
	matches []template.Template
	dirs    []string
	filter  textinput.Model
	index   int
}

func newPicker(all []template.Template, dirs []string) *picker {
	filter := textinput.New()
	filter.Placeholder = "name"
	filter.Prompt = "/"
	filter.CharLimit = 64
	filter.Width = 30
	filter.Focus()

	p := &picker{all: all, dirs: dirs, filter: filter}
	p.narrow()
	return p
}

func (p *picker) narrow() {
	p.matches = template.Match(p.all, strings.TrimSpace(p.filter.Value()))
	if p.index >= len(p.matches) {
		p.index = len(p.matches) - 1
	}
	if p.index < 0 {
		p.index = 0
	}
}

func (p *picker) Update(msg tea.Msg) (pickerResult, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return pickerCancelled, nil
		case "enter":
			if len(p.matches) == 0 {
				return pickerOpen, nil
			}
			return pickerChosen, nil
		case "up", "ctrl+k":
			if p.index > 0 {
				p.index--
			}
			return pickerOpen, nil
		case "down", "ctrl+j":
			if p.index < len(p.matches)-1 {
				p.index++
			}
			return pickerOpen, nil
		}
	}
	var cmd tea.Cmd
	p.filter, cmd = p.filter.Update(msg)
	p.narrow()
	return pickerOpen, cmd
}

func (p *picker) selected() (template.Template, bool) {
	if p.index < 0 || p.index >= len(p.matches) {
		return template.Template{}, false
	}
	return p.matches[p.index], true
}

func (p *picker) View(width int) string {
	inner := modalInner(width)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Preset prompts"))
	b.WriteString("\n\n")
	b.WriteString(p.filter.View())
	b.WriteString("\n\n")

	switch {
	case len(p.all) == 0:
		b.WriteString(hintStyle.Render("No templates yet. A markdown file in one of these\n" +
			"makes a preset, and every {{field}} becomes a question:\n"))
		for _, dir := range p.dirs {
			b.WriteString(hintStyle.Render("\n  " + truncate(dir, inner-4)))
		}
		b.WriteString("\n")
	case len(p.matches) == 0:
		b.WriteString(hintStyle.Render("Nothing matches that name."))
	default:
		rowWidth := inner - 4
		for i, tpl := range p.matches {
			row := pad("/"+tpl.Name, 16) + truncate(tpl.Description, rowWidth-19)
			if i == p.index {
				b.WriteString(selectedRowStyle.Width(rowWidth).Render("▸ " + row))
			} else {
				b.WriteString(rowStyle.Width(rowWidth).Render("  " + row))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n" + hintStyle.Render("↑↓ move · enter choose · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

type fieldForm struct {
	tpl    template.Template
	inputs []textinput.Model
	focus  int
	err    string
}

func newFieldForm(tpl template.Template, values map[string]string) *fieldForm {
	f := &fieldForm{tpl: tpl}
	for _, field := range tpl.Fields {
		input := textinput.New()
		input.Placeholder = field.Default
		input.SetValue(values[field.Name])
		input.CharLimit = 512
		input.Width = 40
		f.inputs = append(f.inputs, input)
	}
	if len(f.inputs) > 0 {
		f.focus = f.firstEmpty()
		f.inputs[f.focus].Focus()
		f.inputs[f.focus].CursorEnd()
	}
	return f
}

func (f *fieldForm) firstEmpty() int {
	for i := range f.inputs {
		if strings.TrimSpace(f.inputs[i].Value()) == "" {
			return i
		}
	}
	return 0
}

func (f *fieldForm) Update(msg tea.Msg) (formResult, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return formCancelled, nil
		case "enter":
			if f.validate() {
				return formSubmitted, nil
			}
			return formOpen, nil
		case "tab", "down":
			f.move(1)
			return formOpen, nil
		case "shift+tab", "up":
			f.move(-1)
			return formOpen, nil
		}
	}
	if len(f.inputs) == 0 {
		return formOpen, nil
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return formOpen, cmd
}

func (f *fieldForm) move(delta int) {
	if len(f.inputs) == 0 {
		return
	}
	f.inputs[f.focus].Blur()
	f.focus = (f.focus + delta + len(f.inputs)) % len(f.inputs)
	f.inputs[f.focus].Focus()
	f.inputs[f.focus].CursorEnd()
}

func (f *fieldForm) validate() bool {
	for i, field := range f.tpl.Fields {
		if strings.TrimSpace(f.inputs[i].Value()) == "" && field.Default == "" {
			f.err = "give a value for " + field.Name
			f.focus = i
			return false
		}
	}
	f.err = ""
	return true
}

func (f *fieldForm) values() map[string]string {
	values := make(map[string]string, len(f.inputs))
	for i, field := range f.tpl.Fields {
		value := strings.TrimSpace(f.inputs[i].Value())
		if value == "" {
			value = field.Default
		}
		values[field.Name] = value
	}
	return values
}

func (f *fieldForm) prompt() string {
	return f.tpl.Expand(f.values())
}

func (f *fieldForm) View(width int) string {
	inner := modalInner(width)

	var b strings.Builder
	b.WriteString(titleStyle.Render("/" + f.tpl.Name))
	if f.tpl.Description != "" {
		b.WriteString("\n" + hintStyle.Render(truncate(f.tpl.Description, inner-2)))
	}
	b.WriteString("\n\n")
	for i, field := range f.tpl.Fields {
		b.WriteString(fieldLabelStyle.Render(pad(field.Name, 16)))
		b.WriteString(f.inputs[i].View())
		b.WriteString("\n")
	}
	if f.err != "" {
		b.WriteString("\n" + errorStyle.Render(f.err) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("tab move · enter fill the prompt · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

func completionNames(all []template.Template, text string) []string {
	trimmed := strings.TrimLeft(text, " \t")
	if !strings.HasPrefix(trimmed, "/") || strings.ContainsAny(trimmed, " \t\n") {
		return nil
	}
	var names []string
	for _, tpl := range template.Match(all, trimmed[1:]) {
		names = append(names, "/"+tpl.Name)
	}
	return names
}
