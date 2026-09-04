package tui

import (
	"fmt"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

// block names the rows of one piece of content: a prompt, one message, one tool
// result, or the output of a ! command.
type block struct {
	from int
	to   int
}

func blocks(lines []render.Line) []block {
	var out []block
	for i, line := range lines {
		if i > 0 && line.Cont {
			out[len(out)-1].to = i + 1
			continue
		}
		out = append(out, block{from: i, to: i + 1})
	}
	return out
}

func markerText(hidden int, under bool) string {
	if hidden <= 0 {
		if under {
			return "⋯ show less · enter to close"
		}
		return "⋯ show less"
	}
	if under {
		return fmt.Sprintf("⋯ %d more lines · enter to open", hidden)
	}
	if hidden == 1 {
		return "⋯ 1 more line"
	}
	return fmt.Sprintf("⋯ %d more lines", hidden)
}

func (m Model) markerRow(hidden int, under bool) string {
	if under {
		return markerCursorStyle.Width(m.outputWidth()).Render("▸ " + markerText(hidden, true))
	}
	return markerStyle.Width(m.outputWidth()).Render("  " + markerText(hidden, false))
}

// blockRows draws one block. It caps the block unless it is open, and returns
// the rows, the rows it hides, and whether it drew a marker row.
func (m Model) blockRows(index int, blk block) ([]string, int, bool) {
	rows := strings.Split(m.wrap(m.shownLines[blk.from:blk.to]), "\n")
	if m.blockCap <= 0 || len(rows) <= m.blockCap {
		return rows, 0, false
	}
	under := m.blockCursor == index
	if m.expanded[index] {
		return append(rows, m.markerRow(0, under)), 0, true
	}
	hidden := len(rows) - m.blockCap
	return append(rows[:m.blockCap:m.blockCap], m.markerRow(hidden, under)), hidden, true
}
