package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestParseBang(t *testing.T) {
	cases := []struct {
		in      string
		command string
		feed    bool
		ok      bool
	}{
		{`!echo hi`, "echo hi", false, true},
		{`!  ls -la  `, "ls -la", false, true},
		{`!!echo hi`, "echo hi", true, true},
		{`!`, "", false, true},
		{`!!`, "", true, true},
		{`hello`, "", false, false},
		{`/preset`, "", false, false},
	}
	for _, c := range cases {
		command, feed, ok := parseBang(c.in)
		if ok != c.ok || command != c.command || feed != c.feed {
			t.Fatalf("parseBang(%q) = (%q, %v, %v), want (%q, %v, %v)",
				c.in, command, feed, ok, c.command, c.feed, c.ok)
		}
	}
}

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("dispatch returned no command")
	}
	return cmd()
}

func TestBangShowsOutputAndDoesNotReachClaude(t *testing.T) {
	m := promptModel(t)
	next, cmd := m.dispatch(`!echo hello`)
	m = next.(Model)
	msg, ok := runCmd(t, cmd).(bashResultMsg)
	if !ok {
		t.Fatalf("dispatch produced %T, want bashResultMsg", runCmd(t, cmd))
	}
	next, _ = m.handleBash(msg)
	m = next.(Model)

	if len(m.queued[m.sel]) != 0 {
		t.Fatalf("a display-only command must not queue a prompt, got %v", m.queued[m.sel])
	}
	found := false
	for _, line := range m.mgr.Lines(m.sel) {
		if line.Class == render.ClassBash && strings.Contains(line.Text, "hello") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the output is not in the session lines: %v", render.Text(m.mgr.Lines(m.sel)))
	}
	if !strings.Contains(visible(m.View()), "hello") {
		t.Fatalf("the pane does not show the output:\n%s", m.View())
	}
}

func TestBangBangFeedsClaude(t *testing.T) {
	m := promptModel(t)
	next, cmd := m.dispatch(`!!echo hello`)
	m = next.(Model)
	msg := runCmd(t, cmd).(bashResultMsg)
	if !msg.feed {
		t.Fatal("!! must set feed")
	}
	next, _ = m.handleBash(msg)
	m = next.(Model)

	queued := m.queued[m.sel]
	if len(queued) != 1 {
		t.Fatalf("!! must queue one prompt, got %v", queued)
	}
	if !strings.Contains(queued[0], "echo hello") || !strings.Contains(queued[0], "hello") {
		t.Fatalf("the prompt to Claude is missing the command or output: %q", queued[0])
	}
}
