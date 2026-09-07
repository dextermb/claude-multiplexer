package tui

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) diffPanelView() string {
	lines := m.diffPanelLines()
	height := m.diffPanelHeight()
	scroll := clampScroll(m.diffScroll, len(lines), height)
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	block := strings.Join(lines[scroll:end], "\n")
	return sidePanelStyle(m.focus == focusDiff).Width(m.diffPanelWidth() - 1).Height(height).Render(block)
}

func (m Model) diffPanelLines() []string {
	p, ok := m.diffs[m.sel]
	if !ok || len(p.groups) == 0 {
		return []string{diffMetaStyle.Render("reading…")}
	}
	if len(p.groups) == 1 && !p.groups[0].repo {
		return []string{diffMetaStyle.Render("not a git repository")}
	}
	if m.diffHorizontal() {
		return m.diffGridLines(p)
	}
	out := []string{taskHeaderStyle.Render("Changes · " + strconv.Itoa(p.totalFiles())), ""}
	multi := len(p.groups) > 1
	if !multi && p.totalFiles() == 0 {
		return append(out, diffMetaStyle.Render("no changes"))
	}
	idx := 0
	for _, g := range p.groups {
		if multi {
			out = append(out, m.diffGroupHeader(g))
		}
		if !g.repo {
			out = append(out, diffMetaStyle.Render("  not a git repository"))
			continue
		}
		if len(g.files) == 0 {
			out = append(out, diffMetaStyle.Render("  no changes"))
			continue
		}
		for _, file := range g.files {
			entry := diffEntry{dir: g.dir, file: file}
			out = append(out, m.diffFileRow(idx, entry))
			if m.diffOpen[m.sel][entry.key()] {
				out = append(out, m.diffFileBodyAt(entry.key(), m.diffScroll-len(out))...)
			}
			idx++
		}
	}
	return out
}

// diffGroupHeader is the header line of one directory of a project: its short
// name and its change count. See docs/tui/diff.md.
func (m Model) diffGroupHeader(g dirDiff) string {
	width := m.diffInner()
	left := taskHeaderStyle.Render(truncate(shortDir(g.dir), width-12))
	counts := diffAddStyle.Render("+"+strconv.Itoa(g.stat.Insertions)) + " " +
		diffDelStyle.Render("−"+strconv.Itoa(g.stat.Deletions))
	gap := width - lipgloss.Width(left) - lipgloss.Width(counts)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + counts
}

// diffGridLines draws every file of a horizontal panel in one grid, then draws
// each open file's diff below the grid. It tags a cell with its directory when
// the project has more than one. See docs/tui/diff.md.
func (m Model) diffGridLines(p projectDiff) []string {
	entries := p.entries()
	out := []string{taskHeaderStyle.Render("Changes · " + strconv.Itoa(len(entries)))}
	if len(entries) == 0 {
		return append(out, "", diffMetaStyle.Render("no changes"))
	}
	multi := len(p.groups) > 1
	cols := m.diffGridCols(len(entries))
	cellW := m.diffInner() / cols
	rows := (len(entries) + cols - 1) / cols
	for r := 0; r < rows; r++ {
		cells := make([]string, 0, cols)
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if i >= len(entries) {
				cells = append(cells, strings.Repeat(" ", cellW))
				continue
			}
			cells = append(cells, m.diffFileCell(i, entries[i], cellW, multi))
		}
		out = append(out, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	for _, e := range entries {
		if !m.diffOpen[m.sel][e.key()] {
			continue
		}
		out = append(out, "", taskHeaderStyle.Render(truncate(entryLabel(e, multi), m.diffInner())))
		out = append(out, m.diffFileBody(e.key())...)
	}
	return out
}

// diffGridCols is the column count of the horizontal file grid: as many cells of
// at least diffCellMin columns as the panel width holds, capped at the file
// count.
func (m Model) diffGridCols(n int) int {
	cols := m.diffInner() / diffCellMin
	if cols < 1 {
		cols = 1
	}
	if cols > n {
		cols = n
	}
	return cols
}

func (m Model) diffFileRow(index int, entry diffEntry) string {
	return m.diffFileCell(index, entry, m.diffInner(), false)
}

// shortDir is the display name of a directory in a group header: its base name.
func shortDir(dir string) string {
	if dir == "" {
		return "(directory)"
	}
	return filepath.Base(dir)
}

// entryLabel is the file name the panel shows for an entry. It prefixes the
// directory's short name when the project has more than one directory, so a file
// name that repeats across directories stays distinct.
func entryLabel(e diffEntry, showDir bool) string {
	if showDir {
		return shortDir(e.dir) + "/" + e.file.Path
	}
	return e.file.Path
}

func (m Model) diffFileCell(index int, entry diffEntry, width int, showDir bool) string {
	file := entry.file
	glyph := foldShutMark
	if m.diffOpen[m.sel][entry.key()] {
		glyph = foldOpenMark
	}
	head := glyph + " " + file.Status + " "
	countsText := "+" + strconv.Itoa(file.Insertions) + " −" + strconv.Itoa(file.Deletions)
	room := width - lipgloss.Width(head) - lipgloss.Width(countsText) - 1
	if room < 4 {
		room = 4
	}
	name := truncate(entryLabel(entry, showDir), room)

	if index == m.diffSel {
		text := head + name
		gap := width - lipgloss.Width(text) - lipgloss.Width(countsText)
		if gap < 1 {
			gap = 1
		}
		return selectedRowStyle.Width(width).Render(text + strings.Repeat(" ", gap) + countsText)
	}

	counts := diffAddStyle.Render("+"+strconv.Itoa(file.Insertions)) + " " +
		diffDelStyle.Render("−"+strconv.Itoa(file.Deletions))
	left := diffMetaStyle.Render(head) + rowStyle.Render(name)
	gap := width - lipgloss.Width(left) - lipgloss.Width(counts)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + counts
}

func (m Model) diffFileBody(key fileKey) []string {
	return m.diffFileBodyAt(key, -1)
}

// diffFileBodyAt renders a file's diff, and marks output line current as the
// current line. A negative current marks nothing.
func (m Model) diffFileBodyAt(key fileKey, current int) []string {
	text, ok := m.fileDiffs[m.sel][key]
	if !ok {
		return []string{diffMetaStyle.Render("  reading…")}
	}
	return m.renderDiffBody(text, current)
}

// renderDiffBody colours a raw git diff and wraps it to the panel width. It
// drops the git file header, and shows new-file line numbers when they are on.
// It marks the current line: it bolds the line number, or the text when the
// numbers are off. See docs/tui/diff.md.
func (m Model) renderDiffBody(text string, current int) []string {
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
			marked := len(out) == current
			lineStyle := style
			if marked && !m.diffLineNumbers {
				lineStyle = lineStyle.Bold(true)
			}
			row := lineStyle.Render(chunk)
			if m.diffLineNumbers {
				gutter := ""
				if i == 0 {
					gutter = number
				}
				numStyle := diffNumStyle
				if marked {
					numStyle = diffCurNumStyle
				}
				row = numStyle.Render(padLeft(gutter, diffNumGutter-1)+" ") + row
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
