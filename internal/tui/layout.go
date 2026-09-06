package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func (m Model) bodyHeight() int {
	height := m.height - m.promptHeight() - statusHeight
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) outputHeight() int {
	height := m.bodyHeight() - barHeight
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) visibleLines() int {
	lines := m.bodyHeight() - titleHeight
	if lines < 1 {
		return 1
	}
	return lines
}

// visibleRowIndexes lists the rows a fold does not hide, in the order they are
// drawn, as indexes into m.rows.
func (m Model) visibleRowIndexes() []int {
	out := make([]int, 0, len(m.rows))
	for _, line := range m.lines {
		if !line.header() {
			out = append(out, line.row)
		}
	}
	return out
}

// leftWidth is the width of the left sidebar, or zero when it is collapsed.
func (m Model) leftWidth() int {
	if m.sidebarHidden {
		return 0
	}
	return m.sidebarCols()
}

// applyLayout resolves the dimensions of the selected session, so the sidebar,
// the panels, and the prompt draw at the layout of that session. See
// docs/tui.md.
func (m *Model) applyLayout() {
	name := ""
	if item, ok := m.selectedRow(); ok {
		name = item.layout
	}
	m.layout = config.ResolveLayout(m.layouts, m.activeLayout, name)
	m.syncPromptHeight()
	m.output.Width = m.outputWidth()
	m.output.Height = m.outputHeight()
}

// sidebarCols is the width of the session list sidebar, from the layout, kept
// narrow enough that the output pane keeps its minimum. A Model with no layout
// takes the built-in default.
func (m Model) sidebarCols() int {
	w := m.layout.SidebarWidth
	if w < 1 {
		w = config.DefaultSidebarWidth
	}
	if lim := m.width - 10; lim > 0 && w > lim {
		w = lim
	}
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) sidebarInnerCols() int {
	inner := m.sidebarCols() - 1 - gutterWidth
	if inner < 1 {
		inner = 1
	}
	return inner
}

// taskCols is the width of the task and background job panel, from the layout. A
// Model with no layout takes the built-in default.
func (m Model) taskCols() int {
	if m.layout.TaskWidth < 1 {
		return config.DefaultTaskWidth
	}
	return m.layout.TaskWidth
}

func (m Model) taskInnerCols() int {
	return m.taskCols() - 2
}

func (m Model) baseOutputWidth() int {
	width := m.width - m.leftWidth() - gutterWidth
	if width < 10 {
		return 10
	}
	return width
}

func (m Model) outputWidth() int {
	if m.showSidePanel() {
		return m.baseOutputWidth() - m.sidePanelWidth()
	}
	return m.baseOutputWidth()
}

// sidePanelWidth is the width of the panel beside the output: the resizable diff
// panel when it is open, else the fixed jobs and tasks panel.
func (m Model) sidePanelWidth() int {
	if m.diffPanel {
		return m.diffPanelWidth()
	}
	return m.taskCols()
}

// showSidePanel says whether the side panel has room beside the output. It reads
// baseOutputWidth, not outputWidth, because outputWidth depends on it. See
// docs/tui/tasks.md.
func (m Model) showSidePanel() bool {
	if m.diffPanel {
		return m.baseOutputWidth()-m.sidePanelWidth() >= minOutputWithPanel
	}
	if len(m.todos[m.sel]) == 0 && len(m.selectedJobs()) == 0 {
		return false
	}
	return m.baseOutputWidth()-m.taskCols() >= minOutputWithPanel
}

func (m Model) View() string {
	if !m.ready {
		return "starting…"
	}
	body := withEdge(m.paneView(), m.focus == focusOutput)
	if !m.sidebarHidden {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.sidebarView(), body)
	}
	if dialog, ok := m.bodyDialogView(); ok {
		body = dialog
	}
	prompt := withEdge(m.promptView(), m.focus == focusPrompt)
	return lipgloss.JoinVertical(lipgloss.Left, body, prompt, m.statusView())
}

// A session dialog draws in the pane, not over the whole body; see docs/tui.md.
func (m Model) paneView() string {
	if dialog, ok := m.sessionDialogView(); ok {
		return lipgloss.JoinVertical(lipgloss.Left, m.barView(), dialog)
	}
	pane := lipgloss.JoinVertical(lipgloss.Left, m.barView(), m.outputView())
	if m.showSidePanel() {
		pane = lipgloss.JoinHorizontal(lipgloss.Top, pane, m.sidePanelView())
	}
	return pane
}

