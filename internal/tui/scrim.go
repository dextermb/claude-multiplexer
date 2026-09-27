package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var scrimStyle = fgStyle(colScrim)

// scrim is a faint copy of a block: the same text in one dark grey, so it stays
// in place under a dialog but no longer reads as live.
func scrim(block string) string {
	lines := strings.Split(ansi.Strip(block), "\n")
	for i, line := range lines {
		lines[i] = scrimStyle.Render(line)
	}
	return strings.Join(lines, "\n")
}

// overlay lays a dialog over a faint copy of the region under it, centred in a
// region of width by height, with the Lip Gloss compositor. See docs/tui.md.
func overlay(under, dialog string, width, height int) string {
	return overlayIn(under, dialog, width, height, 0, height)
}

// overlayIn fades the whole of under, and centres the dialog in the rows from
// top to top+rows, so a dialog can sit in the body while the frame fades.
func overlayIn(under, dialog string, width, height, top, rows int) string {
	x := maxInt(0, (width-lipgloss.Width(dialog))/2)
	y := top + maxInt(0, (rows-lipgloss.Height(dialog))/2)
	base := lipgloss.NewStyle().Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(scrim(under))
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(dialog).X(x).Y(y).Z(1),
	).Render()
}
