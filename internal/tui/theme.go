package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The Blackline tokens, each as truecolor, xterm-256 and ANSI-16. The interface
// always draws the dark set on a black ground; see docs/tui/theme.md.
var (
	colGround     = token("#000000", "16", "0")
	colSurface    = token("#080808", "232", "0")
	colCode       = token("#262626", "235", "0")
	colFg         = token("#ffffff", "231", "15")
	colHeading    = token("#eeeeee", "255", "15")
	colSecondary  = token("#c6c6c6", "251", "7")
	colMuted      = token("#8a8a8a", "245", "7")
	colDimmed     = token("#6c6c6c", "242", "8")
	colFaint      = token("#3a3a3a", "237", "8")
	colBorder     = token("#3a3a3a", "237", "8")
	colSubtle     = token("#1d1d1d", "234", "0")
	colAccent     = token("#ffffff", "231", "15")
	colAccentFg   = token("#000000", "16", "0")
	colSubdued    = token("#2e2e2e", "236", "8")
	colPositive   = token("#4ade80", "78", "10")
	colWarning    = token("#fbbf24", "214", "11")
	colDanger     = token("#ff6666", "203", "9")
	colInfo       = token("#60a5fa", "75", "12")
	colPositiveBg = token("#06110a", "", "")
	colWarningBg  = token("#140f03", "", "")
	colDangerBg   = token("#140808", "", "")
)

func token(hex, xterm, ansi16 string) lipgloss.CompleteColor {
	return lipgloss.CompleteColor{TrueColor: hex, ANSI256: xterm, ANSI: ansi16}
}

func fgStyle(c lipgloss.TerminalColor) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}

var (
	invertStyle = lipgloss.NewStyle().Foreground(colAccentFg).Background(colAccent)

	subduedStyle = lipgloss.NewStyle().Foreground(colFg).Background(colSubdued)

	labelStyle = fgStyle(colMuted).Transform(strings.ToUpper)

	headingLabelStyle = fgStyle(colHeading).Transform(strings.ToUpper)

	keyStyle = fgStyle(colFg)
)
