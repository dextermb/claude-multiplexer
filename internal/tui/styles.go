package tui

import (
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const cursorMark = "▌"

const spinInterval = 120 * time.Millisecond

// These name the built-in layout dimensions, so a Model with no layout draws the
// same as before. A layout overrides them; see docs/tui.md and docs/config.md.
const (
	sidebarWidth     = config.DefaultSidebarSize
	promptHintRows   = 1
	promptBorderRows = 1
	promptRowsMin    = config.DefaultPromptMin
	promptRowsMax    = config.DefaultPromptMax
	statusHeight     = 1
	titleHeight      = 1
	bandHeight       = 1
	barHeight        = 1
	gutterWidth      = 1

	taskPanelWidth           = config.DefaultTaskSize
	minOutputWithPanel       = 40
	minOutputHeightWithPanel = 6
)

const (
	foldOpenMark = "▾"
	foldShutMark = "▸"
	// The single-letter session flags, shown muted to the right of a session name
	// in the sidebar. They concatenate, so a hoisted control session reads "HC".
	// See docs/tui/sessions.md.
	controlMark  = "C"
	scheduleMark = "S"
	hoistMark    = "H"
	// readOnlyMark flags a spectator session: it streams a peer's session
	// read-only, so the interface takes no input for it. See docs/tui/sessions.md.
	readOnlyMark = "R"
	// watchedMark flags a session a spectator watches now, through a share this
	// host minted, so the host sees it is shared. See docs/tui/sessions.md.
	watchedMark = "W"
	// heldMark flags a session the context governor holds, so it takes no prompt
	// until the human clears the hold. See docs/sessions/context.md.
	heldMark = "!"
)

// modalInner caps a dialog at width-2, because a wider box pushes the sidebar
// beside it out of line.
func modalInner(width int) int {
	inner := width - 8
	if inner < 40 {
		inner = 40
	}
	if inner > width-2 {
		inner = width - 2
	}
	if inner < 1 {
		inner = 1
	}
	return inner
}

var (
	titleStyle = headingLabelStyle.Padding(0, 1)

	selectedRowStyle = invertStyle

	rowStyle = fgStyle(colSecondary)

	rowMutedStyle = fgStyle(colDimmed)

	groupMarkStyle = fgStyle(colDimmed)

	groupLabelStyle = labelStyle

	groupMutedStyle = fgStyle(colDimmed).Transform(strings.ToUpper)

	groupCountStyle = fgStyle(colDimmed)

	sectionLabelStyle = fgStyle(colDimmed).Transform(strings.ToUpper)

	sectionRuleStyle = fgStyle(colSubtle)

	sectionParentStyle = labelStyle

	hintStyle = fgStyle(colMuted)

	searchStyle = lipgloss.NewStyle().
			Foreground(colFg).
			Background(colSurface)

	pickedPathStyle = invertStyle

	statusBackground color.Color = colSurface

	statusStyle = lipgloss.NewStyle().
			Foreground(colSecondary).
			Background(statusBackground).
			Padding(0, 1)

	statusMutedStyle = fgStyle(colMuted).Background(statusBackground)

	statusKeyStyle = fgStyle(colFg).Background(statusBackground)

	statusCostStyle = fgStyle(colSecondary).Background(statusBackground)

	errorStyle = fgStyle(colDanger)

	updateBannerStyle = lipgloss.NewStyle().
				Foreground(colWarning).
				Background(colWarningBg).
				Padding(0, 1)

	spinnerStyle = fgStyle(colDimmed)

	ageStyle = fgStyle(colDimmed)

	markerStyle = fgStyle(colMuted)

	markerCursorStyle = invertStyle

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colBorder).
			Padding(1, 2)

	fieldLabelStyle = labelStyle

	selectArrowStyle = fgStyle(colDimmed)

	selectValueStyle = fgStyle(colFg)

	questionTextStyle = fgStyle(colHeading)

	emptyStyle = fgStyle(colMuted).Padding(2, 4)

	selectionStyle = invertStyle

	barStyle = lipgloss.NewStyle()

	barNameStyle = fgStyle(colHeading)

	barMutedStyle = fgStyle(colMuted)

	barCostStyle = fgStyle(colSecondary)

	barPRStyle = fgStyle(colWarning)

	taskHeaderStyle = labelStyle

	taskDoneStyle    = fgStyle(colDimmed)
	taskActiveStyle  = fgStyle(colFg)
	taskPendingStyle = fgStyle(colMuted)

	barAddStyle = fgStyle(colPositive)
	barDelStyle = fgStyle(colDanger)

	diffAddStyle    = fgStyle(colPositive)
	diffDelStyle    = fgStyle(colDanger)
	diffHunkStyle   = fgStyle(colDimmed)
	diffMetaStyle   = fgStyle(colMuted)
	diffNumStyle    = fgStyle(colDimmed)
	diffCurNumStyle = fgStyle(colFg)
)

func classStyle(class render.Class) lipgloss.Style {
	switch class {
	case render.ClassPrompt:
		return fgStyle(colHeading)
	case render.ClassMeta, render.ClassThinking, render.ClassJob, render.ClassResult:
		return fgStyle(colDimmed)
	case render.ClassToolUse, render.ClassToolResult, render.ClassSkill:
		return fgStyle(colMuted)
	case render.ClassStderr:
		return fgStyle(colWarning)
	case render.ClassError:
		return fgStyle(colDanger)
	}
	return fgStyle(colSecondary)
}

func jobStyle(status session.JobStatus) lipgloss.Style {
	switch status {
	case session.JobDone:
		return fgStyle(colPositive)
	case session.JobFailed:
		return fgStyle(colDanger)
	case session.JobKilled:
		return fgStyle(colDimmed)
	default:
		return fgStyle(colWarning)
	}
}

func stateStyle(state session.State) lipgloss.Style {
	switch state {
	case session.StateIdle:
		return fgStyle(colPositive)
	case session.StateBusy:
		return fgStyle(colWarning)
	case session.StateWaiting:
		return fgStyle(colInfo)
	case session.StateFailed:
		return fgStyle(colDanger)
	}
	return fgStyle(colDimmed)
}

func truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		lines = append(lines, wrapParagraph(para, width)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func wrapParagraph(para string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(para) {
		for lipgloss.Width(word) > width {
			head := string([]rune(word)[:width])
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			lines = append(lines, head)
			word = string([]rune(word)[width:])
		}
		switch {
		case line == "":
			line = word
		case lipgloss.Width(line)+1+lipgloss.Width(word) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func pad(text string, width int) string {
	runes := []rune(text)
	if len(runes) >= width {
		return truncate(text, width)
	}
	out := make([]rune, width)
	copy(out, runes)
	for i := len(runes); i < width; i++ {
		out[i] = ' '
	}
	return string(out)
}
