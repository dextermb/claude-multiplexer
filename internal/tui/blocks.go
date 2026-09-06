package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func (m *Model) rebuildOutput() {
	m.selection = selRange{}
	m.expanded = make(map[int]bool)
	m.blockCursor = -1
	if m.sel == "" {
		m.outputText = ""
		m.shownLines = nil
		m.clearBlocks()
		m.output.SetContent("")
		return
	}
	lines := m.linesFor(m.sel)
	m.output.Width = m.outputWidth()
	m.shownLines = append([]render.Line(nil), lines...)
	m.redrawBlocks()
	m.resetBlockCursor()
	m.setContent()
	m.output.GotoBottom()
}

func (m *Model) appendOutput(lines []render.Line) {
	from := len(m.shownLines)
	atNewest := m.blockCursor < 0 || (len(m.capped) > 0 && m.blockCursor == m.capped[len(m.capped)-1])
	m.shownLines = append(m.shownLines, lines...)
	// An event brings whole blocks, so the first line it brings starts one.
	m.shownLines[from].Cont = false
	chunk := m.drawBlocks(from)
	if m.outputText == "" {
		m.outputText = chunk
	} else if chunk != "" {
		m.outputText += "\n" + chunk
	}
	if atNewest {
		m.resetBlockCursor()
	}
	m.setContent()
}

func (m *Model) clearBlocks() {
	m.capped = nil
	m.markerAt = make(map[int]int)
	m.blockStart = make(map[int]int)
	m.hiddenRows = make(map[int]int)
}

// redrawBlocks draws every block again, keeping which blocks are open. It runs
// when a block opens or closes, because that moves every row below it.
func (m *Model) redrawBlocks() {
	m.clearBlocks()
	m.outputText = m.drawBlocks(0)
}

// drawBlocks draws the blocks that start at line from or after it, and records
// where each block and each marker row landed.
func (m *Model) drawBlocks(from int) string {
	row := 0
	if from > 0 {
		row = rowCount(m.outputText)
	}
	var parts []string
	for index, blk := range blocks(m.shownLines) {
		if blk.from < from {
			continue
		}
		rows, hidden, marked := m.blockRows(index, blk)
		m.blockStart[index] = row
		if marked {
			m.capped = append(m.capped, index)
			m.markerAt[index] = row + len(rows) - 1
			m.hiddenRows[index] = hidden
		}
		row += len(rows)
		parts = append(parts, strings.Join(rows, "\n"))
	}
	return strings.Join(parts, "\n")
}

func rowCount(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

// setBlockCursor moves the cursor and draws the two marker rows it changed, so
// a move does not draw the whole pane again.
func (m *Model) setBlockCursor(index int) {
	was := m.blockCursor
	if was == index {
		return
	}
	m.blockCursor = index
	rows := strings.Split(m.outputText, "\n")
	for _, at := range []int{was, index} {
		row, ok := m.markerAt[at]
		if !ok || row >= len(rows) {
			continue
		}
		rows[row] = m.markerRow(m.hiddenRows[at], at == index)
	}
	m.outputText = strings.Join(rows, "\n")
	m.setContent()
}

// resetBlockCursor puts the cursor on the newest capped block; see
// docs/tui/output.md.
func (m *Model) resetBlockCursor() {
	if len(m.capped) == 0 {
		m.blockCursor = -1
		return
	}
	m.setBlockCursor(m.capped[len(m.capped)-1])
}

func (m *Model) moveBlockCursor(delta int) {
	if len(m.capped) == 0 {
		return
	}
	at := 0
	for i, index := range m.capped {
		if index == m.blockCursor {
			at = i
		}
	}
	at += delta
	if at < 0 {
		at = 0
	}
	if at >= len(m.capped) {
		at = len(m.capped) - 1
	}
	m.setBlockCursor(m.capped[at])
	m.showBlock(m.capped[at])
}

// showBlock scrolls the pane only when the block is out of sight.
func (m *Model) showBlock(index int) {
	top, bottom := m.blockStart[index], m.markerAt[index]
	switch {
	case top < m.output.YOffset:
		m.output.SetYOffset(top)
	case bottom >= m.output.YOffset+m.outputHeight():
		m.output.SetYOffset(bottom - m.outputHeight() + 1)
	}
}

// toggleBlock opens or closes a block, and holds its first row where it was, so
// the text under your eyes does not jump.
func (m *Model) toggleBlock(index int) {
	anchor := m.blockStart[index] - m.output.YOffset
	m.expanded[index] = !m.expanded[index]
	m.blockCursor = index
	m.redrawBlocks()
	m.setContent()
	m.output.SetYOffset(m.blockStart[index] - anchor)
}

func (m *Model) setPartial(name, text string) {
	if text == "" {
		delete(m.partials, name)
		return
	}
	m.partials[name] = text
}

func (m *Model) setContent() {
	atBottom := m.output.AtBottom()
	m.content = m.outputText + m.liveView()
	m.output.SetContent(m.highlight(m.content))
	if atBottom {
		m.output.GotoBottom()
	}
}

func (m Model) liveView() string {
	var parts []string
	if partial := m.partials[m.sel]; partial != "" {
		parts = append(parts, classStyle(render.ClassText).Width(m.outputWidth()).Render(partial+cursorMark))
	}
	for _, text := range m.queued[m.sel] {
		parts = append(parts, m.wrap(render.PromptLines(text)))
	}
	if m.thinkingSelected() {
		parts = append(parts, spinnerStyle.Render(spinnerFrame(m.spinFrame)+" thinking…"))
	}
	if len(parts) == 0 {
		return ""
	}
	body := strings.Join(parts, "\n")
	if m.outputText == "" {
		return body
	}
	return "\n" + body
}

func (m *Model) linesFor(name string) []render.Line {
	item, ok := m.selectedRow()
	if ok && item.name == name && !item.live {
		if cached, seen := m.replays[name]; seen {
			return cached
		}
		lines := m.mgr.Replay(name)
		m.replays[name] = lines
		m.todos[name] = m.mgr.Todos(name)
		return lines
	}
	return m.mgr.Lines(name)
}

func (m Model) wrap(lines []render.Line) string {
	width := m.outputWidth()
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line.Text == "" {
			wrapped = append(wrapped, "")
			continue
		}
		if line.Class == render.ClassText && !m.showRaw {
			wrapped = append(wrapped, m.md.Render(line.Text, width))
			continue
		}
		text := line.Text
		if line.Class == render.ClassPrompt && !m.showRaw {
			styled := inlineEmphasis(text, classStyle(render.ClassPrompt))
			wrapped = append(wrapped, lipgloss.NewStyle().Width(width).Render(styled))
			continue
		}
		wrapped = append(wrapped, classStyle(line.Class).Width(width).Render(text))
	}
	return strings.Join(wrapped, "\n")
}
