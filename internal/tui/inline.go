package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

type spanKind uint8

const (
	spanPlain spanKind = iota
	spanItalic
	spanBold
	spanCode
)

type inlineSpan struct {
	text string
	kind spanKind
}

// inlineEmphasis renders a prompt line with inline markdown emphasis on top of
// the base style. See docs/markdown.md.
func inlineEmphasis(text string, base lipgloss.Style) string {
	spans := parseInline(text)
	if len(spans) == 1 && spans[0].kind == spanPlain {
		return base.Render(text)
	}
	var b strings.Builder
	for _, sp := range spans {
		switch sp.kind {
		case spanItalic:
			b.WriteString(base.Italic(true).Render(sp.text))
		case spanBold:
			b.WriteString(base.Bold(true).Render(sp.text))
		case spanCode:
			b.WriteString(base.Bold(false).
				Foreground(lipgloss.Color("219")).
				Background(lipgloss.Color("237")).
				Render(" " + sp.text + " "))
		default:
			b.WriteString(base.Render(sp.text))
		}
	}
	return b.String()
}

func parseInline(s string) []inlineSpan {
	runes := []rune(s)
	n := len(runes)
	var spans []inlineSpan
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			spans = append(spans, inlineSpan{plain.String(), spanPlain})
			plain.Reset()
		}
	}
	i := 0
	for i < n {
		c := runes[i]
		switch c {
		case '`':
			if j := findCode(runes, i+1); j > 0 {
				flush()
				spans = append(spans, inlineSpan{string(runes[i+1 : j]), spanCode})
				i = j + 1
				continue
			}
		case '*', '_':
			l := 1
			if i+1 < n && runes[i+1] == c {
				l = 2
			}
			if canOpen(runes, i, c, l) {
				if j := findClose(runes, i+l, c, l); j > 0 {
					flush()
					kind := spanItalic
					if l == 2 {
						kind = spanBold
					}
					spans = append(spans, inlineSpan{string(runes[i+l : j]), kind})
					i = j + l
					continue
				}
			}
		}
		plain.WriteRune(c)
		i++
	}
	flush()
	return spans
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func canOpen(runes []rune, i int, c rune, l int) bool {
	after := i + l
	if after >= len(runes) || unicode.IsSpace(runes[after]) {
		return false
	}
	if c == '_' && i > 0 && isWordRune(runes[i-1]) {
		return false
	}
	return true
}

// findClose returns the start index of a closing delimiter run for an emphasis
// span opened before start, or -1 when the span does not close.
func findClose(runes []rune, start int, c rune, l int) int {
	n := len(runes)
	for j := start; j < n; j++ {
		if runes[j] != c {
			continue
		}
		k := j
		for k < n && runes[k] == c {
			k++
		}
		if k-j < l {
			j = k - 1
			continue
		}
		p := k - l
		if p <= start || unicode.IsSpace(runes[p-1]) {
			j = k - 1
			continue
		}
		if c == '_' && k < n && isWordRune(runes[k]) {
			j = k - 1
			continue
		}
		return p
	}
	return -1
}

func findCode(runes []rune, start int) int {
	for j := start; j < len(runes); j++ {
		if runes[j] == '`' {
			if j > start {
				return j
			}
			return -1
		}
	}
	return -1
}
