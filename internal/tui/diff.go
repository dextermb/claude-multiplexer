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
	lines := renderDiffLines(msg.text)
	if msg.err != nil {
		lines = []string{diffMetaStyle.Render("  diff failed: " + msg.err.Error())}
	}
	if m.fileDiffs[msg.name] == nil {
		m.fileDiffs[msg.name] = make(map[string][]string)
	}
	m.fileDiffs[msg.name][msg.path] = lines
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
		return m, nil
	case "down", "j":
		if m.diffSel < len(files)-1 {
			m.diffSel++
		}
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
	limit := len(m.diffPanelLines()) - m.bodyHeight()
	if limit < 0 {
		limit = 0
	}
	if m.diffScroll > limit {
		m.diffScroll = limit
	}
	if m.diffScroll < 0 {
		m.diffScroll = 0
	}
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

func (m Model) diffPanelView() string {
	lines := m.diffPanelLines()
	height := m.bodyHeight()
	limit := len(lines) - height
	if limit < 0 {
		limit = 0
	}
	scroll := m.diffScroll
	if scroll > limit {
		scroll = limit
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	block := strings.Join(lines[scroll:end], "\n")
	return taskPanelStyle.Width(taskPanelWidth - 1).Height(height).Render(block)
}

func (m Model) diffPanelLines() []string {
	d, ok := m.diffs[m.sel]
	if !ok {
		return []string{diffMetaStyle.Render("reading…")}
	}
	if !d.repo {
		return []string{diffMetaStyle.Render("not a git repository")}
	}
	head := taskHeaderStyle.Render("Changes · "+strconv.Itoa(len(d.files))) + ""
	out := []string{head, ""}
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
	glyph := foldShutMark
	if m.diffOpen[m.sel][file.Path] {
		glyph = foldOpenMark
	}
	counts := diffAddStyle.Render("+"+strconv.Itoa(file.Insertions)) + " " +
		diffDelStyle.Render("−"+strconv.Itoa(file.Deletions))
	head := glyph + " " + file.Status + " "
	room := taskPanelInner - lipgloss.Width(head) - lipgloss.Width(counts) - 1
	if room < 4 {
		room = 4
	}
	pathStyle := rowStyle
	if index == m.diffSel && m.focus == focusDiff {
		pathStyle = diffPathStyle
	}
	left := diffMetaStyle.Render(head) + pathStyle.Render(truncate(file.Path, room))
	gap := taskPanelInner - lipgloss.Width(left) - lipgloss.Width(counts)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + counts
}

func (m Model) diffFileBody(path string) []string {
	lines, ok := m.fileDiffs[m.sel][path]
	if !ok {
		return []string{diffMetaStyle.Render("  reading…")}
	}
	return lines
}

// renderDiffLines colours a raw git diff and wraps it to the panel width. It
// drops the git file header, because a narrow panel has no room for it.
func renderDiffLines(text string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if isDiffHeader(line) {
			continue
		}
		out = append(out, styleDiffLine(line)...)
	}
	if len(out) == 0 {
		return []string{diffMetaStyle.Render("  (no text change)")}
	}
	return out
}

func isDiffHeader(line string) bool {
	for _, prefix := range []string{"diff --git", "index ", "--- ", "+++ ", "new file", "deleted file", "old mode", "new mode", "similarity ", "rename ", "copy "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func styleDiffLine(line string) []string {
	style := rowStyle
	switch {
	case strings.HasPrefix(line, "@@"):
		style = diffHunkStyle
	case strings.HasPrefix(line, "+"):
		style = diffAddStyle
	case strings.HasPrefix(line, "-"):
		style = diffDelStyle
	}
	chunks := wrapHard(line, taskPanelInner)
	for i := range chunks {
		chunks[i] = style.Render(chunks[i])
	}
	return chunks
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
