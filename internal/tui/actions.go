package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/template"
)

func (m *Model) maybeOpenForm() tea.Cmd {
	if !m.ready || !m.storedLoaded || m.greeted {
		return nil
	}
	m.greeted = true
	if m.form != nil || len(m.rows) > 0 || m.opts.InitialDir != "" {
		return nil
	}
	m.form = newForm(m.opts.DefaultDir, m.opts.DefaultModel, m.opts.DefaultPermissionMode)
	return textarea.Blink
}

func (m Model) openJobs() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	m.jobsModal = newJobsModal(item.displayName(), item.jobList, m.baseOutputWidth(), m.outputHeight())
	return m, nil
}

func (m Model) toggleMarkdown() (tea.Model, tea.Cmd) {
	m.showRaw = !m.showRaw
	m.status = "markdown on"
	if m.showRaw {
		m.status = "raw text"
	}
	m.rebuildOutput()
	return m, nil
}

func (m Model) resumeSelected() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if item.running() {
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	}
	meta := item.meta
	if meta.ClaudeSessionID == "" {
		stored, err := m.mgr.Meta(item.name)
		if err != nil {
			m.errText = "this session has nothing to resume from"
			return m, nil
		}
		meta = stored
	}
	m.errText = ""
	m.status = "resuming " + item.name
	return m, resumeCmd(m.mgr, meta)
}

func (m Model) archiveSelected() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if item.running() {
		m.errText = "stop the session before you archive it"
		return m, nil
	}
	m.errText = ""
	return m, archiveCmd(m.mgr, item.name, !item.archived)
}

func (m Model) openNewForm() (tea.Model, tea.Cmd) {
	m.form = newForm(m.newFormDir(), m.opts.DefaultModel, m.opts.DefaultPermissionMode)
	return m, textinputBlink()
}

// newFormDir is the directory a new session starts in: the repository root of
// the selected session group, or the directory of the control session at the
// root of a control session group. It falls back to the default directory when
// nothing is selected.
func (m Model) newFormDir() string {
	item, ok := m.selectedRow()
	if !ok {
		return m.opts.DefaultDir
	}
	if creator, ok := strings.CutPrefix(item.group, byPrefix); ok {
		for _, control := range m.rows {
			if control.name == creator {
				return control.dir
			}
		}
		return item.dir
	}
	if root, ok := strings.CutPrefix(item.group, dirPrefix); ok && root != "" {
		return root
	}
	return m.opts.DefaultDir
}

func (m Model) toggleArchived() (tea.Model, tea.Cmd) {
	m.showArchived = !m.showArchived
	m.status = "archived sessions hidden"
	if m.showArchived {
		m.status = "archived sessions shown"
	}
	m.refresh()
	m.rebuildOutput()
	return m, nil
}

func (m Model) toggleFocus() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusSidebar:
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	case focusPrompt:
		m.focus = focusOutput
		m.prompt.Blur()
		return m, nil
	case focusOutput:
		if m.diffPanel {
			m.focus = focusDiff
			return m, nil
		}
	}
	// The next pane is the sidebar, but it is skipped when it is hidden, so the
	// focus continues to the prompt.
	if m.sidebarHidden {
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	}
	m.focus = focusSidebar
	return m, nil
}

// retreatFocus is the pane Esc leaves a pane for: the sidebar, or the output
// when the sidebar is hidden.
func (m Model) retreatFocus() focusArea {
	if m.sidebarHidden {
		return focusOutput
	}
	return focusSidebar
}

func (m Model) send() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.prompt.Value())
	item, ok := m.selectedRow()
	if !ok {
		if text == "" {
			return m, nil
		}
		m.errText = "no session is selected"
		return m, nil
	}
	if text == "" {
		if item.state == session.StateBusy && len(m.queued[m.sel]) > 0 {
			m.errText = ""
			m.status = "sending now…"
			return m, interruptCmd(m.mgr, m.sel, false)
		}
		return m, nil
	}
	if !item.running() {
		m.errText = "this session is not running — press Enter to resume it"
		return m, nil
	}
	m.recordHistory(text)
	m.prompt.Reset()
	return m.dispatch(text)
}

func (m Model) stopBusy() (tea.Model, tea.Cmd, bool) {
	item, ok := m.selectedRow()
	if !ok || item.state != session.StateBusy {
		return m, nil, false
	}
	delete(m.queued, m.sel)
	m.errText = ""
	m.status = "interrupted"
	m.refresh()
	m.setContent()
	return m, interruptCmd(m.mgr, m.sel, true), true
}

