package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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
// inverted, and the band bar on the right. See docs/tui.md.
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
	right := m.bandRight(m.width - lipgloss.Width(left) - 1)
	gap := maxInt(0, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return lipgloss.NewStyle().MaxWidth(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

type bandPart struct {
	text string
	keep int
}

// bandRight draws the band bar in width. When it does not fit, it drops the
// parts that call you least: the custom and session parts, then the cost, the
// session count, and the busy count. See docs/config/bars.md.
func (m Model) bandRight(width int) string {
	parts := m.bandParts()
	for {
		text := joinBand(parts)
		if lipgloss.Width(text) <= width || len(parts) == 0 {
			if len(parts) == 0 {
				return ""
			}
			return text
		}
		parts = dropLeast(parts)
	}
}

func joinBand(parts []bandPart) string {
	texts := make([]string, len(parts))
	for i, p := range parts {
		texts[i] = p.text
	}
	return strings.Join(texts, barSepStyle.Render(" · ")) + " "
}

func dropLeast(parts []bandPart) []bandPart {
	at := len(parts) - 1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i].keep < parts[at].keep {
			at = i
		}
	}
	return append(parts[:at:at], parts[at+1:]...)
}

func (m Model) bandParts() []bandPart {
	var parts []bandPart
	for _, e := range m.barSpec(config.BarBand).Right {
		parts = append(parts, m.bandPart(e)...)
	}
	return parts
}

func (m Model) bandPart(e config.BarElement) []bandPart {
	if e.Custom() {
		return renderBandSegs(m.customSegs(config.BarBand, "", e))
	}
	count := func(state session.State) int {
		n := 0
		for _, item := range m.rows {
			if item.live && item.state == state {
				n++
			}
		}
		return n
	}
	switch e.ID {
	case "sessions":
		live, _ := m.liveBusy()
		return []bandPart{{barMutedStyle.Render("sessions ") +
			fgStyle(colSecondary).Render(fmt.Sprintf("(%d · %d live)", len(m.rows), live)), 1}}
	case "busy":
		if n := count(session.StateBusy); n > 0 {
			return []bandPart{{stateStyle(session.StateBusy).Render(fmt.Sprintf("%d busy", n)), 2}}
		}
	case "waiting":
		if n := count(session.StateWaiting); n > 0 {
			return []bandPart{{stateStyle(session.StateWaiting).Render(fmt.Sprintf("%d waiting", n)), 3}}
		}
	case "failed":
		if n := count(session.StateFailed); n > 0 {
			return []bandPart{{stateStyle(session.StateFailed).Render(fmt.Sprintf("%d failed", n)), 3}}
		}
	case "cost":
		text := fgStyle(colSecondary).Render(fmt.Sprintf("$%.4f", m.cost))
		if m.costWindow != "" {
			text += barMutedStyle.Render(" / " + m.costWindow)
		}
		return []bandPart{{text, 0}}
	default:
		if item, ok := m.selectedRow(); ok {
			return renderBandSegs(m.sessionSeg(item, e))
		}
	}
	return nil
}

func renderBandSegs(segs []barSeg) []bandPart {
	parts := make([]bandPart, len(segs))
	for i, seg := range segs {
		parts[i] = bandPart{seg.style.Render(seg.text), -1}
	}
	return parts
}

// promptRule is the rule above the prompt. It carries a junction under each
// vertical rule of the body, so the rules meet.
func (m Model) promptRule() string {
	joins := map[int]bool{}
	if _, dialog := m.bodyDialogView(); !dialog && m.reviewMode {
		joins[gutterWidth+m.reviewDiffWidth()] = true
	}
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
	line := []rune(b.String())

	start := 1
	if !m.sidebarHidden && !m.reviewMode {
		start = m.sidebarCols() + 1
	}
	end := len(line)
	for x := start; x < len(line); x++ {
		if joins[x] {
			end = x - 1
			break
		}
	}
	style := labelStyle
	if m.promptFocused() {
		style = focusLabelStyle
	}
	room := end - start
	if start >= len(line) || room < 4 {
		return ruleStyle.Render(string(line))
	}
	label := style.Render(truncate(" "+m.promptLabel()+" ", room))
	after := start + lipgloss.Width(label)
	return ruleStyle.Render(string(line[:start])) + label + ruleStyle.Render(string(line[after:]))
}

// promptLabel names what the prompt sends to.
func (m Model) promptLabel() string {
	switch {
	case m.sel == "":
		return "prompt"
	case m.reviewMode:
		return "follow-up → " + m.sel
	}
	return "prompt → " + m.sel
}

func (m Model) promptFocused() bool {
	return m.focus == focusPrompt || (m.reviewMode && m.reviewFocus == reviewPrompt)
}
