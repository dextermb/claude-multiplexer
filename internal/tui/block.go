package tui

import (
	"fmt"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
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

func markerText(hidden int) string {
	if hidden <= 0 {
		return "⋯ show less"
	}
	if hidden == 1 {
		return "⋯ 1 more line"
	}
	return fmt.Sprintf("⋯ %d more lines", hidden)
}

func (m Model) markerRow(hidden int, under bool) string {
	if under {
		return markerCursorStyle.Width(m.outputWidth()).Render("▸ " + markerText(hidden))
	}
	return markerStyle.Width(m.outputWidth()).Render("  " + markerText(hidden))
}

// capFor gives the resolved cap for a block of this class: -1 never caps, 0
// draws only the marker, and a positive number draws that many rows.
func (m Model) capFor(c render.Class) int {
	if cap, ok := m.caps[render.BucketFor(c)]; ok {
		return cap
	}
	return config.DefaultBlockCap
}

// blockRows draws one block. It caps the block by its bucket unless it is open,
// and returns the rows, the rows it hides, and whether it drew a marker row.
func (m Model) blockRows(index int, blk block) ([]string, int, bool) {
	cap := m.capFor(m.shownLines[blk.from].Class)
	rows := strings.Split(m.wrap(m.shownLines[blk.from:blk.to]), "\n")
	if cap < 0 || (cap > 0 && len(rows) <= cap) {
		return rows, 0, false
	}
	under := m.blockCursor == index
	if m.expanded[index] {
		return append(rows, m.markerRow(0, under)), 0, true
	}
	hidden := len(rows) - cap
	return append(rows[:cap:cap], m.markerRow(hidden, under)), hidden, true
}
