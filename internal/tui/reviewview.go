package tui

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

const reviewMinExplain = 24

// reviewSelBg is the subtle band behind the selected hunk on the diff side.
var reviewSelBg color.Color = colSubdued

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

// reviewHunkLines colours a hunk and wraps it to the diff width. When the line
// numbers are on it prefixes each row with a gutter of the new-side line number,
// the same as the diff panel. A removed line has no new-side number, so its
// gutter is blank. See docs/tui/review.md.
func (m Model) reviewHunkLines(h git.Hunk, width int, marked bool) []string {
	content := width
	if m.reviewLineNumbers {
		content -= diffNumGutter
	}
	if content < 1 {
		content = 1
	}
	var out []string
	render := func(style lipgloss.Style, text, number string) {
		for i, chunk := range wrapHard(text, content) {
			if marked {
				style = style.Background(reviewSelBg).Width(content)
			}
			row := style.Render(chunk)
			if m.reviewLineNumbers {
				gutter := ""
				if i == 0 {
					gutter = number
				}
				numStyle := diffNumStyle
				if marked {
					numStyle = diffCurNumStyle.Background(reviewSelBg)
				}
				row = numStyle.Render(padLeft(gutter, diffNumGutter-1)+" ") + row
			}
			out = append(out, row)
		}
	}
	headerStyle := diffHunkStyle
	if marked {
		headerStyle = headerStyle.Foreground(colFg)
	}
	render(headerStyle, h.Header, "")
	newLine := h.NewStart
	for _, line := range h.Body {
		style := rowStyle
		number := ""
		switch {
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
		render(style, line, number)
	}
	return out
}
