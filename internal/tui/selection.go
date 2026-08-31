package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
)

type pos struct {
	line int
	col  int
}

func (p pos) before(o pos) bool {
	return p.line < o.line || (p.line == o.line && p.col < o.col)
}

type selRange struct {
	active   bool
	dragging bool
	anchor   pos
	cursor   pos
}

func (s selRange) empty() bool {
	return s.anchor == s.cursor
}

func (s selRange) bounds() (pos, pos) {
	if s.cursor.before(s.anchor) {
		return s.cursor, s.anchor
	}
	return s.anchor, s.cursor
}

var copyToClipboard = clipboard.WriteAll

func selectedText(lines []string, a, b pos) string {
	if len(lines) == 0 {
		return ""
	}
	if b.before(a) {
		a, b = b, a
	}
	last := len(lines) - 1
	clamp := func(p pos) pos {
		if p.line < 0 {
			return pos{0, 0}
		}
		if p.line > last {
			return pos{last, len([]rune(lines[last]))}
		}
		return p
	}
	a, b = clamp(a), clamp(b)

	slice := func(line string, from, to int) string {
		runes := []rune(line)
		if from < 0 {
			from = 0
		}
		if to < 0 || to > len(runes) {
			to = len(runes)
		}
		if from > len(runes) {
			from = len(runes)
		}
		if from > to {
			from = to
		}
		return string(runes[from:to])
	}

	if a.line == b.line {
		return slice(lines[a.line], a.col, b.col)
	}
	out := make([]string, 0, b.line-a.line+1)
	out = append(out, slice(lines[a.line], a.col, -1))
	for i := a.line + 1; i < b.line; i++ {
		out = append(out, lines[i])
	}
	out = append(out, slice(lines[b.line], 0, b.col))
	return strings.Join(out, "\n")
}

func plainLines(content string) []string {
	return strings.Split(ansi.Strip(content), "\n")
}
