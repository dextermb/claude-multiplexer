package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// diffLineTint gives the tint and the mark style of a diff body line: an added
// line and a removed line take a faint tint, and every other line none. See
// docs/tui/theme.md.
func diffLineTint(line string) (color.Color, lipgloss.Style) {
	switch {
	case strings.HasPrefix(line, "+"):
		return colPositiveBg, diffAddStyle
	case strings.HasPrefix(line, "-"):
		return colDangerBg, diffDelStyle
	}
	return nil, rowStyle
}

func tinted(style lipgloss.Style, tint color.Color) lipgloss.Style {
	if tint == nil {
		return style
	}
	return style.Background(tint)
}

// diffChunk draws one wrapped chunk of a body line across width, on the tint of
// its line, in the hue of the line. A non-nil override sets the colour of the
// code, such as the white of the current line.
func diffChunk(chunk string, first bool, tint color.Color, mark lipgloss.Style, override color.Color, width int) string {
	code := mark
	if tint == nil {
		code = fgStyle(colSecondary)
	}
	if override != nil {
		code = fgStyle(override)
	}
	row := ""
	if first && tint != nil && chunk != "" {
		row = tinted(mark, tint).Render(chunk[:1])
		chunk = chunk[1:]
		width--
	}
	return row + tinted(code, tint).Width(maxInt(width, 0)).Render(chunk)
}
