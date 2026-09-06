package tui

import (
	"os"
	"path/filepath"
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
	name, isDefault := d.choice()
	if isDefault || name != "b" {
		t.Fatalf("the cursor must start on the session layout, got %q default=%v", name, isDefault)
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

func TestReloadLayoutsReadsTheFileFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{"activeLayout":"wide","layouts":{"wide":{"sidebarWidth":40}}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Model{opts: Options{ConfigPaths: []string{path}}}
	m.reloadLayouts()
	if m.activeLayout != "wide" {
		t.Fatalf("activeLayout = %q, want wide", m.activeLayout)
	}
	if _, ok := m.layouts["wide"]; !ok {
		t.Fatalf("layouts = %+v, want a wide layout read from disk", m.layouts)
	}
}

func TestLayoutSwitchDefaultRowIsFirstAndClears(t *testing.T) {
	d := newLayoutSwitch("api", []string{"a", "b"}, "", "")
	if name, isDefault := d.choice(); !isDefault || name != "" {
		t.Fatalf("with no session layout the cursor must start on default, got %q default=%v", name, isDefault)
	}
	d.Update(tea.KeyMsg{Type: tea.KeyDown})
	if name, isDefault := d.choice(); isDefault || name != "a" {
		t.Fatalf("down from default must land on the first layout, got %q default=%v", name, isDefault)
	}
}
