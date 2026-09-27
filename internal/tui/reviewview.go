package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

const reviewMinExplain = 24

// reviewHeader draws a pane header as a label set in a rule, inverted when the
// pane holds the focus, so the split shows which side takes the keys. See
// docs/tui/review.md.
func reviewHeader(title string, width int, focused bool) string {
	return ruleLabel(title, width, focused)
}

// reviewView draws the review screen: the split of the diff and the explanation
// thread. The band names the session. See docs/tui/review.md.
func (m Model) reviewView() string {
	return m.reviewSplit()
}

func (m Model) reviewHeight() int {
	if h := m.bodyHeight(); h > 1 {
		return h
	}
	return 1
}

// reviewDiffWidth and reviewExplainWidth divide the pane between the diff and
// the thread, with one column for the rule between them.
func (m Model) reviewExplainWidth() int {
	total := m.baseOutputWidth()
	w := total * 2 / 5
	if w < reviewMinExplain {
		w = reviewMinExplain
	}
	if w > total-21 {
		w = total - 21
	}
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) reviewDiffWidth() int {
	w := m.baseOutputWidth() - m.reviewExplainWidth() - 1
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) reviewSplit() string {
	diffW, height := m.reviewDiffWidth(), m.reviewHeight()
	diffLines, _ := m.reviewDiffContent(diffW)
	left := reviewBlock(diffLines, m.reviewScroll, diffW, height)
	right := m.reviewExplainPane()
	sep := reviewRule(height, isRule(strings.SplitN(left, "\n", 2)[0]))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

// reviewExplainPane is the explanation side: a header, then the session output
// pane, so the reply carries the block cursor and its capped blocks. See
// docs/tui/review.md.
func (m Model) reviewExplainPane() string {
	head := reviewHeader("explanation", m.reviewExplainWidth(), m.reviewFocus == reviewExplain)
	body := lipgloss.NewStyle().PaddingLeft(1).Render(m.output.View())
	return lipgloss.JoinVertical(lipgloss.Left, head, body)
}

func reviewBlock(lines []string, scroll, width, height int) string {
	scroll = clampScroll(scroll, len(lines), height)
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	window := lines[scroll:end]
	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(strings.Join(window, "\n"))
}

// reviewRule is the rule between the diff and the explanation. It joins the
// header rules with a junction when the diff header is on the screen.
func reviewRule(height int, headed bool) string {
	col := make([]string, height)
	for i := range col {
		col[i] = ruleStyle.Render("│")
	}
	if headed && height > 0 {
		col[0] = ruleStyle.Render("┬")
	}
	return strings.Join(col, "\n")
}

// reviewDiffContent builds the diff side: a header, the file list, and the
// selected file expanded to its hunks. It returns the line index of the
// selected hunk, so the caller can scroll it into view. See docs/tui/review.md.
func (m Model) reviewDiffContent(width int) (lines []string, selLine int) {
	selLine = -1
	p, ok := m.diffs[m.sel]
	if !ok || len(p.groups) == 0 {
		return []string{diffMetaStyle.Render("reading…")}, -1
	}
	if len(p.groups) == 1 && !p.groups[0].repo {
		return []string{diffMetaStyle.Render("not a git repository")}, -1
	}
	entries := m.diffEntries()
	stat := p.stat()
	title := "diff · " + m.sel + " (" + plural(len(entries), "file") + " · +" +
		strconv.Itoa(stat.Insertions) + " −" + strconv.Itoa(stat.Deletions) + ")"
	lines = append(lines, reviewHeader(title, width, m.reviewFocus == reviewDiff), "")
	if len(entries) == 0 {
		return append(lines, diffMetaStyle.Render("no changes")), -1
	}
	for i, e := range entries {
		lines = append(lines, m.reviewFileRow(i, e, width))
		if i != m.reviewFile {
			continue
		}
		text, cached := m.fileDiffs[m.sel][e.key()]
		if !cached {
			lines = append(lines, diffMetaStyle.Render("  reading…"))
			continue
		}
		hunks := git.Hunks(text)
		if len(hunks) == 0 {
			lines = append(lines, diffMetaStyle.Render("  (no text change)"))
			continue
		}
		for hi, h := range hunks {
			if hi == m.reviewHunk {
				selLine = len(lines)
			}
			lines = append(lines, m.reviewHunkLines(h, width, hi == m.reviewHunk)...)
		}
	}
	return lines, selLine
}

func (m Model) reviewFileRow(index int, entry diffEntry, width int) string {
	file := entry.file
	head := file.Status + " "
	counts := "+" + strconv.Itoa(file.Insertions) + " −" + strconv.Itoa(file.Deletions)
	room := width - lipgloss.Width(head) - lipgloss.Width(counts) - 1
	if room < 4 {
		room = 4
	}
	name := truncate(entryLabel(entry, m.multiGroup()), room)
	if index == m.reviewFile {
		text := head + name
		gap := width - lipgloss.Width(text) - lipgloss.Width(counts)
		if gap < 1 {
			gap = 1
		}
		return selectedRowStyle.Width(width).Render(text + strings.Repeat(" ", gap) + counts)
	}
	left := diffMetaStyle.Render(head) + rowStyle.Render(name)
	gap := width - lipgloss.Width(left) - lipgloss.Width(counts)
	if gap < 1 {
		gap = 1
	}
	rightCounts := diffAddStyle.Render("+"+strconv.Itoa(file.Insertions)) + " " +
		diffDelStyle.Render("−"+strconv.Itoa(file.Deletions))
	return left + strings.Repeat(" ", gap) + rightCounts
}

// reviewHunkLines colours a hunk and wraps it to the diff width. An added row
// and a removed row take a faint tint across the row, and only their mark takes
// the hue. The selected hunk carries a white edge mark on each row, so the tints
// still show under it. With the line numbers on, a gutter holds the new-side
// number, and a removed line leaves it blank. See docs/tui/review.md.
func (m Model) reviewHunkLines(h git.Hunk, width int, marked bool) []string {
	content := width - 1
	if m.reviewLineNumbers {
		content -= diffNumGutter
	}
	content = maxInt(content, 1)
	edge := " "
	if marked {
		edge = "▌"
	}
	var out []string
	if marked {
		out = append(out, invertStyle.Width(width).MaxWidth(width).Render(truncate(edge+h.Header, width)))
	} else {
		out = append(out, diffHunkStyle.Width(width).Render(edge+truncate(h.Header, width-1)))
	}
	newLine := h.NewStart
	for _, line := range h.Body {
		number := ""
		if !strings.HasPrefix(line, "-") {
			number = strconv.Itoa(newLine)
			newLine++
		}
		tint, mark := diffLineTint(line)
		for i, chunk := range wrapHard(line, content) {
			row := tinted(keyStyle, tint).Render(edge)
			if m.reviewLineNumbers {
				gutter := ""
				if i == 0 {
					gutter = number
				}
				row += tinted(diffNumStyle, tint).Render(padLeft(gutter, diffNumGutter-1) + " ")
			}
			out = append(out, row+diffChunk(chunk, i == 0, tint, mark, colSecondary, content))
		}
	}
	return out
}
