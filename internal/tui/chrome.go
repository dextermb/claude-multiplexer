package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

var (
	ruleStyle = fgStyle(colBorder)

	focusLabelStyle = invertStyle.Transform(strings.ToUpper)
)

func rule(n int) string {
	if n <= 0 {
		return ""
	}
	return ruleStyle.Render(strings.Repeat("─", n))
}

// ruleLabel is a pane head: a label set in a rule, inverted when the pane holds
// the focus. See docs/tui.md.
func ruleLabel(label string, width int, focused bool) string {
	style := labelStyle
	if focused {
		style = focusLabelStyle
	}
	text := style.Render(truncate(" "+label+" ", maxInt(1, width-2)))
	return rule(1) + text + rule(width-1-lipgloss.Width(text))
}

func isRule(line string) bool {
	return strings.HasPrefix(ansi.Strip(line), "─")
}

// withGutter puts the one-column gutter left of a pane. The gutter carries the
// rule on the first row when the pane starts with a rule, so the rule runs
// unbroken to the border on its left.
func withGutter(block string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lead := " "
		if i == 0 && isRule(line) {
			lead = rule(1)
		}
		lines[i] = lead + line
	}
	return strings.Join(lines, "\n")
}

// borderColumn draws a vertical rule beside a block of lines: a junction where a
// rule meets it, and the accent colour when its pane holds the focus.
func borderColumn(lines []string, focused bool) []string {
	style := ruleStyle
	if focused {
		style = fgStyle(colAccent)
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		switch {
		case i == 0:
			out[i] = style.Render("┬")
		case isRule(line):
			out[i] = style.Render("├")
		default:
			out[i] = style.Render("│")
		}
	}
	return out
}

// sideColumn draws a side panel: a left border with junctions, then the lines,
// each padded one column in from the border unless it is a rule.
func sideColumn(lines []string, width, height int, focused bool) string {
	inner := maxInt(1, width-1)
	rows := make([]string, height)
	for i := range rows {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		if !isRule(line) {
			line = " " + line
		}
		rows[i] = lipgloss.NewStyle().Width(inner).MaxWidth(inner).Render(line)
	}
	border := borderColumn(rows, focused)
	for i := range rows {
		rows[i] = border[i] + rows[i]
	}
	return strings.Join(rows, "\n")
}

// bandView is the top row: the product name, the screens with the current one
// inverted, and the sessions that need you. See docs/tui.md.
func (m Model) bandView() string {
	screens := []struct {
		name string
		on   bool
	}{
		{"workspace", !m.reviewMode && m.help == nil},
		{"review", m.reviewMode && m.help == nil},
		{"keys", m.help != nil},
	}
	left := headingLabelStyle.Render(" multiplexer ") + "  "
	for _, s := range screens {
		style := labelStyle
		if s.on {
			style = focusLabelStyle
		}
		left += " " + style.Render(" "+s.name+" ")
	}
	right := m.bandAlerts()
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		right, gap = "", maxInt(0, m.width-lipgloss.Width(left))
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

// bandAlerts counts the live sessions that wait for an answer or failed, each
// with its state word, so they show from any screen.
func (m Model) bandAlerts() string {
	var waiting, failed int
	for _, item := range m.rows {
		if !item.live {
			continue
		}
		switch item.state {
		case session.StateWaiting:
			waiting++
		case session.StateFailed:
			failed++
		}
	}
	var parts []string
	if waiting > 0 {
		parts = append(parts, stateStyle(session.StateWaiting).Render(fmt.Sprintf("■ %d waiting", waiting)))
	}
	if failed > 0 {
		parts = append(parts, stateStyle(session.StateFailed).Render(fmt.Sprintf("■ %d failed", failed)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, fgStyle(colDimmed).Render(" · ")) + " "
}

// promptRule is the rule above the prompt. It carries a junction under each
// vertical rule of the body, so the rules meet.
func (m Model) promptRule() string {
	joins := map[int]bool{}
	if _, dialog := m.bodyDialogView(); !dialog && !m.reviewMode {
		if !m.sidebarHidden {
			joins[m.sidebarCols()-1] = true
		}
		if _, paneDialog := m.sessionDialogView(); !paneDialog && m.showSidePanel() && !m.sidePanelHorizontal() {
			x := m.leftWidth() + gutterWidth
			if !(m.diffPanel && m.layout.DiffPosition == config.DiffLeft) {
				x += m.outputWidth()
			}
			joins[x] = true
		}
	}
	var b strings.Builder
	for x := 0; x < m.width; x++ {
		if joins[x] {
			b.WriteString("┴")
		} else {
			b.WriteString("─")
		}
	}
	return ruleStyle.Render(b.String())
}
