package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// barSpec resolves the composition of one bar: the configured sides when the
// settings name them, and the embedded defaults when they do not. See
// docs/config/bars.md.
func (m Model) barSpec(bar string) config.BarSpec {
	if spec, ok := m.barSpecs[bar]; ok {
		return spec
	}
	spec, _ := config.ResolveBarSpec(nil, bar)
	return spec
}

// barOutput reads the cached output of a custom element. A miss draws nothing.
func (m Model) barOutput(bar, session, label string) (string, bool) {
	if m.barOutputs == nil {
		return "", false
	}
	text, ok := m.barOutputs[barOutputKey(bar, session, label)]
	return text, ok
}

// barOutputKey keys a custom element output by the bar, the session it draws
// for (empty for the status bar), and the element label.
func barOutputKey(bar, session, label string) string {
	return bar + "\x00" + session + "\x00" + label
}

// barStyleFor maps a custom element's style name to a style that fits the bar.
// See docs/config/bars.md.
func (m Model) barStyleFor(bar, style string) lipgloss.Style {
	if bar == config.BarStatus {
		switch style {
		case "strong":
			return statusKeyStyle
		case "cost":
			return statusCostStyle
		default:
			return statusMutedStyle
		}
	}
	switch style {
	case "strong":
		return barNameStyle
	case "cost":
		return barCostStyle
	case "warn":
		return barPRStyle
	default:
		return barMutedStyle
	}
}

// customSegs draws a custom element from its cached script output.
func (m Model) customSegs(bar, session string, e config.BarElement) []barSeg {
	text, ok := m.barOutput(bar, session, e.Label)
	if !ok || text == "" {
		return nil
	}
	return []barSeg{{text, m.barStyleFor(bar, e.Style)}}
}

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
		return ruleLabel("no session", width, m.focus == focusOutput)
	}

	lefts, rights := m.barLefts(item), m.barRights(item)
	for _, pair := range barLadder(len(lefts), len(rights)) {
		left := lefts[pair[0]]
		right := rights[pair[1]]
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right); gap >= 0 {
			return barLine(width, left, right, gap)
		}
	}

	return ruleLabel(item.displayName(), width, m.focus == focusOutput)
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

// barLine is the head of the output pane: the left segments and the right
// segments set in one rule. See docs/tui/sessions/bars.md.
func barLine(width int, left, right string, gap int) string {
	return barStyle.MaxHeight(1).Width(width).Render(left + rule(gap) + right)
}

type barSeg struct {
	text  string
	style lipgloss.Style
}

func (m Model) barLabelStyle() lipgloss.Style {
	if m.focus == focusOutput {
		return focusLabelStyle
	}
	return headingLabelStyle
}

// leftSegs builds the ordered left segments of the session bar from the resolved
// left elements.
func (m Model) leftSegs(item row) []barSeg {
	var segs []barSeg
	for _, e := range m.barSpec(config.BarSession).Left {
		segs = append(segs, m.sessionLeftSeg(item, e)...)
	}
	return segs
}

func (m Model) sessionLeftSeg(item row, e config.BarElement) []barSeg {
	if e.Custom() {
		return m.customSegs(config.BarSession, item.name, e)
	}
	switch e.ID {
	case "name":
		return []barSeg{{item.displayName(), barNameStyle}}
	case "control":
		if item.control {
			return []barSeg{{"control", barMutedStyle}}
		}
	case "model":
		if item.model != "" {
			return []barSeg{{item.model, barMutedStyle}}
		}
	case "mode":
		return []barSeg{{item.mode, barMutedStyle}}
	case "effort":
		if item.effort != "" {
			return []barSeg{{item.effort + " effort", barMutedStyle}}
		}
	case "workItem":
		if wi := barWorkItem(item); wi != "" {
			return []barSeg{{wi, barMutedStyle}}
		}
	}
	return nil
}

