package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) questionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab":
		return m.toggleFocus()
	}
	result, cmd := m.questions[m.sel].Update(msg)
	switch result {
	case formCancelled:
		delete(m.questions, m.sel)
		m.status = "question dismissed"
		m.setContent()
		return m, nil
	case formSubmitted:
		return m.submitQuestion()
	}
	return m, cmd
}

func (m Model) submitQuestion() (tea.Model, tea.Cmd) {
	name := m.sel
	text := m.questions[name].answer()
	delete(m.questions, name)
	if err := m.mgr.Send(name, text); err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = "answered"
	m.queued[name] = append(m.queued[name], text)
	m.refresh()
	m.setContent()
	m.focus = focusPrompt
	m.prompt.Focus()
	return m, tea.Batch(m.ensureAnimating(), textarea.Blink)
}

func looksDropped(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && len(msg.Runes) > 1 &&
		strings.ContainsRune(string(msg.Runes), '/')
}

func (m Model) handlePaste(raw string) (tea.Model, tea.Cmd) {
	if m.quitting || m.confirm != "" {
		return m, nil
	}
	m.armedQuit = false
	text, paths := dropText(raw)
	if text == "" {
		return m, nil
	}
	if m.form != nil {
		m.form.insert(text, paths)
		return m, nil
	}
	if paths {
		text += " "
	}
	if m.focus != focusPrompt {
		m.focus = focusPrompt
		m.prompt.Focus()
	}
	value := m.prompt.Value()
	if value != "" && !strings.HasSuffix(value, " ") && !strings.HasSuffix(value, "\n") {
		m.prompt.InsertString(" ")
	}
	m.prompt.InsertString(text)
	return m, textarea.Blink
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, nil
	}
	if msg.String() == "ctrl+c" {
		return m.interrupt()
	}
	m.armedQuit = false
	if m.help != nil {
		open, cmd := m.help.Update(msg)
		if !open {
			m.help = nil
		}
		return m, cmd
	}
	if q := m.questions[m.sel]; q != nil && m.focus == focusOutput {
		return m.questionKey(msg)
	}
	if m.choice != nil {
		return m.choiceKey(msg)
	}
	if m.rename != nil {
		return m.renameKey(msg)
	}
	if m.layoutSwitch != nil {
		return m.layoutSwitchKey(msg)
	}
	if m.jobsModal != nil {
		return m.jobsModalKey(msg)
	}
	if m.picker != nil {
		return m.pickerKey(msg)
	}
	if m.fields != nil {
		return m.fieldsKey(msg)
	}
	if m.form != nil {
		result, cmd := m.form.Update(msg)
		switch result {
		case formCancelled:
			m.form = nil
			return m, nil
		case formSubmitted:
			spec := m.form.spec()
			m.pending = m.form.firstPrompt()
			m.form = nil
			return m, spawnCmd(m.mgr, spec)
		}
		return m, cmd
	}
	if m.confirm != "" {
		switch msg.String() {
		case "y", "enter":
			name := m.confirm
			m.confirm = ""
			return m, stopCmd(m.mgr, name)
		default:
			m.confirm = ""
			return m, nil
		}
	}

	if m.seq != nil {
		return m.resolveSequence(msg)
	}
	if target, ok := sequenceTarget(msg.String(), m.focus == focusPrompt, m.diffPanel); ok {
		return m.startSequence(target)
	}

	if m.focus == focusDiff {
		return m.diffKey(msg)
	}

	switch msg.String() {
	case "ctrl+p":
		return m.openPicker()
	case "ctrl+n":
		return m.openNewForm()
	case "ctrl+t":
		m.mouseOn = !m.mouseOn
		if m.mouseOn {
			return m, tea.EnableMouseCellMotion
		}
		return m, tea.DisableMouse
	case "tab":
		if m.focus == focusPrompt {
			return m.complete()
		}
		return m.toggleFocus()
	case "pgup":
		m.output.ViewUp()
		return m, nil
	case "pgdown":
		m.output.ViewDown()
		return m, nil
	}

	if m.focus != focusPrompt {
		if looksDropped(msg) {
			return m.handlePaste(string(msg.Runes))
		}
		switch msg.String() {
		case "n":
			return m.openNewForm()
		case "t":
			return m.openPicker()
		case "?":
			m.help = newHelp()
			return m, textinput.Blink
		case "q":
			return m.startQuit()
		}
	}

	switch m.focus {
	case focusPrompt:
		return m.promptKey(msg)
	case focusOutput:
		return m.outputKey(msg)
	}
	return m.sidebarKey(msg)
}

