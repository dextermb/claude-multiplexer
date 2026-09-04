package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const cursorMark = "▌"

const spinInterval = 120 * time.Millisecond

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinnerFrame(n int) string {
	return spinnerFrames[n%len(spinnerFrames)]
}

const (
	sidebarWidth   = 26
	promptHintRows = 1
	promptRowsMin  = 1
	promptRowsMax  = 4
	statusHeight   = 1
	titleHeight    = 0
	barHeight      = 1
	gutterWidth    = 1

	sidebarInner = sidebarWidth - 1 - gutterWidth

	taskPanelWidth     = 32
	taskPanelInner     = taskPanelWidth - 2
	minOutputWithPanel = 40
)

const edgeMark = "▎"

const (
	foldOpenMark = "▾"
	foldShutMark = "▸"
	controlMark  = "⇄"
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
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("240"))

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("62"))

	rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	rowMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	groupMarkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	groupLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))

	groupCountStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	pickedPathStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	statusMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Background(lipgloss.Color("236"))

	statusKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("236"))

	statusCostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("108")).
			Background(lipgloss.Color("236"))

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

	promptLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)

	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	markerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	markerCursorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("62"))

	focusEdgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	fieldLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	questionTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)

	emptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(2, 4)

	selectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))

	barBackground = lipgloss.Color("237")

	barStyle = lipgloss.NewStyle().Background(barBackground)

	barNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(barBackground)

	barMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(barBackground)

	barCostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("108")).
			Background(barBackground)

	taskPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(1)

	taskHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

	taskDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	taskActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	taskPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func classStyle(class render.Class) lipgloss.Style {
	switch class {
	case render.ClassPrompt:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	case render.ClassMeta:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	case render.ClassToolUse:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("74"))
	case render.ClassToolResult:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	case render.ClassThinking:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	case render.ClassStderr:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	case render.ClassError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	case render.ClassBash:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
}

func jobGlyph(status session.JobStatus) string {
	switch status {
	case session.JobDone:
		return "✓"
	case session.JobFailed:
		return "✗"
	case session.JobKilled:
		return "⊗"
	default:
		return "⚙"
	}
}

func jobStyle(status session.JobStatus) lipgloss.Style {
	switch status {
	case session.JobDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case session.JobFailed:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	case session.JobKilled:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	}
}

func stateStyle(state session.State) lipgloss.Style {
	switch state {
	case session.StateIdle:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case session.StateBusy:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	case session.StateWaiting:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	case session.StateStarting:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	case session.StateFailed:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
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
