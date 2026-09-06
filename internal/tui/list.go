package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func (m Model) move(delta int) (tea.Model, tea.Cmd) {
	visible := m.visibleRowIndexes()
	if len(visible) == 0 {
		return m, nil
	}
	index := 0
	for i, at := range visible {
		if m.rows[at].name == m.sel {
			index = i + delta
			break
		}
	}
	if index < 0 {
		index = 0
	}
	if index >= len(visible) {
		index = len(visible) - 1
	}
	name := m.rows[visible[index]].name
	if name == m.sel {
		return m, nil
	}
	m.sel = name
	m.rebuildOutput()
	m.clampOffset()
	return m, nil
}

func (m Model) toggleFold() (tea.Model, tea.Cmd) {
	index := m.selGroup()
	if index < 0 {
		return m, nil
	}
	m.setFold(m.groups[index].key, !m.groups[index].folded)
	return m, nil
}

func (m Model) foldOthers() (tea.Model, tea.Cmd) {
	return m.setAllFolds(false)
}

func (m Model) unfoldAll() (tea.Model, tea.Cmd) {
	return m.setAllFolds(true)
}

// setAllFolds unfolds every group, or folds every group but the one that holds
// the selection.
func (m Model) setAllFolds(unfold bool) (tea.Model, tea.Cmd) {
	if len(m.groups) == 0 {
		return m, nil
	}
	keep := ""
	if index := m.selGroup(); index >= 0 {
		keep = m.groups[index].key
	}
	if m.folded == nil {
		m.folded = make(map[string]bool)
	}
	for _, item := range m.groups {
		if unfold || item.key == keep {
			delete(m.folded, item.key)
			continue
		}
		m.folded[item.key] = true
	}
	m.status = "groups folded"
	if unfold {
		m.status = "groups unfolded"
	}
	m.applyFolds(keep)
	return m, nil
}

func (m *Model) refresh() {
	rows := make([]row, 0, len(m.stored)+4)
	grants := m.mgr.Grants()
	parents := m.mgr.Parents()
	workDirs := m.mgr.WorkingDirs()
	layouts := m.mgr.SessionLayouts()
	for _, snap := range m.mgr.Snapshots() {
		item := rowFromSnapshot(snap)
		item.control = grants[snap.Name]
		item.parent = parents[snap.Name]
		item.workDir = workDirs[snap.Name]
		item.layout = layouts[snap.Name]
		rows = append(rows, item)
	}
	for _, meta := range m.stored {
		if meta.Archived && !m.showArchived {
			continue
		}
		rows = append(rows, rowFromMeta(meta))
	}
	children := make(map[string]bool, len(rows))
	for _, item := range rows {
		if item.parent != "" {
			children[item.parent] = true
		}
	}
	for i := range rows {
		rows[i].group = m.rowGroup(rows[i], children)
	}
	m.rows, m.groups = groupRows(rows, m.folded)
	m.buildLines()
	m.syncJobsModal()

	if m.selLine() < 0 {
		m.selectFirst()
	}
	if len(m.rows) == 0 {
		m.sel = ""
	}
	m.clampOffset()
	m.applyLayout()
}

// syncJobsModal gives the open jobs dialog the jobs of its session, so a
// running job grows while you read it. See docs/tui/sessions.md.
func (m *Model) syncJobsModal() {
	if m.jobsModal == nil {
		return
	}
	item, ok := m.selectedRow()
	if !ok {
		return
	}
	m.jobsModal.setJobs(item.jobList)
}

// rowGroup keys a row on the control session that created it, or that it
// created rows for, and on its repository when neither holds.
func (m *Model) rowGroup(item row, children map[string]bool) string {
	if item.parent != "" {
		return byPrefix + item.parent
	}
	if children[item.name] {
		return byPrefix + item.name
	}
	return dirPrefix + m.groupKey(item.dir)
}

