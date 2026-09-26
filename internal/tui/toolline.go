package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

const toolNameWidth = 6

var (
	toolArrowStyle = fgStyle(colDimmed)
	toolArgsStyle  = fgStyle(colSecondary)
)

func toolLineView(text string, width int) string {
	rest, ok := strings.CutPrefix(text, "→ ")
	if !ok || rest == "" {
		return classStyle(render.ClassToolUse).Width(width).Render(text)
	}
	name, args, _ := strings.Cut(rest, " ")
	styled := toolArrowStyle.Render("→") + " " + labelStyle.Render(name)
	if args != "" {
		pad := max(toolNameWidth-len([]rune(name)), 0) + 1
		styled += strings.Repeat(" ", pad) + toolArgsStyle.Render(args)
	}
	return lipgloss.NewStyle().Width(width).Render(styled)
}
