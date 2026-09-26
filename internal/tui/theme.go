package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The Blackline tokens: light and dark, each as truecolor, xterm-256 and ANSI-16;
// see docs/tui/theme.md.
var (
	colGround     = token("#ffffff", "231", "15", "#000000", "16", "0")
	colSurface    = token("#eeeeee", "255", "7", "#080808", "232", "0")
	colCode       = token("#eeeeee", "255", "7", "#262626", "235", "0")
	colFg         = token("#000000", "16", "0", "#ffffff", "231", "15")
	colHeading    = token("#000000", "16", "0", "#eeeeee", "255", "15")
	colSecondary  = token("#3a3a3a", "237", "8", "#c6c6c6", "251", "7")
	colMuted      = token("#585858", "240", "8", "#8a8a8a", "245", "7")
	colDimmed     = token("#8a8a8a", "245", "8", "#6c6c6c", "242", "8")
	colFaint      = token("#c6c6c6", "251", "7", "#3a3a3a", "237", "8")
	colBorder     = token("#e4e4e4", "254", "7", "#3a3a3a", "237", "8")
	colSubtle     = token("#efefef", "255", "7", "#1d1d1d", "234", "0")
	colStrong     = token("#c6c6c6", "251", "7", "#585858", "240", "8")
	colAccent     = token("#000000", "16", "0", "#ffffff", "231", "15")
	colAccentFg   = token("#ffffff", "231", "15", "#000000", "16", "0")
	colSubdued    = token("#e0e0e0", "254", "7", "#2e2e2e", "236", "8")
	colHighlight  = token("#f5f5f5", "255", "7", "#141414", "233", "0")
	colPositive   = token("#008000", "28", "2", "#4ade80", "78", "10")
	colWarning    = token("#925e00", "94", "3", "#fbbf24", "214", "11")
	colDanger     = token("#b91c1c", "124", "1", "#ff6666", "203", "9")
	colInfo       = token("#1d4ed8", "26", "4", "#60a5fa", "75", "12")
	colPositiveBg = token("#edf6ed", "", "", "#06110a", "", "")
	colWarningBg  = token("#f6f1e9", "", "", "#140f03", "", "")
	colDangerBg   = token("#faefef", "", "", "#140808", "", "")
)

func token(lightHex, light256, light16, darkHex, dark256, dark16 string) lipgloss.CompleteAdaptiveColor {
	return lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: lightHex, ANSI256: light256, ANSI: light16},
		Dark:  lipgloss.CompleteColor{TrueColor: darkHex, ANSI256: dark256, ANSI: dark16},
	}
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
