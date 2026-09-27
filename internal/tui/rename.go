package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type renameDialog struct {
	session string
	title   string
	form    *huh.Form
}

func newRenameDialog(name, current string) *renameDialog {
	d := &renameDialog{session: name, title: current}
	field := huh.NewInput().Prompt("› ").Placeholder(name).CharLimit(512).Value(&d.title)
	d.form = newHuhForm(huh.NewGroup(field))
	d.form.Init()
	return d
}

func (d *renameDialog) Update(msg tea.Msg) (formResult, tea.Cmd) {
	var result formResult
	var cmd tea.Cmd
	d.form, result, cmd = runForm(d.form, msg)
	return result, cmd
}

func (d *renameDialog) value() string { return strings.TrimSpace(d.title) }

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
	return formBox(width, "Rename", "for "+d.session, d.form, "enter apply · esc cancel · empty clears the title")
}
