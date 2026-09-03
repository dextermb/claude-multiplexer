package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIsMouseArtifact(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{"a full sgr tail", key("<0;1;3M"), true},
		{"a release tail", key("<0;1;3m"), true},
		{"two tails at once", key("<0;1;3M<0;1;4M"), true},
		{"a tail split before the end", key("<0;1"), true},
		{"a mangled tail", key("<0;12;3M"), true},
		{"the alt bracket fragment", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("["), Alt: true}, true},
		{"a plain bracket", key("["), false},
		{"a less-than", key("<"), false},
		{"a comparison", key("<5"), false},
		{"a capital m", key("M"), false},
		{"ordinary text", key("hello"), false},
		{"a non-rune key", tea.KeyMsg{Type: tea.KeyEnter}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMouseArtifact(tc.msg); got != tc.want {
				t.Fatalf("isMouseArtifact(%q) = %v, want %v", string(tc.msg.Runes), got, tc.want)
			}
		})
	}
}

func TestScrollArtifactsNeverReachThePrompt(t *testing.T) {
	m := promptModel(t)

	m, _ = step(t, m, key("hi"))
	m, _ = step(t, m, key("<0;1;3M"))
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("["), Alt: true})

	if got := m.prompt.Value(); got != "hi" {
		t.Fatalf("the prompt holds %q, want %q", got, "hi")
	}
}
