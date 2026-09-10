package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ageRefresh is how often the output pane redraws to keep a relative age
// current while the age column is on. See docs/tui/output.md.
const ageRefresh = 10 * time.Second

// ageWidth is the field the age sits in, and ageGap the space before it. Their
// sum is the rail the age column takes from the right of the output pane.
const (
	ageWidth = 4
	ageGap   = 2
	ageRail  = ageWidth + ageGap
)

type ageTickMsg struct{}

func ageTick() tea.Cmd {
	return tea.Tick(ageRefresh, func(time.Time) tea.Msg { return ageTickMsg{} })
}

// handleAgeTick redraws the pane with fresh ages, then schedules the next tick.
// It stops the loop when the age column turns off.
func (m Model) handleAgeTick() (tea.Model, tea.Cmd) {
	if !m.showAge {
		m.ageTicking = false
		return m, nil
	}
	m.redrawBlocks()
	m.setContent()
	return m, ageTick()
}

// textWidth is the width a block wraps to. With the age column on, it leaves the
// rail free at the right so the age never collides with the text.
func (m Model) textWidth() int {
	width := m.outputWidth()
	if m.showAge {
		width -= ageRail
		if width < 10 {
			return 10
		}
	}
	return width
}

// ageString gives the short relative age of an event: "now" under one minute,
// then minutes, hours, and days.
func ageString(now, at time.Time) string {
	d := now.Sub(at)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	}
}

// withAge places the relative age of a block at the right edge of its first
// row. It pads the row to the text width, then adds the gap and the age.
func (m Model) withAge(row string, at, now time.Time) string {
	if at.IsZero() {
		return row
	}
	pad := max(m.textWidth()-lipgloss.Width(row), 0)
	age := ageString(now, at)
	if len(age) < ageWidth {
		age = strings.Repeat(" ", ageWidth-len(age)) + age
	}
	return row + strings.Repeat(" ", pad+ageGap) + ageStyle.Render(age)
}