func (m Model) sessionDialogView() (string, bool) {
	width, height := m.baseOutputWidth(), m.outputHeight()
	switch {
	case m.confirm != "":
		return centre(width, height, m.confirmView(width)), true
	case m.jobsModal != nil:
		return centre(width, height, m.jobsModal.View(width, height)), true
	case m.rename != nil:
		return centre(width, height, m.rename.View(width)), true
	case m.choice != nil:
		return centre(width, height, m.choice.View(width)), true
	}
	return "", false
}

func (m Model) bodyDialogView() (string, bool) {
	width, height := m.width, m.bodyHeight()
	switch {
	case m.help != nil:
		return centre(width, height, m.help.View(width, height)), true
	case m.fields != nil:
		return centre(width, height, m.fields.View(width)), true
	case m.picker != nil:
		return centre(width, height, m.picker.View(width)), true
	case m.form != nil:
		return centre(width, height, m.form.View(width)), true
	case m.layoutSwitch != nil:
		return centre(width, height, m.layoutSwitch.View(width)), true
	}
	return "", false
}

func withEdge(block string, on bool) string {
	ch := " "
	if on {
		ch = focusEdgeStyle.Render(edgeMark)
	}
	height := lipgloss.Height(block)
	edge := make([]string, height)
	for i := range edge {
		edge[i] = ch
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(edge, "\n"), block)
}

func (m Model) sidebarView() string {
	rows := make([]string, 0, m.bodyHeight())

	visible := m.visibleLines()
	for i := m.listOffset; i < len(m.lines) && len(rows) < visible; i++ {
		line := m.lines[i]
		if line.header() {
			rows = append(rows, m.groupHeader(m.groups[line.group]))
			continue
		}
		rows = append(rows, m.sessionRow(m.rows[line.row]))
	}
	for len(rows) < m.bodyHeight() {
		rows = append(rows, strings.Repeat(" ", m.sidebarInnerCols()))
	}
	block := sidebarStyle.Width(m.sidebarInnerCols()).Height(m.bodyHeight()).Render(strings.Join(rows, "\n"))
	return withEdge(block, m.focus == focusSidebar)
}

func (m Model) sessionRow(item row) string {
	width := m.sidebarInnerCols()
	badge := ""
	if item.control && !headsGroup(item) {
		badge = " " + controlMark
	}
	if item.jobs > 0 {
		badge += fmt.Sprintf(" ⚙%d", item.jobs)
	}
	if item.queued > 0 {
		badge += fmt.Sprintf(" ⇢%d", item.queued)
	}
	nameWidth := width - 3 - lipgloss.Width(badge)
	if nameWidth < 1 {
		nameWidth = 1
	}
	rest := " " + pad(item.displayName(), nameWidth) + badge
	glyph := rowGlyph(item, m.spinFrame)
	if item.name == m.sel {
		return selectedRowStyle.Render(" ") +
			item.style().Background(lipgloss.Color("62")).Render(glyph) +
			selectedRowStyle.Width(width-2).Render(rest)
	}
	nameStyle := rowStyle
	if item.archived {
		nameStyle = rowMutedStyle
	}
	return " " + item.style().Render(glyph) + nameStyle.Width(width-2).Render(rest)
}

// groupHeader names one directory. A folded header also carries the glyph of the
// most urgent row it hides.
func (m Model) groupHeader(item group) string {
	mark, glyph := foldOpenMark, ""
	if item.folded {
		lead := row{live: item.live, archived: item.archived, state: item.state}
		mark = foldShutMark
		glyph = lead.style().Render(rowGlyph(lead, m.spinFrame)) + " "
	}
	label := item.label
	if item.creator {
		label = controlMark + " " + label
	}
	count := strconv.Itoa(item.count)
	width := m.sidebarInnerCols() - 3 - lipgloss.Width(glyph) - len(count)
	if width < 1 {
		width = 1
	}
	return groupMarkStyle.Render(mark) + " " +
		groupLabelStyle.Render(pad(label, width)) + " " +
		glyph + groupCountStyle.Render(count)
}

func (m Model) outputView() string {
	if q := m.questions[m.sel]; q != nil {
		return centre(m.outputWidth(), m.outputHeight(), q.View(m.outputWidth(), m.caps))
	}
	if len(m.rows) == 0 {
		text := "No sessions yet.\n\nPress n to start one.\nPress ctrl+c to quit."
		if len(m.stored) > 0 {
			text = "Every stored session is archived.\n\nPress l a to show them.\nPress n to start a new one."
		}
		return emptyStyle.Width(m.outputWidth()).Height(m.outputHeight()).Render(text)
	}
	return m.output.View()
}