func (m Model) dispatch(text string) (tea.Model, tea.Cmd) {
	if command, feed, ok := parseBang(text); ok {
		return m.runBash(command, feed)
	}
	if name, args, ok := template.ParseInvocation(text); ok {
		m.reloadTemplates()
		if tpl, found := template.Find(m.templates, name); found {
			values, missing := tpl.Fill(args)
			if len(missing) > 0 {
				m.status = "fill in /" + tpl.Name
				return m.openFields(tpl, values)
			}
			text = tpl.Expand(values)
		}
	}
	if err := m.mgr.Send(m.sel, text); err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	m.queued[m.sel] = append(m.queued[m.sel], text)
	m.refresh()
	m.setContent()
	return m, m.ensureAnimating()
}

func (m Model) askToStop() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if !item.running() {
		m.errText = "that session is not running"
		return m, nil
	}
	m.errText = ""
	m.confirm = item.name
	return m, nil
}

func (m Model) openChoice(kind settingKind) (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
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
	m.choice = newChoiceDialog(kind, item.name, current)
	m.errText = ""
	return m, nil
}

func (m Model) submitChoice() (tea.Model, tea.Cmd) {
	kind, name, value := m.choice.kind, m.choice.session, m.choice.chosen()
	m.choice = nil
	switch kind {
	case settingModel:
		return m.appliedSetting(m.mgr.SetModel(name, value), "model", value)
	case settingMode:
		return m.appliedSetting(m.mgr.SetPermissionMode(name, value), "mode", value)
	case settingEffort:
		m.errText = ""
		m.status = "resuming " + name + " with " + value + " effort"
		return m, resumeEffortCmd(m.mgr, name, value)
	}
	return m, nil
}

func (m Model) openRename() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		m.errText = "no session is selected"
		return m, nil
	}
	m.rename = newRenameDialog(item.name, item.title)
	m.errText = ""
	return m, textinput.Blink
}

func (m Model) submitRename() (tea.Model, tea.Cmd) {
	name, title := m.rename.session, m.rename.value()
	m.rename = nil
	if err := m.mgr.SetTitle(name, title); err != nil {
		m.errText = err.Error()
		return m, nil
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
	return m, nil
}

func (m Model) appliedSetting(err error, label, value string) (tea.Model, tea.Cmd) {
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = label + " → " + value
	m.refresh()
	return m, nil
}

func (m Model) openPicker() (tea.Model, tea.Cmd) {
	m.reloadTemplates()
	m.picker = newPicker(m.templates, m.templateDirs())
	m.errText = ""
	return m, textinput.Blink
}

func (m Model) openFields(tpl template.Template, values map[string]string) (tea.Model, tea.Cmd) {
	if len(tpl.Fields) == 0 {
		return m.fillPrompt(tpl.Expand(nil))
	}
	m.fields = newFieldForm(tpl, values)
	return m, textinput.Blink
}

func (m Model) fillPrompt(text string) (tea.Model, tea.Cmd) {
	m.prompt.SetValue(text)
	m.prompt.CursorEnd()
	m.focus = focusPrompt
	m.prompt.Focus()
	return m, textarea.Blink
}

func (m Model) complete() (tea.Model, tea.Cmd) {
	m.reloadTemplates()
	if names := completionNames(m.templates, m.prompt.Value()); len(names) > 0 {
		m.prompt.SetValue(names[0] + " ")
		m.prompt.CursorEnd()
		return m, nil
	}
	if m.pathPicked >= 0 {
		if next, ok := m.walkMention(1); ok {
			return next, nil
		}
	}
	if next, ok := m.completeMention(); ok {
		return next, nil
	}
	return m.toggleFocus()
}

func (m *Model) reloadTemplates() {
	m.templates = template.Load(m.templateDirs()...)
}

func (m Model) templateDirs() []string {
	dir := ""
	if item, ok := m.selectedRow(); ok {
		dir = item.dir
	}
	return template.Dirs(m.mgr.Root(), dir)
}

func (m Model) interrupt() (tea.Model, tea.Cmd) {
	if m.armedQuit {
		return m.startQuit()
	}
	m.armedQuit = true
	m.errText = ""
	m.form = nil
	m.fields = nil
	m.picker = nil
	m.help = nil
	m.questions = map[string]*questionDialog{}
	m.choice = nil
	m.rename = nil
	m.jobsModal = nil
	m.confirm = ""
	m.prompt.Reset()
	m.status = "press ctrl+c again to quit"
	return m, nil
}

func (m Model) startQuit() (tea.Model, tea.Cmd) {
	m.quitting = true
	m.status = "stopping every session…"
	mgr := m.mgr
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), session.DefaultStopGrace)
		defer cancel()
		mgr.Shutdown(ctx)
		return shutdownDoneMsg{}
	}
}
