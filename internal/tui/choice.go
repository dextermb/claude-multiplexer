package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

type settingKind int

const (
	settingModel settingKind = iota
	settingMode
	settingEffort
)

var (
	modelChoices = []string{"opus", "sonnet", "haiku"}
	modeChoices  = []string{"acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "plan"}
)

type choiceDialog struct {
	kind    settingKind
	session string
	title   string
	options []string
	value   string
	form    *huh.Form
}

// newChoiceDialog asks for one setting in a huh select. The current value
// carries a filled mark, and the cursor starts on it.
func newChoiceDialog(kind settingKind, name, current string) *choiceDialog {
	d := &choiceDialog{kind: kind, session: name, value: current}
	note := ""
	switch kind {
	case settingModel:
		d.title, d.options = "Model", modelChoices
	case settingMode:
		d.title, d.options = "Permission mode", modeChoices
	case settingEffort:
		d.title, d.options = "Effort", session.EffortLevels
		note = "resumes the session — Claude Code has no live effort switch"
	}
	options := make([]huh.Option[string], len(d.options))
	for i, option := range d.options {
		mark := "( ) "
		if option == current {
			mark = "(●) "
		}
		options[i] = huh.NewOption(mark+option, option)
	}
	field := huh.NewSelect[string]().Options(options...).Value(&d.value)
	if note != "" {
		field.Description(note)
	}
	d.form = newHuhForm(huh.NewGroup(field))
	d.form.Init()
	return d
}

func (d *choiceDialog) Update(msg tea.Msg) (formResult, tea.Cmd) {
	var result formResult
	var cmd tea.Cmd
	d.form, result, cmd = runForm(d.form, msg)
	return result, cmd
}

func (d *choiceDialog) chosen() string { return d.value }

func (d *choiceDialog) region() modalRegion      { return modalPane }
func (d *choiceDialog) view(width, _ int) string { return d.View(width) }

func (d *choiceDialog) update(m *Model, msg tea.Msg) (modal, tea.Cmd) {
	result, cmd := d.Update(msg)
	return applyForm(d, m, result, cmd, d.submit)
}

func (d *choiceDialog) submit(m *Model) (modal, tea.Cmd) {
	name, value := d.session, d.chosen()
	switch d.kind {
	case settingModel:
		return nil, m.applySetting(m.mgr.SetModel(name, value), "model", value)
	case settingMode:
		return nil, m.applySetting(m.mgr.SetPermissionMode(name, value), "mode", value)
	case settingEffort:
		m.errText = ""
		m.status = "resuming " + name + " with " + value + " effort"
		return nil, resumeEffortCmd(m.mgr, name, value)
	}
	return nil, nil
}

func (m Model) openChoice(kind settingKind) (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if item.readOnly {
		m.errText = readOnlyStatus
		return m, nil
	}
	if !item.running() {
		m.errText = "start the session before you change it"
		return m, nil
	}
	current := item.model
	switch kind {
	case settingMode:
		current = item.mode
	case settingEffort:
		current = item.effort
	}
	m.modal = newChoiceDialog(kind, item.name, current)
	m.errText = ""
	return m, nil
}

func (d *choiceDialog) View(width int) string {
	return formBox(width, d.title, "for "+d.session, d.form, "↑↓ move · enter apply · esc cancel")
}
