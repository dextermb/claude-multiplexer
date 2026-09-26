package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

// toolPairs joins each tool call in the pane to its result, which arrives in a
// later event; see docs/tui/output.md.
type toolPairs struct {
	call        map[string]int
	result      map[string]int
	lastTurnEnd int
	row         map[string]int
	folded      map[int]string
}

var foldedTools = map[string]bool{
	"Read": true, "Edit": true, "Write": true, "MultiEdit": true, "Grep": true, "Glob": true,
}

var foundFiles = regexp.MustCompile(`^Found (\d+) files?`)

func (p *toolPairs) reset() {
	p.call = make(map[string]int)
	p.result = make(map[string]int)
	p.lastTurnEnd = -1
	p.clearRows()
}

func (p *toolPairs) clearRows() {
	p.row = make(map[string]int)
	p.folded = make(map[int]string)
}

func (p *toolPairs) index(lines []render.Line, from int) {
	if p.call == nil {
		p.reset()
	}
	for i := from; i < len(lines); i++ {
		switch line := lines[i]; {
		case line.Class == render.ClassResult:
			p.lastTurnEnd = i
		case line.Tool == "":
		case line.Class == render.ClassToolUse:
			p.call[line.Tool] = i
			delete(p.result, line.Tool)
		case line.Class == render.ClassToolResult:
			p.result[line.Tool] = i
		}
	}
}

type toolBlockKind uint8

const (
	plainBlock toolBlockKind = iota
	callBlock
	pairedResult
	foldedResult
)

func (m Model) toolBlock(blk block) (string, toolBlockKind) {
	line := m.shownLines[blk.from]
	if m.showRaw || line.Tool == "" || blk.to-blk.from != 1 {
		return "", plainBlock
	}
	switch line.Class {
	case render.ClassToolUse:
		if m.tools.call[line.Tool] == blk.from {
			return line.Tool, callBlock
		}
	case render.ClassToolResult:
		if ci, ok := m.tools.call[line.Tool]; ok && ci < blk.from && m.tools.result[line.Tool] == blk.from {
			if m.folds(line.Tool) {
				return line.Tool, foldedResult
			}
			return line.Tool, pairedResult
		}
	}
	return "", plainBlock
}

func (m Model) folds(id string) bool {
	ri, ok := m.tools.result[id]
	if !ok || isErrorResult(m.shownLines[ri].Text) {
		return false
	}
	return foldedTools[toolName(m.shownLines[m.tools.call[id]].Text)]
}

// callBlockRows draws a call as one row. A folded result registers the call row
// as its marker, and draws its body under the call when it is open.
func (m *Model) callBlockRows(id string, row int, blockAt map[int]int) []string {
	m.tools.row[id] = row
	if !m.folds(id) {
		return []string{m.callRowView(id, false)}
	}
	index := blockAt[m.tools.result[id]]
	m.tools.folded[index] = id
	m.blockStart[index] = row
	m.capped = append(m.capped, index)
	under := m.blockCursor == index
	if !m.expanded[index] {
		m.markerAt[index] = row
		m.hiddenRows[index] = len(m.foldBody(id))
		return []string{m.callRowView(id, under)}
	}
	rows := append([]string{m.callRowView(id, false)}, m.foldBody(id)...)
	rows = append(rows, m.markerRow(0, under))
	m.markerAt[index] = row + len(rows) - 1
	m.hiddenRows[index] = 0
	return rows
}

// attachResult takes a result whose call is drawn in an earlier pass: it draws
// the call row again with the note, and a folded result marks that row.
func (m *Model) attachResult(index int, id string, folded bool) {
	row, ok := m.tools.row[id]
	if !ok {
		return
	}
	if folded {
		m.tools.folded[index] = id
		m.blockStart[index] = row
		m.markerAt[index] = row
		m.hiddenRows[index] = len(m.foldBody(id))
		m.capped = append(m.capped, index)
	}
	m.setOutputRow(row, m.callRowView(id, folded && m.blockCursor == index))
}

