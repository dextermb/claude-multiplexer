package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// groundSGR is the escape code that sets the ground colour as the background, in
// the colour profile of the terminal. It is empty when the terminal has no
// colour.
func groundSGR() string {
	cell := lipgloss.NewStyle().Background(colGround).Render(" ")
	at := strings.Index(cell, " ")
	if at <= 0 {
		return ""
	}
	return cell[:at]
}

// paintGround gives every cell of the screen the ground colour, so the screen is
// black whatever the background of the terminal is. A reset clears the
// background with the other attributes, so the ground follows each reset, and
// each line is padded to the full width. See docs/tui/theme.md.
func paintGround(view string, width, height int) string {
	ground := groundSGR()
	if ground == "" {
		return view
	}
	lines := strings.Split(view, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, line := range lines {
		for _, reset := range []string{"\x1b[0m", "\x1b[m", "\x1b[49m"} {
			line = strings.ReplaceAll(line, reset, reset+ground)
		}
		if pad := width - ansi.StringWidth(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		lines[i] = ground + line + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}
