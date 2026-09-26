package tui

import (
	"regexp"

	tea "charm.land/bubbletea/v2"
)

var mouseArtifactRe = regexp.MustCompile(`^(<[0-9]+;[0-9]+;?[0-9]*[Mm]?)+$`)

// A split read of an SGR mouse sequence leaks its tail as key runes; see docs/tui/keys.md.
func isMouseArtifact(msg tea.KeyPressMsg) bool {
	if msg.Mod.Contains(tea.ModAlt) && msg.Code == '[' {
		return true
	}
	if msg.Text == "" {
		return false
	}
	return mouseArtifactRe.MatchString(msg.Text)
}
