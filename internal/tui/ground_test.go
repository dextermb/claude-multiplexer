package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestThePaintedGroundCoversEveryCell(t *testing.T) {
	ground := "\x1b[48;2;0;0;0m"
	if got := groundSGR(); got != ground {
		t.Fatalf("ground = %q, want %q", got, ground)
	}

	view := labelStyle.Render("sessions") + " plain\nshort"
	lines := strings.Split(paintGround(view, 20, 3), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want the full height of 3", len(lines))
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, ground) {
			t.Errorf("line %d does not start on the ground: %q", i, line)
		}
		if w := ansi.StringWidth(line); w != 20 {
			t.Errorf("line %d is %d wide, want the full width of 20: %q", i, w, line)
		}
	}
	if !strings.Contains(lines[0], "\x1b[m"+ground+" plain") {
		t.Errorf("the ground must return after a reset: %q", lines[0])
	}
}

func TestTheViewIsPaintedOnTheGround(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 30)
	lines := strings.Split(m.screen(), "\n")
	if len(lines) != 30 {
		t.Fatalf("the view is %d lines, want 30", len(lines))
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "\x1b[48;2;0;0;0m") {
			t.Fatalf("line %d does not start on the ground: %q", i, line)
		}
	}
}
