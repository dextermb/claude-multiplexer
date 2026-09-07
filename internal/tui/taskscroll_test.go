package tui

import (
	"fmt"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func manyTodos(n int) []protocol.Todo {
	todos := make([]protocol.Todo, n)
	for i := range todos {
		todos[i] = protocol.Todo{
			Content:    fmt.Sprintf("Task %d", i),
			ActiveForm: fmt.Sprintf("Doing task %d", i),
			Status:     protocol.TodoPending,
		}
	}
	return todos
}

// panelWithTasks starts a session, gives it a long task list, and returns the
// model with the task panel shown.
func panelWithTasks(t *testing.T, n int) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: m.sel, Todos: manyTodos(n)}))
	m, _ = step(t, m, key("esc"))
	if !m.showSidePanel() {
		t.Fatal("the panel must show when the session has a long task list")
	}
	if len(m.taskPanelLines()) <= m.bodyHeight() {
		t.Fatalf("the test needs more task lines (%d) than the body height (%d)", len(m.taskPanelLines()), m.bodyHeight())
	}
	return m
}

func giveTaskFocus(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("k"))
	if m.focus != focusTask {
		t.Fatalf("s k must focus the task panel, focus = %d", m.focus)
	}
	return m
}

func TestTheTaskPanelScrollsAndClampsAtTheEnds(t *testing.T) {
	m := giveTaskFocus(t, panelWithTasks(t, 40))

	m, _ = step(t, m, key("j"))
	if m.taskScroll != 1 {
		t.Fatalf("j must scroll one line, got %d", m.taskScroll)
	}
	m, _ = step(t, m, key("k"))
	if m.taskScroll != 0 {
		t.Fatalf("k must scroll back up, got %d", m.taskScroll)
	}

	m, _ = step(t, m, key("k"))
	if m.taskScroll != 0 {
		t.Fatalf("k must not scroll above the top, got %d", m.taskScroll)
	}

	bottom := len(m.taskPanelLines()) - m.bodyHeight()
	m, _ = step(t, m, key("G"))
	if m.taskScroll != bottom {
		t.Fatalf("G must go to the bottom (%d), got %d", bottom, m.taskScroll)
	}
	m, _ = step(t, m, key("j"))
	if m.taskScroll != bottom {
		t.Fatalf("j must not scroll past the bottom (%d), got %d", bottom, m.taskScroll)
	}

	m, _ = step(t, m, key("g"))
	if m.taskScroll != 0 {
		t.Fatalf("g must go to the top, got %d", m.taskScroll)
	}
}

func TestTabReachesTheTaskPanel(t *testing.T) {
	m := panelWithTasks(t, 40)
	if m.focus != focusSidebar {
		t.Fatalf("the model must start on the sidebar, focus = %d", m.focus)
	}
	m, _ = step(t, m, key("tab")) // sidebar -> prompt
	m, _ = step(t, m, key("tab")) // prompt -> output
	m, _ = step(t, m, key("tab")) // output -> task panel
	if m.focus != focusTask {
		t.Fatalf("Tab must reach the task panel, focus = %d", m.focus)
	}
	m, _ = step(t, m, key("tab")) // task panel -> sidebar
	if m.focus != focusSidebar {
		t.Fatalf("Tab must leave the task panel for the sidebar, focus = %d", m.focus)
	}
}

func TestFocusRetreatsWhenTheTaskPanelHides(t *testing.T) {
	m := giveTaskFocus(t, panelWithTasks(t, 40))
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 2, Session: m.sel, Todos: nil}))
	if m.showSidePanel() {
		t.Fatal("the panel must hide when the list is empty")
	}
	if m.focus != focusOutput {
		t.Fatalf("the focus must retreat to the output, focus = %d", m.focus)
	}
}

func TestFocusingTheTaskPanelIsANoOpWhenItIsHidden(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m, _ = step(t, m, key("esc"))
	if m.showSidePanel() {
		t.Fatal("the panel must be hidden with no tasks or jobs")
	}
	before := m.focus
	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("k"))
	if m.focus != before {
		t.Fatalf("s k must not focus a hidden panel, focus = %d", m.focus)
	}
	if m.focus == focusTask {
		t.Fatal("the focus must not move to a hidden task panel")
	}
}

func TestTheTaskScrollResetsWhenTheSelectionChanges(t *testing.T) {
	m := giveTaskFocus(t, panelWithTasks(t, 40))
	m, _ = step(t, m, key("j"))
	if m.taskScroll == 0 {
		t.Fatal("the panel must scroll before the selection changes")
	}

	m = spawn(t, m, m.mgr, "beta", t.TempDir())
	if m.sel != "beta" {
		t.Fatalf("selected = %q, want beta", m.sel)
	}
	if m.taskScroll != 0 {
		t.Fatalf("the scroll must reset to the top on a new selection, got %d", m.taskScroll)
	}
}
