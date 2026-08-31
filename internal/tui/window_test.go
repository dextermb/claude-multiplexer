package tui

import "testing"

func TestContextWindow(t *testing.T) {
	cases := []struct {
		model string
		want  int
	}{
		{"claude-opus-4-8", 200_000},
		{"claude-sonnet-4-6[1m]", 1_000_000},
		{"gpt-4o", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := contextWindow(c.model); got != c.want {
			t.Errorf("contextWindow(%q) = %d, want %d", c.model, got, c.want)
		}
	}
}
