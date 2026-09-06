package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

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
			out = append(out, m.diffFileBodyAt(file.Path, m.diffScroll-len(out))...)
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
	return m.diffFileBodyAt(path, -1)
}

// diffFileBodyAt renders a file's diff, and marks output line current as the
// current line. A negative current marks nothing.
func (m Model) diffFileBodyAt(path string, current int) []string {
	text, ok := m.fileDiffs[m.sel][path]
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
