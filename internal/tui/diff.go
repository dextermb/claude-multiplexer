package tui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/dextermb/claude-multiplexer/internal/config"
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

// dirDiff is the cached working-tree diff of one directory against origin/HEAD.
// See docs/tui/diff.md.
type dirDiff struct {
	dir   string
	repo  bool
	stat  git.Stat
	files []git.FileChange
	err   error
}

// projectDiff is the diff of a session, one group per directory of its project.
// A session with no project has one group. See docs/tui/diff.md.
type projectDiff struct {
	groups []dirDiff
}

// fileKey names one changed file within a project: its directory and its path.
// The directory keeps two files of the same path in different directories
// apart. See docs/tui/diff.md.
type fileKey struct {
	dir  string
	path string
}

// diffEntry is one selectable file of the panel, flattened across the groups in
// order. The navigation indexes this list.
type diffEntry struct {
	dir  string
	file git.FileChange
}

func (p projectDiff) entries() []diffEntry {
	var out []diffEntry
	for _, g := range p.groups {
		for _, f := range g.files {
			out = append(out, diffEntry{dir: g.dir, file: f})
		}
	}
	return out
}

func (p projectDiff) totalFiles() int {
	n := 0
	for _, g := range p.groups {
		n += len(g.files)
	}
	return n
}

func (p projectDiff) anyRepo() bool {
	for _, g := range p.groups {
		if g.repo {
			return true
		}
	}
	return false
}

func (p projectDiff) stat() git.Stat {
	var out git.Stat
	for _, g := range p.groups {
		out.Insertions += g.stat.Insertions
		out.Deletions += g.stat.Deletions
		out.Files += g.stat.Files
	}
	return out
}

func (m Model) diffEntries() []diffEntry {
	return m.diffs[m.sel].entries()
}

// multiGroup reports whether the selected session's diff has more than one
// group, so the panel draws a header for each directory.
func (m Model) multiGroup() bool {
	return len(m.diffs[m.sel].groups) > 1
}

func (e diffEntry) key() fileKey {
	return fileKey{dir: e.dir, path: e.file.Path}
}

type diffMsg struct {
	name   string
	groups []dirDiff
}

type fileDiffMsg struct {
	name string
	dir  string
	path string
	text string
	err  error
}

// diffRefreshCmd reads the diff of every directory of the selected session's
// project.
func (m Model) diffRefreshCmd() tea.Cmd {
	item, ok := m.selectedRow()
	if !ok {
		return nil
	}
	return projectDiffCmd(item.name, item.diffDirs())
}

func projectDiffCmd(name string, dirs []string) tea.Cmd {
	return func() tea.Msg {
		groups := make([]dirDiff, 0, len(dirs))
		for _, dir := range dirs {
			if dir == "" || !git.IsRepo(dir) {
				groups = append(groups, dirDiff{dir: dir})
				continue
			}
			stat, files, err := git.Diff(dir)
			groups = append(groups, dirDiff{dir: dir, repo: true, stat: stat, files: files, err: err})
		}
		return diffMsg{name: name, groups: groups}
	}
}

func fileDiffCmd(name, dir, path string) tea.Cmd {
	return func() tea.Msg {
		text, err := git.FileDiff(dir, path)
		return fileDiffMsg{name: name, dir: dir, path: path, text: text, err: err}
	}
}