func (m *Model) setOutputRow(row int, text string) {
	rows := strings.Split(m.outputText, "\n")
	if row < 0 || row >= len(rows) {
		return
	}
	rows[row] = text
	m.outputText = strings.Join(rows, "\n")
}

// tickCalls draws again the row of each call that has no result, so its timer
// moves and a call left behind by a stopped turn loses its timer.
func (m *Model) tickCalls() {
	for id, row := range m.tools.row {
		if _, done := m.tools.result[id]; !done {
			m.setOutputRow(row, m.callRowView(id, false))
		}
	}
}

func (m Model) foldBody(id string) []string {
	ri := m.tools.result[id]
	return strings.Split(m.wrap(m.shownLines[ri:ri+1]), "\n")
}

func (m Model) callRowView(id string, under bool) string {
	line := m.shownLines[m.tools.call[id]]
	width := m.textWidth()
	name, input := splitToolLine(line.Text)
	note, noteStyle := m.callNote(id)
	head := "→ " + name
	gap := strings.Repeat(" ", max(toolNameWidth-len([]rune(name)), 0)+1)
	room := width - len([]rune(head)) - len(gap)
	if note != "" {
		room -= lipgloss.Width(note) + 2
	}
	input = truncate(input, max(room, 0))
	fill := strings.Repeat(" ", max(width-len([]rune(head))-len(gap)-len([]rune(input))-lipgloss.Width(note), 0))
	var row string
	if under {
		row = invertStyle.Render("→ " + strings.ToUpper(name) + gap + input + fill + note)
	} else {
		row = toolArrowStyle.Render("→") + " " + labelStyle.Render(name) + gap +
			toolArgsStyle.Render(input) + fill + noteStyle.Render(note)
	}
	if m.showAge {
		row = m.withAge(row, line.At, time.Now())
	}
	return row
}

func (m Model) callNote(id string) (string, lipgloss.Style) {
	ci := m.tools.call[id]
	call := m.shownLines[ci]
	ri, done := m.tools.result[id]
	if !done {
		if m.selectedBusy() && ci > m.tools.lastTurnEnd {
			clock := ""
			if !call.At.IsZero() {
				clock = " " + elapsed(time.Since(call.At))
			}
			return "■ running" + clock, fgStyle(colWarning)
		}
		return "", toolArrowStyle
	}
	text := m.shownLines[ri].Text
	if isErrorResult(text) {
		return "× error", fgStyle(colDanger)
	}
	switch toolName(call.Text) {
	case "Read":
		return countOf(len(resultRows(text)), "line"), toolArrowStyle
	case "Glob":
		return countOf(len(resultRows(text)), "file"), toolArrowStyle
	case "Grep":
		rows := resultRows(text)
		if len(rows) > 0 {
			if match := foundFiles.FindStringSubmatch(rows[0]); match != nil {
				n, _ := strconv.Atoi(match[1])
				return countOf(n, "file"), toolArrowStyle
			}
		}
		return countOf(len(rows), "line"), toolArrowStyle
	case "Edit", "Write", "MultiEdit":
		return call.Note, toolArrowStyle
	}
	return "", toolArrowStyle
}

func resultRows(text string) []string {
	body := strings.TrimPrefix(text, "← ")
	if body == "result" || strings.HasPrefix(body, "No files found") || strings.HasPrefix(body, "No matches found") {
		return nil
	}
	return strings.Split(strings.TrimRight(body, "\n"), "\n")
}

func isErrorResult(text string) bool {
	return strings.HasPrefix(text, "←!")
}

func toolName(text string) string {
	name, _ := splitToolLine(text)
	return name
}

func splitToolLine(text string) (string, string) {
	rest := strings.TrimPrefix(text, "→ ")
	name, input, _ := strings.Cut(rest, " ")
	return name, input
}

func countOf(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
