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

func TestABodyDialogFadesTheWholeScreen(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 120, 30)
	if m.form == nil {
		t.Fatal("the new session form must open when the list is empty")
	}
	lines := strings.Split(m.screen(), "\n")
	for _, want := range []string{"─ PROMPT", "MULTIPLEXER"} {
		for _, line := range lines {
			if !strings.Contains(visible(line), want) {
				continue
			}
			if strings.Contains(line, "38;2;138;138;138m") || strings.Contains(line, "38;2;238;238;238m") {
				t.Errorf("the %s row must fade under the dialog: %q", want, line)
			}
		}
	}
	if !strings.Contains(visible(m.screen()), "NEW SESSION") {
		t.Fatal("the dialog must still draw")
	}
}

func TestOverlayInCentresInItsRegion(t *testing.T) {
	under := strings.TrimSuffix(strings.Repeat("abcdefghij\n", 7), "\n")
	lines := strings.Split(visible(overlayIn(under, "XY", 10, 7, 1, 3)), "\n")
	if len(lines) != 7 || lines[2] != "abcdXYghij" {
		t.Fatalf("the dialog must sit in the centre of rows 1 to 3, got %q", lines)
	}
}

func TestASessionDialogFadesOnlyItsSessionPanes(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 140, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.confirm = "alpha"

	var sidebar, prompt string
	for _, line := range strings.Split(m.screen(), "\n") {
		plain := visible(line)
		if strings.Contains(plain, "SESSIONS (") {
			sidebar = line
		}
		if strings.Contains(plain, "─ PROMPT") {
			prompt = line
		}
	}
	for name, line := range map[string]string{"sidebar": sidebar, "prompt": prompt} {
		if line == "" || strings.Contains(line, "38;2;38;38;38m") {
			t.Errorf("the %s row must stay bright under a session dialog: %q", name, line)
		}
	}
	pane := strings.Split(m.paneView(), "\n")
	if len(pane) <= barHeight || !strings.Contains(strings.Join(pane[barHeight:], "\n"), "38;2;38;38;38m") {
		t.Fatal("the session output must fade under a session dialog")
	}
}