func (m Model) handleDiff(msg diffMsg) (tea.Model, tea.Cmd) {
	m.diffs[msg.name] = projectDiff{groups: msg.groups}
	m.clampDiffSel()
	var cmds []tea.Cmd
	for key := range m.diffOpen[msg.name] {
		cmds = append(cmds, fileDiffCmd(msg.name, key.dir, key.path))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleFileDiff(msg fileDiffMsg) (tea.Model, tea.Cmd) {
	text := msg.text
	if msg.err != nil {
		text = "diff failed: " + msg.err.Error()
	}
	if m.fileDiffs[msg.name] == nil {
		m.fileDiffs[msg.name] = make(map[fileKey]string)
	}
	m.fileDiffs[msg.name][fileKey{dir: msg.dir, path: msg.path}] = text
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
	entries := m.diffEntries()
	switch msg.String() {
	case "esc":
		return m.closeDiffPanel()
	case "tab":
		return m.toggleFocus()
	case "k":
		if m.diffHorizontal() {
			return m.diffGridMove(-m.gridCols())
		}
		return m.diffCursorUp()
	case "j":
		if m.diffHorizontal() {
			return m.diffGridMove(m.gridCols())
		}
		return m.diffCursorDown()
	case "h":
		if m.diffHorizontal() {
			return m.diffGridMove(-1)
		}
		return m, nil
	case "l":
		if m.diffHorizontal() {
			return m.diffGridMove(1)
		}
		return m, nil
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
		m.diffSel = len(entries) - 1
		m.clampDiffSel()
		m.ensureDiffSelVisible()
		return m, nil
	case "}", "shift+]":
		return m.diffJumpDown()
	case "{", "shift+[":
		return m.diffJumpUp()
	case "enter", " ":
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
// of an open diff. On a horizontal side it scans the rendered lines, because the
// open diffs sit below the grid, not inline.
func (m Model) diffEmptyRows() []int {
	p, ok := m.diffs[m.sel]
	if !ok || !p.anyRepo() {
		return nil
	}
	if m.diffHorizontal() {
		var rows []int
		for i, line := range m.diffPanelLines() {
			if m.diffRowEmpty(line) {
				rows = append(rows, i)
			}
		}
		return rows
	}
	multi := m.multiGroup()
	var rows []int
	line := 2
	for _, g := range p.groups {
		if multi {
			line++
		}
		if !g.repo || len(g.files) == 0 {
			if multi {
				line++
			}
			continue
		}
		for _, file := range g.files {
			line++
			key := fileKey{dir: g.dir, path: file.Path}
			if !m.diffOpen[m.sel][key] {
				continue
			}
			for _, out := range m.diffFileBody(key) {
				if m.diffRowEmpty(out) {
					rows = append(rows, line)
				}
				line++
			}
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
	entries := m.diffEntries()
	if body := m.selectedBodyLen(); body > 0 {
		last := m.diffFileLine(m.diffSel) + body
		if last >= m.diffScroll+m.bodyHeight() {
			m.diffScroll++
			m.clampDiffScroll()
			return m, nil
		}
	}
	if m.diffSel < len(entries)-1 {
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
	entries := m.diffEntries()
	if m.diffSel < 0 || m.diffSel >= len(entries) {
		return 0
	}
	key := entries[m.diffSel].key()
	if !m.diffOpen[m.sel][key] {
		return 0
	}
	return len(m.diffFileBody(key))
}

// widenDiff, narrowDiff, toggleHalfDiff, and toggleDiffNumbers are the d +, d -,
// d /, and d n actions. The size persists on the model, so a hide and a later
// show keep it. The keys grow the width on a vertical side and the height on a
// horizontal side. See docs/tui/diff.md.
func (m Model) widenDiff() (tea.Model, tea.Cmd) { return m.resizeDiff(diffSizeStep) }

func (m Model) narrowDiff() (tea.Model, tea.Cmd) { return m.resizeDiff(-diffSizeStep) }

func (m Model) resizeDiff(delta int) (tea.Model, tea.Cmd) {
	m.diffSize = m.clampDiffExtent(m.diffPanelExtent() + delta)
	m.diffHalf = false
	m.rebuildOutput()
	return m, nil
}

// toggleHalfDiff is the d / action. It shows the panel at half the screen, or
// returns it to the set size.
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
	entries := m.diffEntries()
	if m.diffSel < 0 || m.diffSel >= len(entries) {
		return m, nil
	}
	entry := entries[m.diffSel]
	key := entry.key()
	open := m.diffOpen[m.sel]
	if open == nil {
		open = make(map[fileKey]bool)
		m.diffOpen[m.sel] = open
	}
	if open[key] {
		delete(open, key)
		return m, nil
	}
	open[key] = true
	if _, cached := m.fileDiffs[m.sel][key]; cached {
		return m, nil
	}
	return m, fileDiffCmd(m.sel, entry.dir, entry.file.Path)
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
	n := len(m.diffEntries())
	if m.diffSel >= n {
		m.diffSel = n - 1
	}
	if m.diffSel < 0 {
		m.diffSel = 0
	}
}

func (m *Model) clampDiffScroll() {
	m.diffScroll = clampScroll(m.diffScroll, len(m.diffPanelLines()), m.diffContentHeight())
}

// gridCols is the column count of the horizontal file grid, from the panel
// width. See docs/tui/diff.md.
func (m Model) gridCols() int {
	n := len(m.diffEntries())
	if n == 0 {
		return 1
	}
	return m.diffGridCols(n)
}

// diffGridMove is the h, l, j, and k motion in the horizontal grid. It moves the
// selection by delta files, and keeps it on screen.
func (m Model) diffGridMove(delta int) (tea.Model, tea.Cmd) {
	n := len(m.diffEntries())
	next := m.diffSel + delta
	if n == 0 || next < 0 || next >= n {
		return m, nil
	}
	m.diffSel = next
	m.ensureDiffSelVisible()
	return m, nil
}

// ensureDiffSelVisible scrolls the panel so the selected file row is on screen.
func (m *Model) ensureDiffSelVisible() {
	line := m.diffSelLine()
	height := m.diffContentHeight()
	if line < m.diffScroll {
		m.diffScroll = line
	}
	if line >= m.diffScroll+height {
		m.diffScroll = line - height + 1
	}
	m.clampDiffScroll()
}

// diffSelLine is the line index of the selected file row within the panel lines.
// On a horizontal side, a file sits in the grid, so its line is the header plus
// its grid row.
func (m Model) diffSelLine() int {
	if m.diffHorizontal() {
		return 1 + m.diffSel/m.gridCols()
	}
	return m.diffFileLine(m.diffSel)
}

// diffFileLine is the line index of a file's row within the panel lines. It
// walks the groups the same way the panel draws them, so the header of each
// group and the message of an empty group shift the row down. See docs/tui/diff.md.
func (m Model) diffFileLine(target int) int {
	p := m.diffs[m.sel]
	multi := len(p.groups) > 1
	line := 2
	idx := 0
	for _, g := range p.groups {
		if multi {
			line++
		}
		if !g.repo || len(g.files) == 0 {
			if multi {
				line++
			}
			continue
		}
		for _, file := range g.files {
			if idx == target {
				return line
			}
			line++
			key := fileKey{dir: g.dir, path: file.Path}
			if m.diffOpen[m.sel][key] {
				line += len(m.diffFileBody(key))
			}
			idx++
		}
	}
	return line
}

func (m Model) diffPage() int {
	page := m.diffContentHeight() - 2
	if page < 1 {
		page = 1
	}
	return page
}

func barDiffCount(stat git.Stat) string {
	plus := barAddStyle.Render("+" + strconv.Itoa(stat.Insertions))
	minus := barDelStyle.Render("−" + strconv.Itoa(stat.Deletions))
	return plus + barStyle.Render(" ") + minus
}

const (
	diffSizeStep  = 6
	diffColsMin   = 20
	diffRowsMin   = 4
	diffCellMin   = 24
	diffNumGutter = 5
)

// diffHorizontal reports whether the diff panel is on a horizontal side, so its
// size is rows and its files draw in a grid. See docs/tui/diff.md.
func (m Model) diffHorizontal() bool {
	return config.DiffHorizontal(m.layout.DiffPosition)
}

// diffPanelExtent is the diff panel size along its axis: columns on left or
// right, rows on top or bottom. It reads the half mode, else the size the human
// set live, else the layout size. See docs/tui/diff.md.
func (m Model) diffPanelExtent() int {
	if m.diffHorizontal() {
		if m.diffHalf {
			return m.clampDiffRows((m.bodyHeight() - barHeight) / 2)
		}
		if m.diffSize > 0 {
			return m.clampDiffRows(m.diffSize)
		}
		return m.clampDiffRows(m.layoutDiffSize())
	}
	if m.diffHalf {
		return m.clampDiffCols(m.width / 2)
	}
	if m.diffSize > 0 {
		return m.clampDiffCols(m.diffSize)
	}
	return m.clampDiffCols(m.layoutDiffSize())
}

func (m Model) layoutDiffSize() int {
	if m.layout.DiffSize > 0 {
		return m.layout.DiffSize
	}
	if m.diffHorizontal() {
		return config.DefaultDiffRows
	}
	return config.DefaultDiffSize
}

// diffPanelWidth is the rendered width of the diff panel: the extent on a
// vertical side, else the full output width on a horizontal side.
func (m Model) diffPanelWidth() int {
	if m.diffHorizontal() {
		return m.baseOutputWidth()
	}
	return m.diffPanelExtent()
}

// diffPanelHeight is the rendered height of the diff panel: the body height on a
// vertical side, else the extent on a horizontal side.
func (m Model) diffPanelHeight() int {
	if m.diffHorizontal() {
		return m.diffPanelExtent()
	}
	return m.bodyHeight()
}

// diffContentHeight is the rows the panel body draws inside its footprint. A
// horizontal panel gives one row to the separator border, so its body is one row
// shorter than the footprint. A vertical panel has no such border.
func (m Model) diffContentHeight() int {
	if m.diffHorizontal() {
		if h := m.diffPanelHeight() - 1; h >= 1 {
			return h
		}
		return 1
	}
	return m.diffPanelHeight()
}

func (m Model) clampDiffExtent(size int) int {
	if m.diffHorizontal() {
		return m.clampDiffRows(size)
	}
	return m.clampDiffCols(size)
}

// clampDiffCols keeps the panel wide enough to read, but not so wide that the
// output falls below its minimum.
func (m Model) clampDiffCols(width int) int {
	most := m.baseOutputWidth() - minOutputWithPanel
	if most < diffColsMin {
		most = diffColsMin
	}
	if width < diffColsMin {
		width = diffColsMin
	}
	if width > most {
		width = most
	}
	return width
}

// clampDiffRows keeps the panel tall enough to read, but not so tall that the
// output falls below its minimum.
func (m Model) clampDiffRows(rows int) int {
	most := m.bodyHeight() - barHeight - minOutputHeightWithPanel
	if most < diffRowsMin {
		most = diffRowsMin
	}
	if rows < diffRowsMin {
		rows = diffRowsMin
	}
	if rows > most {
		rows = most
	}
	return rows
}

func (m Model) diffInner() int {
	if m.diffHorizontal() {
		return m.diffPanelWidth()
	}
	return m.diffPanelWidth() - 2
}

// inDiffPanel reports whether a mouse position falls in the open diff panel, by
// its position. The bounds are approximate, so a wheel scroll reaches the panel.
func (m Model) inDiffPanel(x, y int) bool {
	if !m.diffPanel {
		return false
	}
	left := m.leftWidth()
	if m.diffHorizontal() {
		if x < left {
			return false
		}
		h := m.diffPanelHeight()
		if m.layout.DiffPosition == config.DiffTop {
			return y >= barHeight && y < barHeight+h
		}
		return y >= m.bodyHeight()-h && y < m.bodyHeight()
	}
	if m.layout.DiffPosition == config.DiffLeft {
		return x >= left && x < left+m.diffPanelWidth()+2
	}
	return x >= m.width-m.diffPanelWidth()
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
