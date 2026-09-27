package tui

import (
	"regexp"
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

var sgrCode = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

// clearsBackground reads the parameters of one colour code and reports whether
// it resets the background: a full reset (0, or none), or 49. The numbers inside
// an extended colour, such as 48;2;49;0;0, are skipped.
func clearsBackground(params string) bool {
	if params == "" {
		return true
	}
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "0", "", "49":
			return true
		case "38", "48", "58":
			if i+1 < len(parts) && parts[i+1] == "5" {
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" {
				i += 4
			}
		}
	}
	return false
}

// paintGround gives every cell of the screen the ground colour, so the screen is
// black whatever the background of the terminal is. A code that resets the
// background is followed by the ground again, and each line is padded to the
// full width. See docs/tui/theme.md.
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
		line = sgrCode.ReplaceAllStringFunc(line, func(code string) string {
			if clearsBackground(code[2 : len(code)-1]) {
				return code + ground
			}
			return code
		})
		if pad := width - ansi.StringWidth(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		lines[i] = ground + line + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}
