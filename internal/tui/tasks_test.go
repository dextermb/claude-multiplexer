package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

var sampleTodos = []protocol.Todo{
	{Content: "First task", Status: protocol.TodoCompleted, ActiveForm: "Doing the first task"},
	{Content: "Second task", Status: protocol.TodoInProgress, ActiveForm: "Doing the second task"},
	{Content: "Third task", Status: protocol.TodoPending, ActiveForm: "Doing the third task"},
}

func TestTheTaskPanelShowsTheListAndShrinksTheOutput(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	base := m.outputWidth()
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: m.sel, Todos: sampleTodos}))

	if !m.showTaskPanel() {
		t.Fatal("the panel must show when the selected session has tasks")
	}
	if m.outputWidth() != base-taskPanelWidth {
		t.Fatalf("output width = %d, want %d (shrunk by the panel)", m.outputWidth(), base-taskPanelWidth)
	}
	view := visible(m.View())
	for _, want := range []string{"Tasks · 1/3", "First task", "Doing the second task", "Third task"} {
		if !strings.Contains(view, want) {
			t.Errorf("the panel has no %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Second task") {
		t.Error("an in-progress task must show its active form, not its content")
	}
}

func TestTheTaskPanelHidesWhenTheListIsEmpty(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	base := m.outputWidth()
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: m.sel, Todos: sampleTodos}))
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 2, Session: m.sel, Todos: nil}))

	if m.showTaskPanel() {
		t.Fatal("the panel must hide when the list is empty")
	}
	if m.outputWidth() != base {
		t.Fatalf("output width = %d, want the full %d again", m.outputWidth(), base)
	}
	if strings.Contains(visible(m.View()), "Tasks ·") {
		t.Error("the panel must leave no header behind")
	}
}

func TestTheTaskPanelHidesOnANarrowTerminal(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 70, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	base := m.outputWidth()
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: m.sel, Todos: sampleTodos}))

	if m.showTaskPanel() {
		t.Fatal("a narrow terminal has no room for the panel")
	}
	if m.outputWidth() != base {
		t.Fatalf("output width = %d, want the full %d", m.outputWidth(), base)
	}
}

func TestTheViewFillsTheWindowWithTheTaskPanel(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 110, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: m.sel, Todos: sampleTodos}))

	if !m.showTaskPanel() {
		t.Fatal("the panel must show at this width")
	}
	lines := strings.Split(m.View(), "\n")
	if len(lines) != m.height {
		t.Fatalf("the view has %d lines, want %d", len(lines), m.height)
	}
	for i := 0; i < m.bodyHeight(); i++ {
		if width := lipgloss.Width(lines[i]); width != m.width {
			t.Errorf("body line %d is %d wide, want exactly %d:\n%s", i, width, m.width, lines[i])
		}
	}
}

func TestATaskEventForAnotherSessionDoesNotShowThePanel(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	if m.sel != "beta" {
		t.Fatalf("selected = %q, want beta", m.sel)
	}
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: "alpha", Todos: sampleTodos}))

	if m.showTaskPanel() {
		t.Fatal("the panel follows the selected session, not another one")
	}
}
