package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func TestColsFollowTheResolvedLayout(t *testing.T) {
	m := Model{width: 200, layout: config.ResolvedLayout{
		SidebarWidth: 40, TaskWidth: 24, DiffWidth: 50, PromptMin: 1, PromptMax: 4,
	}}
	if got := m.sidebarCols(); got != 40 {
		t.Errorf("sidebarCols = %d, want 40", got)
	}
	if got := m.taskCols(); got != 24 {
		t.Errorf("taskCols = %d, want 24", got)
	}
	if got := m.diffPanelWidth(); got != 50 {
		t.Errorf("diffPanelWidth = %d, want 50", got)
	}
}

func TestColsFallBackToDefaultsWithoutALayout(t *testing.T) {
	m := Model{width: 200}
	if got := m.sidebarCols(); got != config.DefaultSidebarWidth {
		t.Errorf("sidebarCols = %d, want default %d", got, config.DefaultSidebarWidth)
	}
	if got := m.taskCols(); got != config.DefaultTaskWidth {
		t.Errorf("taskCols = %d, want default %d", got, config.DefaultTaskWidth)
	}
}

func TestLayoutSwitchStartsOnTheSessionLayoutAndTogglesScope(t *testing.T) {
	d := newLayoutSwitch("api", []string{"a", "b"}, "b", "a")
	if d.allSessions {
		t.Fatal("a session is selected, so the scope must start at this session")
	}
	if d.chosen() != "b" {
		t.Fatalf("the cursor must start on the session layout, chosen = %q", d.chosen())
	}
	d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !d.allSessions {
		t.Fatal("tab must switch the scope to all sessions")
	}
}

func TestLayoutSwitchWithNoSessionUsesAllScope(t *testing.T) {
	d := newLayoutSwitch("", []string{"a"}, "", "a")
	if !d.allSessions {
		t.Fatal("with no session selected, the scope must be all sessions")
	}
}