func (m Model) sidePanelView() string {
	if m.diffPanel {
		return m.diffPanelView()
	}
	var rows []string
	if jobs := orderJobs(m.selectedJobs()); len(jobs) > 0 {
		running := 0
		for _, job := range jobs {
			if job.Status.Running() {
				running++
			}
		}
		rows = append(rows, taskHeaderStyle.Render(fmt.Sprintf("Jobs · %d/%d", running, len(jobs))), "")
		for _, job := range jobs {
			rows = append(rows, m.panelJobRow(job))
		}
	}
	if todos := m.todos[m.sel]; len(todos) > 0 {
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		done := 0
		for _, todo := range todos {
			if todo.Status == protocol.TodoCompleted {
				done++
			}
		}
		rows = append(rows, taskHeaderStyle.Render(fmt.Sprintf("Tasks · %d/%d", done, len(todos))), "")
		item, ok := m.selectedRow()
		busy := ok && item.live && item.state == session.StateBusy
		for _, todo := range todos {
			rows = append(rows, m.taskRow(todo, busy))
		}
	}
	block := strings.Join(rows, "\n")
	return taskPanelStyle.Width(m.taskCols() - 1).Height(m.bodyHeight()).Render(block)
}

func (m Model) panelJobRow(job session.Job) string {
	desc := job.Description
	if desc == "" {
		desc = job.ID
	}
	textStyle := rowStyle
	if !job.Status.Running() {
		textStyle = rowMutedStyle
	}
	return jobStyle(job.Status).Render(jobGlyph(job.Status)) + " " + textStyle.Render(truncate(desc, m.taskInnerCols()-2))
}

func (m Model) taskRow(todo protocol.Todo, busy bool) string {
	glyph, glyphStyle, textStyle := "○", taskPendingStyle, rowStyle
	text := todo.Content
	switch todo.Status {
	case protocol.TodoCompleted:
		glyph, glyphStyle, textStyle = "✔", taskDoneStyle, rowMutedStyle
	case protocol.TodoInProgress:
		glyph, glyphStyle = "◐", taskActiveStyle
		if todo.ActiveForm != "" {
			text = todo.ActiveForm
		}
		if busy {
			glyph = spinnerFrame(m.spinFrame)
		}
	}
	return glyphStyle.Render(glyph) + " " + textStyle.Render(truncate(text, m.taskInnerCols()-2))
}

func (m Model) barView() string {
	width := m.outputWidth()
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
	if d, ok := m.diffs[item.name]; ok && d.repo && !d.stat.Empty() {
		segs = append(segs, barSeg{barDiffCount(d.stat), barStyle})
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
	return append(segs, barSeg{fmt.Sprintf("$%.4f", item.cost), barCostStyle})
}

func contextLabel(item row) string {
	if limit := contextWindow(item.model); limit > 0 {
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
	out := make([][]string, 0, len(full)+1)
	for n := len(full); n >= 1; n-- {
		out = append(out, full[:n])
	}
	return append(out, nil)
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func formatCount(n int) string {
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

func (m Model) promptView() string {
	label := "prompt"
	if m.sel != "" {
		label = m.sel
	}
	if hint, ok := m.mentionHint(); ok {
		return hint + "\n" + m.prompt.View()
	}
	if names := completionNames(m.templates, m.prompt.Value()); len(names) > 0 && m.focus == focusPrompt {
		return hintStyle.Render(truncate(strings.Join(names, "  ")+"   tab completes", m.width-2)) +
			"\n" + m.prompt.View()
	}
	if item, ok := m.selectedRow(); ok && !item.running() {
		return hintStyle.Render(label+" — not running, press Enter to resume") + "\n" + m.prompt.View()
	}
	if item, ok := m.selectedRow(); ok && item.state == session.StateBusy && m.focus == focusPrompt {
		hint := label + " — esc stops"
		if len(m.queued[m.sel]) > 0 {
			hint += " · enter sends queued"
		}
		return hintStyle.Render(hint) + "\n" + m.prompt.View()
	}
	if m.focus == focusPrompt {
		return promptLabelStyle.Render(label+" ⌁ ") + "\n" + m.prompt.View()
	}
	return hintStyle.Render(label+" — press Enter or Tab to type") + "\n" + m.prompt.View()
}

func (m Model) confirmView(width int) string {
	return modalStyle.Width(modalInner(width)).Render(fmt.Sprintf("Stop session %q?\n\n%s",
		m.confirm, hintStyle.Render("y stop · any other key cancel")))
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
	left = append(left, barSeg{fmt.Sprintf("$%.4f", m.mgr.TotalCost()), statusCostStyle})
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
