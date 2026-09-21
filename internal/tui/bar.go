package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// barWorkItem is the work-item segment of the output pane status bar: the
// mirrored status, next to the effort, and the key when no status is known. See
// docs/work-items.md.
func barWorkItem(item row) string {
	switch {
	case item.workItem.Status != "":
		return item.workItem.Status
	case item.workItem.Key != "":
		return item.workItem.Key
	default:
		return ""
	}
}

// barPR is one pull-request segment of the output pane status bar: the number
// with a provider prefix, the state when it is not open, and the count of
// unresolved review threads. A project draws one segment per code base. See
// docs/pull-requests.md.
func barPR(pr manager.PRBadge) string {
	if pr.Number == 0 {
		return ""
	}
	prefix := "#"
	if pr.Provider == "gitlab" {
		prefix = "!"
	}
	label := prefix + strconv.Itoa(pr.Number)
	switch pr.State {
	case "draft":
		label += " draft"
	case "merged":
		label += " merged"
	case "closed":
		label += " closed"
	}
	if pr.Unresolved > 0 {
		label += fmt.Sprintf(" (%d)", pr.Unresolved)
	}
	return label
}

func (m Model) barView() string {
	return m.barViewWidth(m.outputWidth())
}

func (m Model) barViewWidth(width int) string {
	item, ok := m.selectedRow()
	if !ok {
		return barStyle.Width(width).Render(barMutedStyle.Render(" no session"))
	}

	lefts, rights := barDetails(item), m.barRights(item)
	for _, pair := range barLadder(len(lefts), len(rights)) {
		left := barLeft(item.displayName(), lefts[pair[0]])
		right := rights[pair[1]]
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right); gap >= 0 {
			return barLine(width, left, right, gap)
		}
	}

	room := maxInt(3, width-1)
	left := barLeft(truncate(item.displayName(), room), nil)
	return barLine(width, left, "", maxInt(0, width-lipgloss.Width(left)))
}

func barLadder(lefts, rights int) [][2]int {
	pairs := [][2]int{{0, 0}}
	left, right, takeLeft := 0, 0, true
	for left < lefts-1 || right < rights-1 {
		switch {
		case takeLeft && left < lefts-1:
			left++
		case right < rights-1:
			right++
		default:
			left++
		}
		takeLeft = !takeLeft
		pairs = append(pairs, [2]int{left, right})
	}
	return pairs
}

func barLine(width int, left, right string, gap int) string {
	return barStyle.MaxHeight(1).Width(width).
		Render(left + barStyle.Render(strings.Repeat(" ", gap)) + right)
}

func barLeft(name string, details []string) string {
	left := barNameStyle.Render(" " + name)
	if len(details) > 0 {
		left += barMutedStyle.Render(" · " + strings.Join(details, " · "))
	}
	return left
}

type barSeg struct {
	text  string
	style lipgloss.Style
}

func (m Model) barRights(item row) []string {
	segs := m.rightSegs(item)
	out := make([]string, 0, len(segs)+1)
	for n := len(segs); n >= 0; n-- {
		out = append(out, renderRight(segs[:n]))
	}
	return out
}

func (m Model) rightSegs(item row) []barSeg {
	segs := []barSeg{{item.label, item.style().Background(barBackground)}}
	if d, ok := m.diffs[item.name]; ok && d.anyRepo() && !d.stat().Empty() {
		segs = append(segs, barSeg{barDiffCount(d.stat()), barStyle})
	}
	for _, pr := range item.prs {
		label := barPR(pr)
		if label == "" {
			continue
		}
		style := barMutedStyle
		if pr.Unresolved > 0 {
			style = barPRStyle
		}
		segs = append(segs, barSeg{label, style})
	}
	if item.live && item.context > 0 {
		segs = append(segs, barSeg{contextLabel(item), barMutedStyle})
	}
	if m.showRaw {
		segs = append(segs, barSeg{"raw", barMutedStyle})
	}
	if scroll := m.scrollIndicator(); scroll != "" {
		segs = append(segs, barSeg{scroll, barMutedStyle})
	}
	if item.jobs > 0 {
		segs = append(segs, barSeg{fmt.Sprintf("⚙%d", item.jobs), barMutedStyle})
	}
	if item.queued > 0 {
		segs = append(segs, barSeg{fmt.Sprintf("⇢%d", item.queued), barMutedStyle})
	}
	if item.input+item.output > 0 {
		segs = append(segs, barSeg{
			fmt.Sprintf("%s in %s out", formatCount(item.input), formatCount(item.output)),
			barMutedStyle,
		})
	}
	if rate, ok := session.CacheHitRate(item.input, item.cacheRead); ok {
		segs = append(segs, barSeg{fmt.Sprintf("cache %d%%", rate), barMutedStyle})
	}
	return append(segs, barSeg{fmt.Sprintf("$%.4f", item.cost), barCostStyle})
}

