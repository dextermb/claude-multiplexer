package protocol

import (
	"encoding/json"
	"strconv"
)

const (
	ToolTaskCreate = "TaskCreate"
	ToolTaskUpdate = "TaskUpdate"
)

type taskCreateInput struct {
	Subject    string `json:"subject"`
	ActiveForm string `json:"activeForm"`
}

type taskUpdateInput struct {
	TaskID string `json:"taskId"`
	Status string `json:"status"`
}

// TaskEdit is one status change to a task, keyed by the id the session gave the
// task when it created it.
type TaskEdit struct {
	ID     string
	Status string
}

// TaskCreate reads the tasks that TaskCreate tool_use blocks add, in block
// order. A new task starts pending; see docs/tui/tasks.md.
func (e Event) TaskCreate() []Todo {
	if e.Type != TypeAssistant || e.Message == nil {
		return nil
	}
	var out []Todo
	for _, block := range e.Message.Content {
		if block.Type != "tool_use" || block.Name != ToolTaskCreate {
			continue
		}
		var input taskCreateInput
		if err := json.Unmarshal(block.Input, &input); err != nil {
			continue
		}
		out = append(out, Todo{
			Content:    input.Subject,
			ActiveForm: input.ActiveForm,
			Status:     TodoPending,
		})
	}
	return out
}

// TaskUpdate reads the status changes that TaskUpdate tool_use blocks make, in
// block order; see docs/tui/tasks.md.
func (e Event) TaskUpdate() []TaskEdit {
	if e.Type != TypeAssistant || e.Message == nil {
		return nil
	}
	var out []TaskEdit
	for _, block := range e.Message.Content {
		if block.Type != "tool_use" || block.Name != ToolTaskUpdate {
			continue
		}
		var input taskUpdateInput
		if err := json.Unmarshal(block.Input, &input); err != nil {
			continue
		}
		out = append(out, TaskEdit{ID: input.TaskID, Status: input.Status})
	}
	return out
}

// TaskTracker accumulates a session's task list over the stream. The list is
// incremental: TaskCreate adds a task and TaskUpdate changes one. The older
// TodoWrite tool replaces the whole list instead. See docs/tui/tasks.md.
type TaskTracker struct {
	order []string
	byID  map[string]*Todo
}

// Apply folds one event into the list.
func (t *TaskTracker) Apply(e Event) {
	if list, ok := e.TodoWrite(); ok {
		t.replace(list)
		return
	}
	for _, todo := range e.TaskCreate() {
		t.add(todo)
	}
	for _, edit := range e.TaskUpdate() {
		if todo := t.byID[edit.ID]; todo != nil {
			todo.Status = edit.Status
		}
	}
}

// List returns the tasks in creation order, or nil when the list is empty.
func (t *TaskTracker) List() []Todo {
	if len(t.order) == 0 {
		return nil
	}
	out := make([]Todo, 0, len(t.order))
	for _, id := range t.order {
		out = append(out, *t.byID[id])
	}
	return out
}

// add appends a task and gives it the next id in creation order, so the id
// matches the number the session assigns.
func (t *TaskTracker) add(todo Todo) {
	if t.byID == nil {
		t.byID = make(map[string]*Todo)
	}
	id := strconv.Itoa(len(t.order) + 1)
	stored := todo
	t.byID[id] = &stored
	t.order = append(t.order, id)
}

func (t *TaskTracker) replace(list []Todo) {
	t.order = nil
	t.byID = nil
	for _, todo := range list {
		t.add(todo)
	}
}
