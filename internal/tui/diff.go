package tui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

// diffRefreshInterval is how often the open panel re-reads the diff, so the
// changes of a running agent show while it works. See docs/tui/diff.md.
const diffRefreshInterval = 800 * time.Millisecond

type diffTickMsg struct{}

func diffTick() tea.Cmd {
	return tea.Tick(diffRefreshInterval, func(time.Time) tea.Msg { return diffTickMsg{} })
}

// handleDiffTick re-reads the diff while the panel is open, then schedules the
// next tick. It stops the loop when the panel closes.
func (m Model) handleDiffTick() (tea.Model, tea.Cmd) {
	if !m.diffPanel {
		m.diffTicking = false
		return m, nil
	}
	return m, tea.Batch(m.diffRefreshCmd(), diffTick())
}

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
	cmds := []tea.Cmd{m.diffRefreshCmd()}
	if !m.diffTicking {
		m.diffTicking = true
		cmds = append(cmds, diffTick())
	}
	return m, tea.Batch(cmds...)
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
	case "k":
		return m.diffCursorUp()
	case "j":
		return m.diffCursorDown()
	case "up":
		m.diffScroll--
		m.clampDiffScroll()
		return m, nil
	case "down":
		m.diffScroll++
		m.clampDiffScroll()
		return m, nil
	case "pgup":
		m.diffScroll -= m.diffPage()
		m.clampDiffScroll()
		return m, nil
	case "pgdown":
		m.diffScroll += m.diffPage()
		m.clampDiffScroll()
		return m, nil
	case "g":
		if m.anyDiffOpen() {
			m.diffScroll = 0
			return m, nil
		}
		m.diffSel = 0
		m.diffScroll = 0
		return m, nil
	case "G":
		if m.anyDiffOpen() {
			m.diffScroll = len(m.diffPanelLines())
			m.clampDiffScroll()
			return m, nil
		}
		m.diffSel = len(files) - 1
		m.clampDiffSel()
		m.ensureDiffSelVisible()
		return m, nil
	case "}", "shift+]":
		return m.diffJumpDown()
	case "{", "shift+[":
		return m.diffJumpUp()
	case "enter":
		return m.toggleDiffFile()
	}
	return m, nil
}

// diffJumpDown and diffJumpUp are } and {. They scroll to the next and the
// previous empty line of an open diff, like vim's paragraph motions. See
// docs/tui/diff.md.
func (m Model) diffJumpDown() (tea.Model, tea.Cmd) {
	for _, row := range m.diffEmptyRows() {
		if row > m.diffScroll {
			m.diffScroll = row
			m.clampDiffScroll()
			break
		}
	}
	return m, nil
}

func (m Model) diffJumpUp() (tea.Model, tea.Cmd) {
	rows := m.diffEmptyRows()
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i] < m.diffScroll {
			m.diffScroll = rows[i]
			m.clampDiffScroll()
			break
		}
	}
	return m, nil
}

// diffEmptyRows lists, in order, the panel-line indices that show an empty line
// of an open diff. It walks the panel the same way as diffPanelLines.
func (m Model) diffEmptyRows() []int {
	d, ok := m.diffs[m.sel]
	if !ok || !d.repo {
		return nil
	}
	var rows []int
	line := 2
	for _, file := range d.files {
		line++
		if !m.diffOpen[m.sel][file.Path] {
			continue
		}
		for _, out := range m.diffFileBody(file.Path) {
			if m.diffRowEmpty(out) {
				rows = append(rows, line)
			}
			line++
		}
	}
	return rows
}

// diffRowEmpty reports whether a rendered diff line has no code content, once its
// line-number gutter and its +/− marker are removed.
func (m Model) diffRowEmpty(rendered string) bool {
	s := ansi.Strip(rendered)
	if m.diffLineNumbers && len(s) >= diffNumGutter {
		s = s[diffNumGutter:]
	}
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimPrefix(s, "-")
	return strings.TrimSpace(s) == ""
}

// anyDiffOpen reports whether the selected session has an expanded file. It
// scopes g and G: to the diff when a file is open, else to the file list.
func (m Model) anyDiffOpen() bool {
	return len(m.diffOpen[m.sel]) > 0
}

// diffCursorDown and diffCursorUp are j and k. On an open file, they scroll the
// diff one line, so the content moves under a fixed viewport. At the bottom of an
// open diff, j moves to the next file. k moves to the previous file, and enters
// an open file at the bottom of its diff. See docs/tui/diff.md.
func (m Model) diffCursorDown() (tea.Model, tea.Cmd) {
	files := m.diffs[m.sel].files
	if body := m.selectedBodyLen(); body > 0 {
		last := m.diffFileLine(m.diffSel) + body
		if last >= m.diffScroll+m.bodyHeight() {
			m.diffScroll++
			m.clampDiffScroll()
			return m, nil
		}
	}
	if m.diffSel < len(files)-1 {
		m.diffSel++
		m.ensureDiffSelVisible()
	}
	return m, nil
}

func (m Model) diffCursorUp() (tea.Model, tea.Cmd) {
	if m.selectedBodyLen() > 0 && m.diffFileLine(m.diffSel) < m.diffScroll {
		m.diffScroll--
		m.clampDiffScroll()
		return m, nil
	}
	if m.diffSel > 0 {
		m.diffSel--
		if body := m.selectedBodyLen(); body > 0 {
			m.diffScroll = m.diffFileLine(m.diffSel) + body - m.bodyHeight() + 1
			m.clampDiffScroll()
		} else {
			m.ensureDiffSelVisible()
		}
	}
	return m, nil
}

// selectedBodyLen is the count of rendered diff lines of the selected file when
// it is expanded, or 0 when it is collapsed.
func (m Model) selectedBodyLen() int {
	files := m.diffs[m.sel].files
	if m.diffSel < 0 || m.diffSel >= len(files) {
		return 0
	}
	path := files[m.diffSel].Path
	if !m.diffOpen[m.sel][path] {
		return 0
	}
	return len(m.diffFileBody(path))
}

// widenDiff, narrowDiff, toggleHalfDiff, and toggleDiffNumbers are the d +, d -,
// d /, and d n actions. The width persists on the model, so a hide and a later
// show keep it.
func (m Model) widenDiff() (tea.Model, tea.Cmd) { return m.resizeDiff(diffWidthStep) }

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
func (m Model) toggleSidebar() (tea.Model, tea.Cmd) { return m.setSidebar(!m.sidebarHidden) }

func (m Model) collapseSidebar() (tea.Model, tea.Cmd) { return m.setSidebar(true) }

func (m Model) expandSidebar() (tea.Model, tea.Cmd) { return m.setSidebar(false) }

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
	return m.diffFileLine(m.diffSel)
}

// diffFileLine is the line index of a file's row within the panel lines.
func (m Model) diffFileLine(target int) int {
	line := 2
	for i, file := range m.diffs[m.sel].files {
		if i == target {
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