// groupKey caches the walk to the repository, because refresh runs on every
// event and a session directory never changes.
func (m *Model) groupKey(dir string) string {
	if dir == "" {
		return ""
	}
	if root, ok := m.roots[dir]; ok {
		return root
	}
	root := repoRoot(dir)
	if m.roots == nil {
		m.roots = make(map[string]string)
	}
	m.roots[dir] = root
	return root
}

func (m *Model) buildLines() {
	m.lines = listLines(m.rows, m.groups)
}

// selLine is the line the selected session is drawn on, or -1 when a fold hides
// it or nothing is selected.
func (m Model) selLine() int {
	for i, line := range m.lines {
		if !line.header() && m.rows[line.row].name == m.sel {
			return i
		}
	}
	return -1
}

func (m Model) selGroup() int {
	for _, item := range m.rows {
		if item.name != m.sel {
			continue
		}
		for i := range m.groups {
			if m.groups[i].key == item.group {
				return i
			}
		}
	}
	return -1
}

func (m *Model) selectFirst() {
	for _, line := range m.lines {
		if !line.header() {
			m.sel = m.rows[line.row].name
			return
		}
	}
	if len(m.rows) == 0 {
		return
	}
	m.setFold(m.rows[0].group, false)
	m.sel = m.rows[0].name
}

// selectNear takes the selection to the row after a group that just folded, or
// to the row before it when the group is the last one.
func (m *Model) selectNear(key string) {
	at := -1
	for i, line := range m.lines {
		if line.header() && m.groups[line.group].key == key {
			at = i
			break
		}
	}
	if at < 0 {
		m.selectFirst()
		return
	}
	for i := at + 1; i < len(m.lines); i++ {
		if !m.lines[i].header() {
			m.sel = m.rows[m.lines[i].row].name
			return
		}
	}
	for i := at - 1; i >= 0; i-- {
		if !m.lines[i].header() {
			m.sel = m.rows[m.lines[i].row].name
			return
		}
	}
}

func (m *Model) setFold(key string, folded bool) {
	if m.folded == nil {
		m.folded = make(map[string]bool)
	}
	if folded {
		m.folded[key] = true
	} else {
		delete(m.folded, key)
	}
	m.applyFolds(key)
}

func (m *Model) applyFolds(key string) {
	for i := range m.groups {
		m.groups[i].folded = m.folded[m.groups[i].key]
	}
	m.buildLines()
	previous := m.sel
	if m.selLine() < 0 {
		m.selectNear(key)
	}
	if m.sel != previous {
		m.rebuildOutput()
	}
	m.clampOffset()
}

// revealSelection unfolds the group of the selected session, so a session that
// starts in a folded group is visible.
func (m *Model) revealSelection() {
	for _, item := range m.rows {
		if item.name == m.sel && m.folded[item.group] {
			m.setFold(item.group, false)
			return
		}
	}
	m.clampOffset()
}

func (m *Model) clampOffset() {
	visible := m.visibleLines()
	index := m.selLine()
	if index >= 0 && visible > 0 {
		if index < m.listOffset {
			m.listOffset = index
		}
		if index >= m.listOffset+visible {
			m.listOffset = index - visible + 1
		}
	}
	if last := len(m.lines) - visible; m.listOffset > last {
		m.listOffset = last
	}
	if m.listOffset < 0 {
		m.listOffset = 0
	}
}

func (m Model) selIndex() int {
	for i, item := range m.rows {
		if item.name == m.sel {
			return i
		}
	}
	return -1
}

func (m Model) selectedRow() (row, bool) {
	index := m.selIndex()
	if index < 0 {
		return row{}, false
	}
	return m.rows[index], true
}

func (m Model) selectedJobs() []session.Job {
	if item, ok := m.selectedRow(); ok {
		return item.jobList
	}
	return nil
}

func (m Model) thinkingSelected() bool {
	if m.sel == "" || m.partials[m.sel] != "" {
		return false
	}
	if len(m.queued[m.sel]) > 0 {
		return true
	}
	return m.selectedBusy()
}

func (m Model) selectedBusy() bool {
	item, ok := m.selectedRow()
	return ok && item.state == session.StateBusy
}
