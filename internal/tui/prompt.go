package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) promptHeight() int {
	return promptHintRows + m.prompt.Height()
}

func (m *Model) syncPromptHeight() {
	rows := wrappedRows(m.prompt.Value(), m.prompt.Width())
	if rows < promptRowsMin {
		rows = promptRowsMin
	}
	if rows > promptRowsMax {
		rows = promptRowsMax
	}
	if rows == m.prompt.Height() {
		return
	}
	atBottom := m.output.AtBottom()
	m.prompt.SetHeight(rows)
	m.output.Height = m.outputHeight()
	if atBottom {
		m.output.GotoBottom()
	}
}

// wrappedRows counts the display rows of text at width, the way the textarea
// wraps it; see docs/tui/input.md.
func wrappedRows(text string, width int) int {
	if width <= 0 {
		return promptRowsMin
	}
	rows := 0
	for _, line := range strings.Split(text, "\n") {
		rows += lineRows(line, width)
		if rows >= promptRowsMax {
			return promptRowsMax
		}
	}
	return rows
}

func lineRows(line string, width int) int {
	rows, used := 1, 0
	for _, chunk := range wrapChunks(line) {
		size := lipgloss.Width(chunk)
		if used > 0 && used+size > width {
			rows++
			used = 0
		}
		for size > width {
			rows++
			size -= width
		}
		used += size
	}
	return rows
}

// wrapChunks splits a line into words, each one with the spaces that follow it.
func wrapChunks(line string) []string {
	var chunks []string
	var current strings.Builder
	spacing := false
	for _, r := range line {
		if r == ' ' || r == '\t' {
			spacing = true
			current.WriteRune(r)
			continue
		}
		if spacing {
			chunks = append(chunks, current.String())
			current.Reset()
			spacing = false
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}