// barLefts renders the left side, from the full set of segments down to the
// first, so the ladder sheds the trailing segments to make the line fit.
func (m Model) barLefts(item row) []string {
	segs := m.leftSegs(item)
	spec := m.barSpec(config.BarSession).Left
	labelled := len(spec) > 0 && spec[0].ID == "name"
	if len(segs) == 0 {
		return []string{renderLeft(nil, false, m.barLabelStyle())}
	}
	out := make([]string, 0, len(segs))
	for n := len(segs); n >= 1; n-- {
		out = append(out, renderLeft(segs[:n], labelled, m.barLabelStyle()))
	}
	return out
}

// renderLeft sets the name segment, when it comes first, as the pane label in
// its own gap in the rule, and the other segments after it, separated by dots.
func renderLeft(segs []barSeg, labelled bool, label lipgloss.Style) string {
	var b strings.Builder
	b.WriteString(rule(1))
	rest := segs
	if len(segs) > 0 && labelled {
		b.WriteString(label.Render(" " + segs[0].text + " "))
		rest = segs[1:]
		if len(rest) > 0 {
			b.WriteString(rule(1))
		}
	}
	if len(rest) == 0 {
		return b.String()
	}
	b.WriteString(" ")
	for i, seg := range rest {
		if i > 0 {
			b.WriteString(barMutedStyle.Render(" · "))
		}
		b.WriteString(seg.style.Render(seg.text))
	}
	b.WriteString(" ")
	return b.String()
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
	var segs []barSeg
	for _, e := range m.barSpec(config.BarSession).Right {
		segs = append(segs, m.sessionRightSeg(item, e)...)
	}
	return segs
}

func (m Model) sessionRightSeg(item row, e config.BarElement) []barSeg {
	if e.Custom() {
		return m.customSegs(config.BarSession, item.name, e)
	}
	switch e.ID {
	case "state":
		return []barSeg{{rowGlyph(item) + " " + m.stateWord(item), item.style().Background(barBackground)}}
	case "diff":
		if d, ok := m.diffs[item.name]; ok && d.anyRepo() && !d.stat().Empty() {
			return []barSeg{{barDiffCount(d.stat()), barStyle}}
		}
	case "pr":
		var segs []barSeg
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
		return segs
	case "context":
		if item.live && item.context > 0 {
			return []barSeg{{contextLabel(item), barMutedStyle}}
		}
	case "raw":
		if m.showRaw {
			return []barSeg{{"raw", barMutedStyle}}
		}
	case "scroll":
		if scroll := m.scrollIndicator(); scroll != "" {
			return []barSeg{{scroll, barMutedStyle}}
		}
	case "jobs":
		if item.jobs > 0 {
			return []barSeg{{fmt.Sprintf("jobs (%d)", item.jobs), barMutedStyle}}
		}
	case "queued":
		if item.queued > 0 {
			return []barSeg{{fmt.Sprintf("queued (%d)", item.queued), barMutedStyle}}
		}
	case "tokens":
		if item.input+item.output > 0 {
			return []barSeg{{
				fmt.Sprintf("%s in %s out", formatCount(item.input), formatCount(item.output)),
				barMutedStyle,
			}}
		}
	case "cache":
		if rate, ok := session.CacheHitRate(item.input, item.cacheRead); ok {
			return []barSeg{{fmt.Sprintf("cache %d%%", rate), barMutedStyle}}
		}
	case "cost":
		return []barSeg{{fmt.Sprintf("$%.4f", item.cost), barCostStyle}}
	}
	return nil
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
	b.WriteString(" ")
	for i, seg := range segs {
		if i > 0 {
			b.WriteString(barMutedStyle.Render(" · "))
		}
		b.WriteString(seg.style.Render(seg.text))
	}
	b.WriteString(" " + rule(1))
	return b.String()
}

func (m Model) scrollIndicator() string {
	if m.output.AtBottom() {
		return ""
	}
	return fmt.Sprintf("↑ %d%%", int(m.output.ScrollPercent()*100))
}

