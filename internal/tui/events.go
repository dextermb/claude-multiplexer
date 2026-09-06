package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func (m Model) handleSettings(msg settingsMsg) (tea.Model, tea.Cmd) {
	capsChanged := !sameCaps(msg.caps, m.caps)
	m.caps = msg.caps
	prev := m.layout
	m.layouts = msg.layouts
	m.activeLayout = msg.activeLayout
	m.applyLayout()
	if !capsChanged && m.layout == prev {
		return m, nil
	}
	m.rebuildOutput()
	return m, nil
}

func sameCaps(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (m Model) handleStored(msg storedMsg) (tea.Model, tea.Cmd) {
	m.stored = msg.metas
	m.storedLoaded = true
	m.refresh()
	if m.sel != "" {
		m.rebuildOutput()
	}
	return m, m.maybeOpenForm()
}

func (m Model) handleEvent(ev manager.Event) (tea.Model, tea.Cmd) {
	gap := m.lastSeq != 0 && ev.Seq != m.lastSeq+1
	m.lastSeq = ev.Seq
	if ev.Notice != "" || ev.Reload {
		return m.handleNotice(ev)
	}
	prevWidth := m.outputWidth()
	m.setPartial(ev.Session, ev.Partial)
	if !ev.Closed {
		m.todos[ev.Session] = ev.Todos
	}
	if hasPrompt(ev.Lines) {
		m.dropQueued(ev.Session)
	}
	wasBusy := m.selectedBusy()
	m.refresh()
	turnEnded := ev.Session == m.sel && wasBusy && !m.selectedBusy()

	widthChanged := ev.Session == m.sel && m.outputWidth() != prevWidth
	switch {
	case gap || widthChanged:
		m.rebuildOutput()
	case ev.Session == m.sel && len(ev.Lines) > 0:
		m.appendOutput(ev.Lines)
	case ev.Session == m.sel:
		m.setContent()
	}
	if turnEnded {
		m.resetBlockCursor()
	}
	cmds := []tea.Cmd{waitEvent(m.sub)}
	if turnEnded {
		if refresh := m.diffRefreshCmd(); refresh != nil {
			cmds = append(cmds, refresh)
		}
	}
	if cmd := m.maybeAskQuestion(ev); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if spin := m.ensureAnimating(); spin != nil {
		cmds = append(cmds, spin)
	}
	if ev.Closed {
		cmds = append(cmds, reloadStored(m.mgr))
	}
	return m, tea.Batch(cmds...)
}

// handleNotice takes a change an MCP tool made. It must not reach the normal
// path, because that path clears the streaming text of a session whose event
// carries none. See docs/mcp/notices.md.
func (m Model) handleNotice(ev manager.Event) (tea.Model, tea.Cmd) {
	if ev.Notice != "" {
		m.status = ev.Notice
	}
	m.refresh()
	cmds := []tea.Cmd{waitEvent(m.sub), m.readSettings()}
	if ev.Reload {
		cmds = append(cmds, reloadStored(m.mgr))
	}
	if ev.Session == m.sel {
		m.setContent()
	}
	return m, tea.Batch(cmds...)
}

func hasPrompt(lines []render.Line) bool {
	for _, line := range lines {
		if line.Class == render.ClassPrompt {
			return true
		}
	}
	return false
}

func (m *Model) dropQueued(name string) {
	if len(m.queued[name]) == 0 {
		return
	}
	rest := m.queued[name][1:]
	if len(rest) == 0 {
		delete(m.queued, name)
		return
	}
	m.queued[name] = rest
}

func (m *Model) maybeAskQuestion(ev manager.Event) tea.Cmd {
	if len(ev.Questions) == 0 {
		return nil
	}
	if m.questions[ev.Session] != nil {
		m.status = "another question waits for " + ev.Session
		return nil
	}
	m.questions[ev.Session] = newQuestionDialog(ev.Session, ev.Questions)
	m.status = ev.Session + " asks a question"
	return textinput.Blink
}

func (m Model) handleSpawned(msg spawnedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errText = msg.err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = "started " + msg.name
	delete(m.replays, msg.name)
	m.todos[msg.name] = m.mgr.Todos(msg.name)
	m.refresh()
	m.sel = msg.name
	m.revealSelection()
	m.rebuildOutput()
	m.focus = focusPrompt
	m.prompt.Focus()

	if m.pending != "" {
		text := m.pending
		m.pending = ""
		next, cmd := m.dispatch(text)
		return next, tea.Batch(cmd, textarea.Blink, reloadStored(m.mgr))
	}
	return m, tea.Batch(textarea.Blink, reloadStored(m.mgr))
}

func (m Model) handleSpin() (tea.Model, tea.Cmd) {
	if !m.spinning() {
		m.animating = false
		return m, nil
	}
	m.spinFrame++
	m.setContent()
	return m, spinTick()
}
