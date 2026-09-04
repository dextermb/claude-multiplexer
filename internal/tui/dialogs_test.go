package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func withJobs(m Model) Model {
	for i := range m.rows {
		m.rows[i].jobList = []session.Job{
			{ID: "j1", Description: "build the binary", Status: session.JobRunning},
			{ID: "j2", Description: "run the tests", Status: session.JobDone},
		}
	}
	return m
}

func jobsOpen(t *testing.T, width, height int) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, width, height)
	m.form = nil
	m = spawn(t, m, mgr, "api", t.TempDir())
	m = withJobs(m)
	m.focus = focusSidebar
	m, _ = step(t, m, key("J"))
	if m.jobsModal == nil {
		t.Fatal("J must open the jobs dialog")
	}
	return m
}

func TestSessionDialogKeepsTheSidebar(t *testing.T) {
	m := jobsOpen(t, 100, 24)
	view := visible(m.View())
	if !strings.Contains(view, "Background jobs") {
		t.Fatalf("the jobs dialog must draw:\n%s", view)
	}
	if !strings.Contains(view, foldOpenMark) {
		t.Fatalf("the sidebar must stay beside the dialog:\n%s", view)
	}
	if !strings.Contains(view, "starting") {
		t.Fatalf("the bar must stay above the dialog:\n%s", view)
	}
}

func TestSessionDialogLeavesTheSidebarColumns(t *testing.T) {
	m := jobsOpen(t, 100, 24)
	for _, line := range strings.Split(visible(m.View()), "\n") {
		if strings.Contains(line, "Background jobs") {
			if column := strings.Index(line, "Background jobs"); column < sidebarWidth {
				t.Fatalf("the dialog starts at column %d, inside the sidebar:\n%s", column, line)
			}
		}
	}
}

func TestBodyDialogCoversTheSidebar(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form = nil
	m = spawn(t, m, mgr, "api", t.TempDir())
	m.focus = focusSidebar
	m, _ = step(t, m, key("?"))
	if m.help == nil {
		t.Fatal("? must open the key list")
	}
	view := visible(m.View())
	if strings.Contains(view, foldOpenMark) {
		t.Fatalf("the key list belongs to no session, so it covers the sidebar:\n%s", view)
	}
}

func TestSessionDialogFitsANarrowTerminal(t *testing.T) {
	const width = 60
	m := jobsOpen(t, width, 20)
	for i, line := range strings.Split(visible(m.View()), "\n") {
		if got := len([]rune(line)); got > width {
			t.Fatalf("line %d is %d columns wide, the terminal is %d:\n%s", i, got, width, line)
		}
	}
}

func TestModalInnerFitsItsRegion(t *testing.T) {
	cases := []struct {
		width int
		want  int
	}{
		{width: 100, want: 92},
		{width: 48, want: 40},
		{width: 30, want: 28},
		{width: 10, want: 8},
		{width: 1, want: 1},
	}
	for _, c := range cases {
		if got := modalInner(c.width); got != c.want {
			t.Fatalf("modalInner(%d) = %d, want %d", c.width, got, c.want)
		}
	}
}
