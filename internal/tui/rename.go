package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type renameDialog struct {
	session string
	input   textinput.Model
}

func newRenameDialog(name, current string) *renameDialog {
	input := textinput.New()
	input.Placeholder = name
	input.CharLimit = 512
	input.Width = 40
	input.SetValue(current)
	input.CursorEnd()
	input.Focus()
	return &renameDialog{session: name, input: input}
}

func (d *renameDialog) Update(msg tea.Msg) (formResult, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return formCancelled, nil
		case "enter":
			return formSubmitted, nil
		}
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return formOpen, cmd
}

func (d *renameDialog) value() string { return strings.TrimSpace(d.input.Value()) }

func (d *renameDialog) View(width int) string {
	inner := width - 8
	if inner < 40 {
		inner = 40
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Rename"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("for " + d.session))
	b.WriteString("\n\n")
	b.WriteString(d.input.View())
	b.WriteString("\n\n" + hintStyle.Render("enter apply · esc cancel · empty clears the title"))
	return modalStyle.Width(inner).Render(b.String())
}
