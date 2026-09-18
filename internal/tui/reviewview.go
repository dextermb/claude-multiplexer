package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

const reviewMinExplain = 24

// reviewSelBg is the subtle band behind the selected hunk on the diff side.
var reviewSelBg = lipgloss.Color("237")

// reviewView draws the review screen: the session bar, then the split of the
// diff and the explanation thread. See docs/tui/review.md.
func (m Model) reviewView() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.barViewWidth(m.baseOutputWidth()), m.reviewSplit())
}

func (m Model) reviewHeight() int {
	if h := m.bodyHeight() - barHeight; h > 1 {
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
	sep := reviewRule(height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

// reviewExplainPane is the explanation side: a header, then the session output
// pane, so the reply carries the block cursor and its capped blocks. See
// docs/tui/review.md.
func (m Model) reviewExplainPane() string {
	width := m.reviewExplainWidth()
	head := " " + taskHeaderStyle.Render("Explanation")
	if m.reviewFocus != reviewDiff {
		head = focusEdgeStyle.Render("▌") + taskHeaderStyle.Render("Explanation")
	}
	head = lipgloss.NewStyle().Width(width).Render(head)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.output.View())
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

func reviewRule(height int) string {
	col := make([]string, height)
	for i := range col {
		col[i] = diffMetaStyle.Render("│")
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
	label := "Review · " + m.sel
	if m.reviewFocus == reviewDiff {
		label = focusEdgeStyle.Render("▌") + taskHeaderStyle.Render(label)
	} else {
		label = " " + taskHeaderStyle.Render(label)
	}
	stat := p.stat()
	counts := diffAddStyle.Render("+"+strconv.Itoa(stat.Insertions)) + " " +
		diffDelStyle.Render("−"+strconv.Itoa(stat.Deletions)) + "  " +
		diffMetaStyle.Render(plural(len(entries), "file"))
	lines = append(lines, label, counts, "")
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

func (m Model) reviewHunkLines(h git.Hunk, width int, marked bool) []string {
	render := func(style lipgloss.Style, text string) []string {
		var rows []string
		for _, chunk := range wrapHard(text, width) {
			if marked {
				style = style.Background(reviewSelBg).Width(width)
			}
			rows = append(rows, style.Render(chunk))
		}
		return rows
	}
	headerStyle := diffHunkStyle
	if marked {
		headerStyle = headerStyle.Bold(true)
	}
	out := render(headerStyle, h.Header)
	for _, line := range h.Body {
		style := rowStyle
		switch {
		case strings.HasPrefix(line, "+"):
			style = diffAddStyle
		case strings.HasPrefix(line, "-"):
			style = diffDelStyle
		}
		out = append(out, render(style, line)...)
	}
	return out
}
