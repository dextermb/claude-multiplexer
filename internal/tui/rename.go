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

func (d *renameDialog) region() modalRegion      { return modalPane }
func (d *renameDialog) view(width, _ int) string { return d.View(width) }

func (d *renameDialog) update(m *Model, msg tea.Msg) (modal, tea.Cmd) {
	result, cmd := d.Update(msg)
	return applyForm(d, m, result, cmd, d.submit)
}

func (d *renameDialog) submit(m *Model) (modal, tea.Cmd) {
	name, title := d.session, d.value()
	if err := m.mgr.SetTitle(name, title); err != nil {
		m.errText = err.Error()
		return nil, nil
	}
	for i := range m.stored {
		if m.stored[i].Name == name {
			m.stored[i].Title = title
		}
	}
	m.errText = ""
	m.status = "renamed " + name
	m.refresh()
	m.rebuildOutput()
	return nil, nil
}

func (m Model) openRename() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		m.errText = "no session is selected"
		return m, nil
	}
	if item.readOnly {
		m.errText = readOnlyStatus
		return m, nil
	}
	m.modal = newRenameDialog(item.name, item.title)
	m.errText = ""
	return m, textinput.Blink
}

func (d *renameDialog) View(width int) string {
	inner := modalInner(width)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Rename"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("for " + d.session))
	b.WriteString("\n\n")
	b.WriteString(d.input.View())
	b.WriteString("\n\n" + hintStyle.Render("enter apply · esc cancel · empty clears the title"))
	return modalStyle.Width(inner).Render(b.String())
}
