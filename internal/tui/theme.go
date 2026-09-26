package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// The Blackline tokens. Bubble Tea converts each one to the colours the
// terminal shows, and maps every grey to its exact xterm grey; see
// docs/tui/theme.md.
var (
	colGround     = lipgloss.Color("#000000")
	colSurface    = lipgloss.Color("#080808")
	colCode       = lipgloss.Color("#262626")
	colFg         = lipgloss.Color("#ffffff")
	colHeading    = lipgloss.Color("#eeeeee")
	colSecondary  = lipgloss.Color("#c6c6c6")
	colMuted      = lipgloss.Color("#8a8a8a")
	colDimmed     = lipgloss.Color("#6c6c6c")
	colFaint      = lipgloss.Color("#3a3a3a")
	colBorder     = lipgloss.Color("#3a3a3a")
	colSubtle     = lipgloss.Color("#1d1d1d")
	colStrong     = lipgloss.Color("#585858")
	colAccent     = lipgloss.Color("#ffffff")
	colAccentFg   = lipgloss.Color("#000000")
	colSubdued    = lipgloss.Color("#2e2e2e")
	colPositive   = lipgloss.Color("#4ade80")
	colWarning    = lipgloss.Color("#fbbf24")
	colDanger     = lipgloss.Color("#ff6666")
	colInfo       = lipgloss.Color("#60a5fa")
	colPositiveBg = lipgloss.Color("#06110a")
	colWarningBg  = lipgloss.Color("#140f03")
	colDangerBg   = lipgloss.Color("#140808")
)

func fgStyle(c color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}

var (
	invertStyle = lipgloss.NewStyle().Foreground(colAccentFg).Background(colAccent)

	subduedStyle = lipgloss.NewStyle().Foreground(colFg).Background(colSubdued)

	labelStyle = fgStyle(colMuted).Transform(strings.ToUpper)

	headingLabelStyle = fgStyle(colHeading).Transform(strings.ToUpper)

	keyStyle = fgStyle(colFg)
)
