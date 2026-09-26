package tui

import (
	"strings"
	"testing"
)

func TestTheOverlayCentresTheDialogOverAFaintCopy(t *testing.T) {
	under := strings.Repeat(fgStyle(colMuted).Render("abcdefghij")+"\n", 5)
	under = strings.TrimSuffix(under, "\n")
	out := overlay(under, "XY", 10, 5)

	lines := strings.Split(visible(out), "\n")
	if len(lines) != 5 {
		t.Fatalf("the overlay is %d lines, want the region height 5", len(lines))
	}
	if lines[2] != "abcdXYghij" {
		t.Fatalf("the dialog must sit in the centre, got %q", lines[2])
	}
	if lines[0] != "abcdefghij" {
		t.Fatalf("the region must stay under the dialog, got %q", lines[0])
	}
	if !strings.Contains(out, "38;2;38;38;38m") || strings.Contains(out, "38;2;138;138;138m") {
		t.Fatalf("the region under the dialog must be the faint copy only: %q", out)
	}
}
