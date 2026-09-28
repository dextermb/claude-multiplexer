package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

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

// barViewWidth is the head of the output pane: the session name set in the
// rule, and a row of details under it. See docs/tui/sessions/bars.md.
func (m Model) barViewWidth(width int) string {
	focused := m.focus == focusOutput
	item, ok := m.selectedRow()
	if !ok {
		return ruleLabel("no session", width, focused) + "\n" + barLine(width, "", "", width)
	}
	return ruleLabel(item.displayName(), width, focused) + "\n" + m.barDetails(item, width)
}

func (m Model) barDetails(item row, width int) string {
	lefts, rights := m.barLefts(item), m.barRights(item)
	for _, pair := range barLadder(len(lefts), len(rights)) {
		left := lefts[pair[0]]
		right := rights[pair[1]]
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right); gap >= 0 {
			return barLine(width, left, right, gap)
		}
	}
	return barLine(width, "", "", width)
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
	return barStyle.MaxHeight(1).Width(width).Render(left + strings.Repeat(" ", gap) + right)
}

type barSeg struct {
	text  string
	style lipgloss.Style
}

// leftSegs builds the ordered left segments of the session bar from the resolved
// left elements.
func (m Model) leftSegs(item row) []barSeg {
	var segs []barSeg
	for _, e := range m.barSpec(config.BarSession).Left {
		if !e.Custom() && e.ID == "name" {
			continue
		}
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

// barLefts renders the left side, from the full set of segments down to none,
// so the ladder sheds the trailing segments to make the line fit.
func (m Model) barLefts(item row) []string {
	segs := m.leftSegs(item)
	out := make([]string, 0, len(segs)+1)
	for n := len(segs); n >= 0; n-- {
		out = append(out, strings.TrimPrefix(renderSegs(segs[:n]), " "))
	}
	return out
}

func (m Model) barRights(item row) []string {
	segs := m.rightSegs(item)
	out := make([]string, 0, len(segs)+1)
	for n := len(segs); n >= 0; n-- {
		out = append(out, renderSegs(segs[:n]))
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
		return []barSeg{{rowGlyph(item) + " " + m.stateWord(item), item.style()}}
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

// renderSegs sets segments apart with dimmed dots, with one space at each end.
func renderSegs(segs []barSeg) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(" ")
	for i, seg := range segs {
		if i > 0 {
			b.WriteString(barSepStyle.Render(" · "))
		}
		b.WriteString(seg.style.Render(seg.text))
	}
	b.WriteString(" ")
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
	if view, ok := m.findStatus(); ok {
		return statusStyle.Width(m.width).Render(view)
	}
	if m.seq != nil {
		label := invertStyle.Render(" " + targetLabel(m.keys, m.seq.target) + " → ")
		hints := fitHints(m.sequenceHints(m.seq.target), m.width-lipgloss.Width(label)-4)
		return statusStyle.Width(m.width).Render(label + statusMutedStyle.Render("  ") + styleHints(hints))
	}
	spec := m.barSpec(config.BarStatus)
	left, right := m.statusParts(spec.Left), m.statusParts(spec.Right)
	return statusStyle.Width(m.width).Render(statusLine(m.width-2, left, right))
}

// statusPart is one element of the status bar. A hint list is kept plain, so it
// can shed hints to fit the width it gets.
type statusPart struct {
	text  string
	hints string
}

func (p statusPart) render(width int) string {
	if p.hints != "" {
		return styleHints(fitHints(p.hints, width))
	}
	return p.text
}

func (m Model) statusParts(elements []config.BarElement) []statusPart {
	var parts []statusPart
	for _, e := range elements {
		if !e.Custom() && e.ID == "hints" {
			parts = append(parts, statusPart{hints: m.statusHints()})
			continue
		}
		for _, seg := range m.statusSeg(e) {
			parts = append(parts, statusPart{text: seg.style.Render(seg.text)})
		}
	}
	return parts
}

func (m Model) statusSeg(e config.BarElement) []barSeg {
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

// statusLine fits the two sides in the width. A hint list is the one part that
// shrinks, so the other parts keep their text: it sheds hints first, and only
// when it is down to its ellipsis do the other parts drop, from the right side
// first. See docs/config/bars.md.
func statusLine(width int, left, right []statusPart) string {
	for {
		need := fixedWidth(left) + fixedWidth(right) + 1
		if need <= width {
			extra := width - need
			l, r := joinParts(left, fixedWidth(left)+extra), joinParts(right, fixedWidth(right))
			if !hasHints(left) {
				r = joinParts(right, fixedWidth(right)+extra)
			}
			return statusFill(l, r, maxInt(0, width-lipgloss.Width(l)-lipgloss.Width(r)))
		}
		var ok bool
		if right, ok = dropFixed(right); !ok {
			if left, ok = dropFixed(left); !ok {
				return ""
			}
		}
	}
}

// fixedWidth is the width a side takes with each hint list cut to its ellipsis.
func fixedWidth(parts []statusPart) int {
	w := 0
	for i, p := range parts {
		if i > 0 {
			w += lipgloss.Width(statusSepStyle.Render(" · "))
		}
		if p.hints != "" {
			w++
			continue
		}
		w += lipgloss.Width(p.text)
	}
	return w
}

func hasHints(parts []statusPart) bool {
	for _, p := range parts {
		if p.hints != "" {
			return true
		}
	}
	return false
}

func dropFixed(parts []statusPart) ([]statusPart, bool) {
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i].hints == "" {
			return append(parts[:i:i], parts[i+1:]...), true
		}
	}
	return parts, false
}

// joinParts sets the parts apart with dimmed dots, and gives a hint list the
// width that the other parts leave.
func joinParts(parts []statusPart, width int) string {
	sep := statusSepStyle.Render(" · ")
	fixed := 0
	for i, p := range parts {
		if i > 0 {
			fixed += lipgloss.Width(sep)
		}
		if p.hints == "" {
			fixed += lipgloss.Width(p.text)
		}
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = p.render(width - fixed)
	}
	return strings.Join(out, sep)
}

func statusFill(left, right string, gap int) string {
	return left + statusMutedStyle.Render(strings.Repeat(" ", gap)) + right
}
