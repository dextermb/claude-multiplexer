package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Clean drops the escape sequences and the control characters of text that
// comes from outside, so a line cannot move the cursor and paint over another
// pane. See docs/tui/output.md.
func Clean(text string) string {
	if text == "" {
		return text
	}
	if !needsClean(text) {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range ansi.Strip(text) {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
		case r >= 0x80 && r <= 0x9f:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func needsClean(text string) bool {
	for _, r := range text {
		switch {
		case r == '\n' || r == '\t':
		case r < 0x20 || r == 0x7f:
			return true
		case r >= 0x80 && r <= 0x9f:
			return true
		}
	}
	return false
}

func cleanLines(lines []Line) []Line {
	for i := range lines {
		lines[i].Text = Clean(lines[i].Text)
		lines[i].Summary = Clean(lines[i].Summary)
	}
	return lines
}