func formatCount(n int) string {
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

// liveBusy counts the live sessions and the busy ones among them.
func (m Model) liveBusy() (live, busy int) {
	for _, item := range m.rows {
		if !item.live {
			continue
		}
		live++
		if item.state == session.StateBusy {
			busy++
		}
	}
	return live, busy
}

func (m Model) statusView() string {
	if m.errText != "" {
		return statusStyle.Width(m.width).Render(errorStyle.Render(truncate(m.errText, m.width-2)))
	}
	if m.seq != nil {
		label := invertStyle.Render(" " + targetLabel(m.keys, m.seq.target) + " → ")
		hints := truncate(m.sequenceHints(m.seq.target), m.width-lipgloss.Width(label)-4)
		return statusStyle.Width(m.width).Render(label + statusMutedStyle.Render("  ") + styleHints(hints))
	}
	left := m.statusLeftSegs()
	right := m.statusRight()
	return statusStyle.Width(m.width).Render(statusLine(m.width-2, left, right))
}

func (m Model) statusLeftSegs() []barSeg {
	var segs []barSeg
	for _, e := range m.barSpec(config.BarStatus).Left {
		segs = append(segs, m.statusLeftSeg(e)...)
	}
	return segs
}

func (m Model) statusLeftSeg(e config.BarElement) []barSeg {
	if e.Custom() {
		return m.customSegs(config.BarStatus, "", e)
	}
	switch e.ID {
	case "sessions":
		live, _ := m.liveBusy()
		return []barSeg{{plural(live, "session"), statusMutedStyle}}
	case "busy":
		if _, busy := m.liveBusy(); busy > 0 {
			return []barSeg{{fmt.Sprintf("%d busy", busy), statusMutedStyle}}
		}
	case "cost":
		return []barSeg{{m.costSeg(), statusCostStyle}}
	case "status":
		if m.status != "" {
			return []barSeg{{m.status, statusMutedStyle}}
		}
	default:
		return m.statusSessionSeg(e)
	}
	return nil
}

// statusSessionSeg draws a session element on the status bar, for the selected
// session, restyled to the status bar. See docs/config/bars.md.
func (m Model) statusSessionSeg(e config.BarElement) []barSeg {
	item, ok := m.selectedRow()
	if !ok {
		return nil
	}
	return statusRestyle(m.sessionSeg(item, e))
}

// sessionSeg draws one built-in session element, from the side its id belongs to.
func (m Model) sessionSeg(item row, e config.BarElement) []barSeg {
	if config.SessionElementSide(e.ID) == "left" {
		return m.sessionLeftSeg(item, e)
	}
	return m.sessionRightSeg(item, e)
}

// statusRestyle rebackgrounds session segments to the status bar, which shares
// the foreground colours but sits on a different background.
func statusRestyle(segs []barSeg) []barSeg {
	for i := range segs {
		segs[i].style = segs[i].style.Background(statusBackground)
	}
	return segs
}

func (m Model) statusRight() string {
	var parts []string
	for _, e := range m.barSpec(config.BarStatus).Right {
		if e.Custom() {
			if text, ok := m.barOutput(config.BarStatus, "", e.Label); ok && text != "" {
				parts = append(parts, m.barStyleFor(config.BarStatus, e.Style).Render(text))
			}
			continue
		}
		if e.ID == "hints" {
			parts = append(parts, styleHints(m.statusHints()))
			continue
		}
		for _, seg := range m.statusSessionSeg(e) {
			parts = append(parts, seg.style.Render(seg.text))
		}
	}
	return strings.Join(parts, statusMutedStyle.Render("  "))
}

func statusSegs(segs []barSeg) string {
	parts := make([]string, len(segs))
	for i, seg := range segs {
		parts[i] = seg.style.Render(seg.text)
	}
	return strings.Join(parts, statusMutedStyle.Render(" · "))
}

func statusLine(width int, left []barSeg, right string) string {
	if len(left) == 0 {
		if gap := width - lipgloss.Width(right); gap >= 0 {
			return statusFill("", right, gap)
		}
		return statusMutedStyle.Render(truncate(right, width))
	}
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
