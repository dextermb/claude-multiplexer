package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.form != nil || m.confirm != "" || m.quitting || m.questions[m.sel] != nil || m.choice != nil || m.rename != nil || m.jobsModal != nil {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if !m.sidebarHidden && msg.X < sidebarWidth {
			if msg.Button == tea.MouseButtonWheelUp {
				return m.move(-1)
			}
			return m.move(1)
		}
		if m.diffPanel && msg.X >= m.width-m.diffPanelWidth() {
			if msg.Button == tea.MouseButtonWheelUp {
				m.diffScroll -= 3
			} else {
				m.diffScroll += 3
			}
			m.clampDiffScroll()
			return m, nil
		}
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		return m, cmd
	case tea.MouseButtonLeft:
		return m.handleLeftMouse(msg)
	}
	return m, nil
}

func (m Model) handleLeftMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Y >= m.bodyHeight() && msg.Y < m.bodyHeight()+m.promptHeight() {
			m.clearSelection()
			m.focus = focusPrompt
			m.prompt.Focus()
			return m, textarea.Blink
		}
		if p, ok := m.outputPos(msg.X, msg.Y); ok {
			if index, hit := m.blockAtRow(p.line); hit {
				m.clearSelection()
				m.focus = focusOutput
				m.prompt.Blur()
				m.toggleBlock(index)
				return m, nil
			}
			m.focus = focusOutput
			m.prompt.Blur()
			m.selection = selRange{active: true, dragging: true, anchor: p, cursor: p}
			m.setContent()
			return m, nil
		}
		if m.sidebarHidden || msg.X >= sidebarWidth {
			m.clearSelection()
			m.focus = focusOutput
			m.prompt.Blur()
			return m, nil
		}
		m.clearSelection()
		index := m.listOffset + msg.Y - titleHeight
		if msg.Y >= titleHeight && index >= 0 && index < len(m.lines) {
			line := m.lines[index]
			m.focus = focusSidebar
			m.prompt.Blur()
			if line.header() {
				m.setFold(m.groups[line.group].key, !m.groups[line.group].folded)
				return m, nil
			}
			m.sel = m.rows[line.row].name
			m.rebuildOutput()
		}
		return m, nil
	case tea.MouseActionMotion:
		if m.selection.dragging {
			if p, ok := m.outputPos(msg.X, msg.Y); ok {
				m.selection.cursor = p
				m.setContent()
			}
		}
		return m, nil
	case tea.MouseActionRelease:
		if m.selection.dragging {
			m.selection.dragging = false
			if p, ok := m.outputPos(msg.X, msg.Y); ok {
				m.selection.cursor = p
			}
			if m.selection.empty() {
				m.selection.active = false
			}
			m.copySelection()
			m.setContent()
		}
		return m, nil
	}
	return m, nil
}

// blockAtRow names the block whose marker row is the given row of the pane.
func (m Model) blockAtRow(row int) (int, bool) {
	for index, at := range m.markerAt {
		if at == row {
			return index, true
		}
	}
	return 0, false
}

func (m Model) outputPos(x, y int) (pos, bool) {
	if x < m.leftWidth()+gutterWidth {
		return pos{}, false
	}
	row := y - barHeight
	if row < 0 || row >= m.outputHeight() {
		return pos{}, false
	}
	return pos{line: m.output.YOffset + row, col: x - m.leftWidth() - gutterWidth}, true
}

func (m *Model) clearSelection() {
	if !m.selection.active {
		return
	}
	m.selection = selRange{}
	m.setContent()
}

func (m *Model) copySelection() {
	if m.selection.empty() {
		return
	}
	a, b := m.selection.bounds()
	text := selectedText(plainLines(m.content), a, b)
	if strings.TrimSpace(text) == "" {
		return
	}
	if err := copyToClipboard(text); err != nil {
		m.errText = "could not copy: " + err.Error()
		return
	}
	m.errText = ""
	m.status = "copied " + plural(strings.Count(text, "\n")+1, "line")
}

func (m Model) highlight(content string) string {
	if !m.selection.active || m.selection.empty() {
		return content
	}
	a, b := m.selection.bounds()
	styled := strings.Split(content, "\n")
	plain := plainLines(content)
	for i := a.line; i <= b.line && i < len(styled) && i < len(plain); i++ {
		if i < 0 {
			continue
		}
		runes := []rune(plain[i])
		from, to := 0, len(runes)
		if i == a.line {
			from = a.col
		}
		if i == b.line {
			to = b.col
		}
		if from < 0 {
			from = 0
		}
		if from > len(runes) {
			from = len(runes)
		}
		if to > len(runes) {
			to = len(runes)
		}
		if to < from {
			to = from
		}
		styled[i] = string(runes[:from]) + selectionStyle.Render(string(runes[from:to])) + string(runes[to:])
	}
	return strings.Join(styled, "\n")
}