func contextLabel(item row) string {
	if limit := session.ContextWindow(item.model); limit > 0 {
		pct := item.context * 100 / limit
		return fmt.Sprintf("ctx %s/%s (%d%%)", formatCount(item.context), formatCount(limit), pct)
	}
	return fmt.Sprintf("ctx %s", formatCount(item.context))
}

func renderRight(segs []barSeg) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range segs {
		if i > 0 {
			b.WriteString(barMutedStyle.Render(" · "))
		}
		b.WriteString(seg.style.Render(seg.text))
	}
	b.WriteString(barStyle.Render(" "))
	return b.String()
}

func (m Model) scrollIndicator() string {
	if m.output.AtBottom() {
		return ""
	}
	return fmt.Sprintf("↑ %d%%", int(m.output.ScrollPercent()*100))
}

func barDetails(item row) [][]string {
	var full []string
	if item.control {
		full = append(full, "control")
	}
	if item.model != "" {
		full = append(full, item.model)
	}
	full = append(full, item.mode)
	if item.effort != "" {
		full = append(full, item.effort+" effort")
	}
	if wi := barWorkItem(item); wi != "" {
		full = append(full, wi)
	}
	out := make([][]string, 0, len(full)+1)
	for n := len(full); n >= 1; n-- {
		out = append(out, full[:n])
	}
	return append(out, nil)
}

func formatCount(n int) string {
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

func (m Model) statusView() string {
	if m.errText != "" {
		return statusStyle.Width(m.width).Render(errorStyle.Render(truncate(m.errText, m.width-2)))
	}
	if m.seq != nil {
		hints := truncate(sequenceHints(m.seq.target), m.width-6)
		return statusStyle.Width(m.width).Render(
			statusKeyStyle.Render(m.seq.target) + statusMutedStyle.Render("  "+hints))
	}
	var live, busy int
	for _, item := range m.rows {
		if !item.live {
			continue
		}
		live++
		if item.state == session.StateBusy {
			busy++
		}
	}

	left := []barSeg{{plural(live, "session"), statusMutedStyle}}
	if busy > 0 {
		left = append(left, barSeg{fmt.Sprintf("%d busy", busy), statusMutedStyle})
	}
	left = append(left, barSeg{m.costSeg(), statusCostStyle})
	if m.status != "" {
		left = append(left, barSeg{m.status, statusMutedStyle})
	}
	right := statusMutedStyle.Render(statusHints())

	return statusStyle.Width(m.width).Render(statusLine(m.width-2, left, right))
}

func statusSegs(segs []barSeg) string {
	parts := make([]string, len(segs))
	for i, seg := range segs {
		parts[i] = seg.style.Render(seg.text)
	}
	return strings.Join(parts, statusMutedStyle.Render(" · "))
}

func statusLine(width int, left []barSeg, right string) string {
	full := statusSegs(left)
	if gap := width - lipgloss.Width(full) - lipgloss.Width(right); gap >= 0 {
		return statusFill(full, right, gap)
	}
	for n := len(left); n >= 1; n-- {
		side := statusSegs(left[:n])
		if gap := width - lipgloss.Width(side); gap >= 0 {
			return statusFill(side, "", gap)
		}
	}
	return statusMutedStyle.Render(truncate(left[0].text, width))
}

func statusFill(left, right string, gap int) string {
	return left + statusMutedStyle.Render(strings.Repeat(" ", gap)) + right
}
