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
	if m.reviewMode {
		if h := m.reviewHeight() - 1; h >= 1 {
			return h
		}
		return 1
	}
	height := m.bodyHeight() - barHeight
	if m.showSidePanel() && m.sidePanelHorizontal() {
		height -= m.sidePanelHeight()
	}
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) visibleLines() int {
	lines := m.bodyHeight() - titleHeight - m.searchRows()
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
	w := m.layout.SidebarSize
	if w < 1 {
		w = config.DefaultSidebarSize
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
	if m.layout.TaskSize < 1 {
		return config.DefaultTaskSize
	}
	return m.layout.TaskSize
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
	if m.reviewMode {
		return m.reviewExplainWidth()
	}
	if m.showSidePanel() && !m.sidePanelHorizontal() {
		return m.baseOutputWidth() - m.sidePanelWidth()
	}
	return m.baseOutputWidth()
}

// sidePanelHorizontal reports whether the side panel is the diff panel drawn on
// a horizontal side, so it takes rows below or above the output, not columns
// beside it. The task panel is always a vertical side.
func (m Model) sidePanelHorizontal() bool {
	return m.diffPanel && config.DiffHorizontal(m.layout.DiffPosition)
}

// sidePanelWidth is the width of a vertical side panel: the resizable diff panel
// when it is open on the left or right, else the fixed jobs and tasks panel.
func (m Model) sidePanelWidth() int {
	if m.diffPanel {
		return m.diffPanelWidth()
	}
	return m.taskCols()
}

// sidePanelHeight is the rows of the diff panel when it is on a horizontal side.
func (m Model) sidePanelHeight() int {
	return m.diffPanelHeight()
}

// showSidePanel says whether the side panel has room by the output. It reads
// baseOutputWidth and bodyHeight, not outputWidth and outputHeight, because
// those depend on it. See docs/tui/tasks.md and docs/tui/diff.md.
func (m Model) showSidePanel() bool {
	if m.reviewMode {
		return false
	}
	if m.diffPanel {
		if m.sidePanelHorizontal() {
			return m.bodyHeight()-barHeight-m.sidePanelHeight() >= minOutputHeightWithPanel
		}
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
	prompt := withEdge(promptPanelStyle.Width(m.width-gutterWidth).Render(m.promptView()), m.focus == focusPrompt)
	return lipgloss.JoinVertical(lipgloss.Left, body, prompt, m.statusView())
}

// A session dialog draws in the pane, not over the whole body; see docs/tui.md.
func (m Model) paneView() string {
	if dialog, ok := m.sessionDialogView(); ok {
		return lipgloss.JoinVertical(lipgloss.Left, m.barView(), dialog)
	}
	if m.reviewMode {
		return m.reviewView()
	}
	if !m.showSidePanel() {
		return lipgloss.JoinVertical(lipgloss.Left, m.barView(), m.outputView())
	}
	panel := m.sidePanelView()
	if m.sidePanelHorizontal() {
		if m.layout.DiffPosition == config.DiffTop {
			return lipgloss.JoinVertical(lipgloss.Left, m.barView(), panel, m.outputView())
		}
		return lipgloss.JoinVertical(lipgloss.Left, m.barView(), m.outputView(), panel)
	}
	pane := lipgloss.JoinVertical(lipgloss.Left, m.barView(), m.outputView())
	if m.diffPanel && m.layout.DiffPosition == config.DiffLeft {
		return lipgloss.JoinHorizontal(lipgloss.Top, panel, pane)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, pane, panel)
}

func (m Model) sessionDialogView() (string, bool) {
	width, height := m.baseOutputWidth(), m.outputHeight()
	if m.confirm != "" {
		return centre(width, height, m.confirmView(width)), true
	}
	if m.modal != nil && m.modal.region() == modalPane {
		return centre(width, height, m.modal.view(width, height)), true
	}
	return "", false
}

func (m Model) bodyDialogView() (string, bool) {
	width, height := m.width, m.bodyHeight()
	if m.help != nil {
		return centre(width, height, m.help.View(m.keys, width, height)), true
	}
	if m.form != nil {
		return centre(width, height, m.form.View(width)), true
	}
	if m.modal != nil && m.modal.region() == modalBody {
		return centre(width, height, m.modal.view(width, height)), true
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

	if m.searchActive() {
		rows = append(rows, m.searchView())
	}

	visible := m.visibleLines()
	drawn := 0
	for i := m.listOffset; i < len(m.lines) && drawn < visible; i++ {
		line := m.lines[i]
		drawn++
		if line.isDivider() {
			rows = append(rows, m.sectionDivider(line.divider))
			continue
		}
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
	counts := ""
	if item.jobs > 0 {
		counts += fmt.Sprintf(" ⚙%d", item.jobs)
	}
	if item.queued > 0 {
		counts += fmt.Sprintf(" ⇢%d", item.queued)
	}
	flagText := ""
	if flags := rowFlags(item); flags != "" {
		flagText = " " + flags
	}
	nameWidth := width - 3 - lipgloss.Width(flagText) - lipgloss.Width(counts)
	if nameWidth < 1 {
		nameWidth = 1
	}
	glyph := rowGlyph(item, m.spinFrame)
	if item.name == m.sel {
		rest := " " + pad(item.displayName(), nameWidth) + flagText + counts
		return selectedRowStyle.Render(" ") +
			item.style().Background(lipgloss.Color("62")).Render(glyph) +
			selectedRowStyle.Width(width-2).Render(rest)
	}
	nameStyle := rowStyle
	if item.archived || item.hosted {
		nameStyle = rowMutedStyle
	}
	return " " + item.style().Render(glyph) +
		nameStyle.Render(" "+pad(item.displayName(), nameWidth)) +
		rowMutedStyle.Render(flagText) +
		nameStyle.Render(counts)
}

// rowFlags is the muted single-letter flags for a session row, concatenated in a
// fixed order: held, read-only, watched, hoisted, scheduled, control. A control session
// that heads its own group takes no flag, because the group header already marks
// it. See docs/tui/sessions.md.
func rowFlags(item row) string {
	flags := ""
	if item.held {
		flags += heldMark
	}
	if item.readOnly {
		flags += readOnlyMark
	}
	if item.watched {
		flags += watchedMark
	}
	if item.lender != "" {
		flags += hoistMark
	}
	if item.scheduled != "" {
		flags += scheduleMark
	}
	if item.control && !headsGroup(item) {
		flags += controlMark
	}
	return flags
}

// sectionDivider draws a section band header: a label between horizontal rules.
// The "remote sessions" parent takes a brighter style than its sub-bands. See
// docs/peers.md.
func (m Model) sectionDivider(label string) string {
	width := m.sidebarInnerCols()
	labelStyle := sectionLabelStyle
	if label == "remote sessions" {
		labelStyle = sectionParentStyle
	}
	text := " " + label + " "
	dashes := width - lipgloss.Width(text) - 1
	if dashes < 0 {
		dashes = 0
	}
	return sectionRuleStyle.Render("─") + labelStyle.Render(text) + sectionRuleStyle.Render(strings.Repeat("─", dashes))
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
	labelStyle := groupLabelStyle
	if item.section == sectionHosted {
		labelStyle = groupMutedStyle
	}
	count := strconv.Itoa(item.count)
	width := m.sidebarInnerCols() - 3 - lipgloss.Width(glyph) - len(count)
	if width < 1 {
		width = 1
	}
	return groupMarkStyle.Render(mark) + " " +
		labelStyle.Render(pad(label, width)) + " " +
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
	lines := m.taskPanelLines()
	height := m.bodyHeight()
	scroll := clampScroll(m.taskScroll, len(lines), height)
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	block := strings.Join(lines[scroll:end], "\n")
	return sidePanelStyle(m.focus == focusTask).Width(m.taskCols() - 1).Height(height).Render(block)
}

func (m Model) taskPanelLines() []string {
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
	return rows
}

func (m *Model) clampTaskScroll() {
	m.taskScroll = clampScroll(m.taskScroll, len(m.taskPanelLines()), m.bodyHeight())
}

// sidePanelStyle is the border of a side panel: the highlight colour when the
// panel holds the focus, and the muted colour otherwise. The task panel and the
// diff panel share it.
func sidePanelStyle(focused bool) lipgloss.Style {
	if focused {
		return taskPanelStyle.BorderForeground(lipgloss.Color("62"))
	}
	return taskPanelStyle
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

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func (m Model) promptView() string {
	label := "prompt"
	if m.sel != "" {
		label = m.sel
	}
	if m.reviewMode {
		if m.reviewFocus == reviewPrompt {
			return promptLabelStyle.Render(label+" — follow-up ⌁ ") + "\n" + m.prompt.View()
		}
		return hintStyle.Render(label+" — tab to the prompt to ask a follow-up") + "\n" + m.prompt.View()
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
