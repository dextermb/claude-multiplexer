package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/git"
)

// diffState is the cached working-tree diff of one session against origin/HEAD.
// See docs/tui/diff.md.
type diffState struct {
	repo  bool
	stat  git.Stat
	files []git.FileChange
	err   error
}

type diffMsg struct {
	name  string
	repo  bool
	stat  git.Stat
	files []git.FileChange
	err   error
}

type fileDiffMsg struct {
	name string
	path string
	text string
	err  error
}

// diffRefreshCmd reads the diff of the selected session.
func (m Model) diffRefreshCmd() tea.Cmd {
	item, ok := m.selectedRow()
	if !ok {
		return nil
	}
	return diffCmd(item.name, item.openDir())
}

func diffCmd(name, dir string) tea.Cmd {
	return func() tea.Msg {
		if dir == "" || !git.IsRepo(dir) {
			return diffMsg{name: name}
		}
		stat, files, err := git.Diff(dir)
		return diffMsg{name: name, repo: true, stat: stat, files: files, err: err}
	}
}

func fileDiffCmd(name, dir, path string) tea.Cmd {
	return func() tea.Msg {
		text, err := git.FileDiff(dir, path)
		return fileDiffMsg{name: name, path: path, text: text, err: err}
	}
}

func (m Model) handleDiff(msg diffMsg) (tea.Model, tea.Cmd) {
	m.diffs[msg.name] = diffState{repo: msg.repo, stat: msg.stat, files: msg.files, err: msg.err}
	m.clampDiffSel()
	var cmds []tea.Cmd
	if item, ok := m.rowByName(msg.name); ok {
		for path := range m.diffOpen[msg.name] {
			cmds = append(cmds, fileDiffCmd(msg.name, item.openDir(), path))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleFileDiff(msg fileDiffMsg) (tea.Model, tea.Cmd) {
	text := msg.text
	if msg.err != nil {
		text = "diff failed: " + msg.err.Error()
	}
	if m.fileDiffs[msg.name] == nil {
		m.fileDiffs[msg.name] = make(map[string]string)
	}
	m.fileDiffs[msg.name][msg.path] = text
	return m, nil
}

// toggleDiffPanel is the s d action. It opens the panel, focuses it, or closes
// it, in that order.
func (m Model) toggleDiffPanel() (tea.Model, tea.Cmd) {
	if !m.diffPanel {
		return m.openDiffPanel()
	}
	if m.focus != focusDiff {
		m.focus = focusDiff
		m.prompt.Blur()
		return m, nil
	}
	return m.closeDiffPanel()
}

func (m Model) openDiffPanel() (tea.Model, tea.Cmd) {
	m.diffPanel = true
	m.diffSel = 0
	m.diffScroll = 0
	m.focus = focusDiff
	m.prompt.Blur()
	m.rebuildOutput()
	return m, m.diffRefreshCmd()
}

func (m Model) closeDiffPanel() (tea.Model, tea.Cmd) {
	m.diffPanel = false
	if m.focus == focusDiff {
		m.focus = focusOutput
	}
	m.rebuildOutput()
	return m, nil
}

func (m Model) diffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	files := m.diffs[m.sel].files
	switch msg.String() {
	case "esc":
		return m.closeDiffPanel()
	case "tab":
		return m.toggleFocus()
	case "up", "k":
		if m.diffSel > 0 {
			m.diffSel--
		}
		m.ensureDiffSelVisible()
		return m, nil
	case "down", "j":
		if m.diffSel < len(files)-1 {
			m.diffSel++
		}
		m.ensureDiffSelVisible()
		return m, nil
	case "pgup":
		m.diffScroll -= m.diffPage()
		m.clampDiffScroll()
		return m, nil
	case "pgdown":
		m.diffScroll += m.diffPage()
		m.clampDiffScroll()
		return m, nil
	case "enter":
		return m.toggleDiffFile()
	}
	return m, nil
}

// widenDiff, narrowDiff, toggleHalfDiff, and toggleDiffNumbers are the d +, d -,
// d /, and d n actions. The width persists on the model, so a hide and a later
// show keep it.
func (m Model) widenDiff() (tea.Model, tea.Cmd)  { return m.resizeDiff(diffWidthStep) }
func (m Model) narrowDiff() (tea.Model, tea.Cmd) { return m.resizeDiff(-diffWidthStep) }

func (m Model) resizeDiff(delta int) (tea.Model, tea.Cmd) {
	m.diffWidth = m.clampDiffWidth(m.diffPanelWidth() + delta)
	m.diffHalf = false
	m.rebuildOutput()
	return m, nil
}

// toggleHalfDiff is the d / action. It shows the panel at half the screen, or
// returns it to the set width.
func (m Model) toggleHalfDiff() (tea.Model, tea.Cmd) {
	m.diffHalf = !m.diffHalf
	m.rebuildOutput()
	return m, nil
}

func (m Model) toggleDiffNumbers() (tea.Model, tea.Cmd) {
	m.diffLineNumbers = !m.diffLineNumbers
	return m, nil
}

func (m Model) toggleDiffFile() (tea.Model, tea.Cmd) {
	files := m.diffs[m.sel].files
	if m.diffSel < 0 || m.diffSel >= len(files) {
		return m, nil
	}
	path := files[m.diffSel].Path
	open := m.diffOpen[m.sel]
	if open == nil {
		open = make(map[string]bool)
		m.diffOpen[m.sel] = open
	}
	if open[path] {
		delete(open, path)
		return m, nil
	}
	open[path] = true
	if _, cached := m.fileDiffs[m.sel][path]; cached {
		return m, nil
	}
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	return m, fileDiffCmd(m.sel, item.openDir(), path)
}

// toggleSidebar, collapseSidebar, and expandSidebar are the l t, l c, and l e
// actions. They hide or show the left sidebar, to give the panes more width.
func (m Model) toggleSidebar() (tea.Model, tea.Cmd)   { return m.setSidebar(!m.sidebarHidden) }
func (m Model) collapseSidebar() (tea.Model, tea.Cmd) { return m.setSidebar(true) }
func (m Model) expandSidebar() (tea.Model, tea.Cmd)   { return m.setSidebar(false) }

func (m Model) setSidebar(hidden bool) (tea.Model, tea.Cmd) {
	m.sidebarHidden = hidden
	if hidden && m.focus == focusSidebar {
		m.focus = focusOutput
	}
	m.output.Width = m.outputWidth()
	m.rebuildOutput()
	return m, nil
}

func (m *Model) clampDiffSel() {
	n := len(m.diffs[m.sel].files)
	if m.diffSel >= n {
		m.diffSel = n - 1
	}
	if m.diffSel < 0 {
		m.diffSel = 0
	}
}

func (m *Model) clampDiffScroll() {
	m.diffScroll = clampScroll(m.diffScroll, len(m.diffPanelLines()), m.bodyHeight())
}

// ensureDiffSelVisible scrolls the panel so the selected file row is on screen.
func (m *Model) ensureDiffSelVisible() {
	line := m.diffSelLine()
	height := m.bodyHeight()
	if line < m.diffScroll {
		m.diffScroll = line
	}
	if line >= m.diffScroll+height {
		m.diffScroll = line - height + 1
	}
	m.clampDiffScroll()
}

// diffSelLine is the line index of the selected file row within the panel lines.
func (m Model) diffSelLine() int {
	line := 2
	for i, file := range m.diffs[m.sel].files {
		if i == m.diffSel {
			return line
		}
		line++
		if m.diffOpen[m.sel][file.Path] {
			line += len(m.diffFileBody(file.Path))
		}
	}
	return line
}

func (m Model) diffPage() int {
	page := m.bodyHeight() - 2
	if page < 1 {
		page = 1
	}
	return page
}

func (m Model) rowByName(name string) (row, bool) {
	for _, item := range m.rows {
		if item.name == name {
			return item, true
		}
	}
	return row{}, false
}

func barDiffCount(stat git.Stat) string {
	plus := barAddStyle.Render("+" + strconv.Itoa(stat.Insertions))
	minus := barDelStyle.Render("−" + strconv.Itoa(stat.Deletions))
	return plus + barStyle.Render(" ") + minus
}

const (
	diffWidthStep = 6
	diffWidthMin  = 20
	diffNumGutter = 5
)

// diffPanelWidth is the width of the diff panel: half the screen in the half
// mode, else the remembered width, or the default when none is set yet.
func (m Model) diffPanelWidth() int {
	if m.diffHalf {
		return m.clampDiffWidth(m.width / 2)
	}
	if m.diffWidth <= 0 {
		return taskPanelWidth
	}
	return m.clampDiffWidth(m.diffWidth)
}

// clampDiffWidth keeps the panel wide enough to read, but not so wide that the
// output falls below its minimum.
func (m Model) clampDiffWidth(width int) int {
	most := m.baseOutputWidth() - minOutputWithPanel
	if most < diffWidthMin {
		most = diffWidthMin
	}
	if width < diffWidthMin {
		width = diffWidthMin
	}
	if width > most {
		width = most
	}
	return width
}

func (m Model) diffInner() int {
	return m.diffPanelWidth() - 2
}

func (m Model) diffPanelView() string {
	lines := m.diffPanelLines()
	height := m.bodyHeight()
	scroll := clampScroll(m.diffScroll, len(lines), height)
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	block := strings.Join(lines[scroll:end], "\n")
	return taskPanelStyle.Width(m.diffPanelWidth() - 1).Height(height).Render(block)
}

func clampScroll(scroll, total, height int) int {
	limit := total - height
	if limit < 0 {
		limit = 0
	}
	if scroll > limit {
		scroll = limit
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}

func (m Model) diffPanelLines() []string {
	d, ok := m.diffs[m.sel]
	if !ok {
		return []string{diffMetaStyle.Render("reading…")}
	}
	if !d.repo {
		return []string{diffMetaStyle.Render("not a git repository")}
	}
	out := []string{taskHeaderStyle.Render("Changes · " + strconv.Itoa(len(d.files))), ""}
	if len(d.files) == 0 {
		return append(out, diffMetaStyle.Render("no changes"))
	}
	for i, file := range d.files {
		out = append(out, m.diffFileRow(i, file))
		if m.diffOpen[m.sel][file.Path] {
			out = append(out, m.diffFileBody(file.Path)...)
		}
	}
	return out
}

func (m Model) diffFileRow(index int, file git.FileChange) string {
	inner := m.diffInner()
	glyph := foldShutMark
	if m.diffOpen[m.sel][file.Path] {
		glyph = foldOpenMark
	}
	head := glyph + " " + file.Status + " "
	countsText := "+" + strconv.Itoa(file.Insertions) + " −" + strconv.Itoa(file.Deletions)
	room := inner - lipgloss.Width(head) - lipgloss.Width(countsText) - 1
	if room < 4 {
		room = 4
	}
	name := truncate(file.Path, room)

	if index == m.diffSel {
		text := head + name
		gap := inner - lipgloss.Width(text) - lipgloss.Width(countsText)
		if gap < 1 {
			gap = 1
		}
		return selectedRowStyle.Width(inner).Render(text + strings.Repeat(" ", gap) + countsText)
	}

	counts := diffAddStyle.Render("+"+strconv.Itoa(file.Insertions)) + " " +
		diffDelStyle.Render("−"+strconv.Itoa(file.Deletions))
	left := diffMetaStyle.Render(head) + rowStyle.Render(name)
	gap := inner - lipgloss.Width(left) - lipgloss.Width(counts)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + counts
}

func (m Model) diffFileBody(path string) []string {
	text, ok := m.fileDiffs[m.sel][path]
	if !ok {
		return []string{diffMetaStyle.Render("  reading…")}
	}
	return m.renderDiffBody(text)
}

// renderDiffBody colours a raw git diff and wraps it to the panel width. It
// drops the git file header, and shows new-file line numbers when they are on.
func (m Model) renderDiffBody(text string) []string {
	content := m.diffInner()
	if m.diffLineNumbers {
		content -= diffNumGutter
	}
	if content < 1 {
		content = 1
	}
	var out []string
	newLine := 0
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if isDiffHeader(line) {
			continue
		}
		style := rowStyle
		number := ""
		switch {
		case strings.HasPrefix(line, "@@"):
			style = diffHunkStyle
			newLine = hunkNewStart(line)
		case strings.HasPrefix(line, "+"):
			style = diffAddStyle
			number = strconv.Itoa(newLine)
			newLine++
		case strings.HasPrefix(line, "-"):
			style = diffDelStyle
		default:
			number = strconv.Itoa(newLine)
			newLine++
		}
		for i, chunk := range wrapHard(line, content) {
			row := style.Render(chunk)
			if m.diffLineNumbers {
				gutter := ""
				if i == 0 {
					gutter = number
				}
				row = diffNumStyle.Render(padLeft(gutter, diffNumGutter-1)+" ") + row
			}
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		return []string{diffMetaStyle.Render("  (no text change)")}
	}
	return out
}

// hunkNewStart reads the new-file start line of a hunk header, from the number
// after the "+", for example 12 in "@@ -3,4 +12,6 @@".
func hunkNewStart(line string) int {
	plus := strings.IndexByte(line, '+')
	if plus < 0 {
		return 0
	}
	rest := line[plus+1:]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(rest[:end])
	return n
}

func padLeft(text string, width int) string {
	if pad := width - len(text); pad > 0 {
		return strings.Repeat(" ", pad) + text
	}
	return text
}

func isDiffHeader(line string) bool {
	for _, prefix := range []string{"diff --git", "index ", "--- ", "+++ ", "new file", "deleted file", "old mode", "new mode", "similarity ", "rename ", "copy "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func wrapHard(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{""}
	}
	var out []string
	for len(runes) > width {
		out = append(out, string(runes[:width]))
		runes = runes[width:]
	}
	return append(out, string(runes))
}