func (m Model) promptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if next, cmd, ok := m.stopBusy(); ok {
			return next, cmd
		}
		m.focus = m.retreatFocus()
		m.prompt.Blur()
		return m, nil
	case "enter":
		return m.send()
	case "ctrl+j":
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, cmd
	}
	if next, ok := m.historyKey(msg); ok {
		return next, nil
	}
	if next, cmd, ok := m.mentionKey(msg); ok {
		return next, cmd
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

// historyKey walks the submitted prompts with the up and down arrow keys; see docs/tui/input.md.
func (m Model) historyKey(msg tea.KeyMsg) (Model, bool) {
	switch msg.String() {
	case "up":
		if m.prompt.Line() != 0 || m.histIdx == 0 {
			return m, false
		}
		if m.histIdx == len(m.history) {
			m.histDraft = m.prompt.Value()
		}
		m.histIdx--
		m.prompt.SetValue(m.history[m.histIdx])
		return m, true
	case "down":
		if m.prompt.Line() != m.prompt.LineCount()-1 || m.histIdx >= len(m.history) {
			return m, false
		}
		m.histIdx++
		if m.histIdx == len(m.history) {
			m.prompt.SetValue(m.histDraft)
		} else {
			m.prompt.SetValue(m.history[m.histIdx])
		}
		return m, true
	}
	return m, false
}

// recordHistory keeps a submitted prompt for later recall, without a consecutive duplicate.
func (m *Model) recordHistory(text string) {
	if text != "" {
		if n := len(m.history); n == 0 || m.history[n-1] != text {
			m.history = append(m.history, text)
		}
	}
	m.histIdx = len(m.history)
	m.histDraft = ""
}

func (m Model) jobsModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	open, cmd := m.jobsModal.Update(msg)
	if !open {
		m.jobsModal = nil
	}
	return m, cmd
}

func (m Model) outputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.clearSelection()
	switch msg.String() {
	case "esc":
		if next, cmd, ok := m.stopBusy(); ok {
			return next, cmd
		}
		m.focus = m.retreatFocus()
		return m, nil
	case "enter":
		if m.blockCursor >= 0 {
			m.toggleBlock(m.blockCursor)
			return m, nil
		}
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	case " ":
		if m.blockCursor >= 0 {
			m.toggleBlock(m.blockCursor)
		}
		return m, nil
	case "]":
		m.moveBlockCursor(1)
	case "[":
		m.moveBlockCursor(-1)
	case "i":
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	case "up", "k":
		m.output.LineUp(1)
	case "down", "j":
		m.output.LineDown(1)
	case "u", "ctrl+u":
		m.output.HalfViewUp()
	case "d", "ctrl+d":
		m.output.HalfViewDown()
	case "g", "home":
		m.output.GotoTop()
	case "G", "end":
		m.output.GotoBottom()
	}
	return m, nil
}

func (m Model) sidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		return m.move(-1)
	case "down", "j":
		return m.move(1)
	case "enter", "i":
		if item, ok := m.selectedRow(); ok && !item.running() {
			return m.resumeSelected()
		}
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	}
	return m, nil
}

func (m Model) choiceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.choice.Update(msg)
	switch result {
	case formCancelled:
		m.choice = nil
		return m, nil
	case formSubmitted:
		return m.submitChoice()
	}
	return m, cmd
}

func (m Model) renameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.rename.Update(msg)
	switch result {
	case formCancelled:
		m.rename = nil
		return m, nil
	case formSubmitted:
		return m.submitRename()
	}
	return m, cmd
}

func (m Model) layoutSwitchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.layoutSwitch.Update(msg)
	switch result {
	case formCancelled:
		m.layoutSwitch = nil
		return m, nil
	case formSubmitted:
		return m.submitLayoutSwitch()
	}
	return m, cmd
}

func (m Model) pickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.picker.Update(msg)
	switch result {
	case pickerCancelled:
		m.picker = nil
		return m, nil
	case pickerChosen:
		tpl, ok := m.picker.selected()
		m.picker = nil
		if !ok {
			return m, nil
		}
		return m.openFields(tpl, nil)
	}
	return m, cmd
}

func (m Model) fieldsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.fields.Update(msg)
	switch result {
	case formCancelled:
		m.fields = nil
		return m, nil
	case formSubmitted:
		text := m.fields.prompt()
		m.fields = nil
		return m.fillPrompt(text)
	}
	return m, cmd
}
