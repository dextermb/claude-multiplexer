package tui

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"
)

var mouseArtifactRe = regexp.MustCompile(`^(<[0-9]+;[0-9]+;?[0-9]*[Mm]?)+$`)

// A split read of an SGR mouse sequence leaks its tail as key runes; see docs/tui/keys.md.
func isMouseArtifact(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	if msg.Alt && string(msg.Runes) == "[" {
		return true
	}
	return mouseArtifactRe.MatchString(string(msg.Runes))
}
