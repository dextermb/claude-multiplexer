package protocol

import "encoding/json"

const ToolTodoWrite = "TodoWrite"

const (
	TodoPending    = "pending"
	TodoInProgress = "in_progress"
	TodoCompleted  = "completed"
)

type Todo struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

type todoWriteInput struct {
	Todos []Todo `json:"todos"`
}

// TodoWrite reads the task list from a TodoWrite tool_use block in an assistant
// message. The session sends the whole list every time, so this list replaces
// the last one; see docs/tui/tasks.md. The second result is false when the
// message holds no such block.
func (e Event) TodoWrite() ([]Todo, bool) {
	if e.Type != TypeAssistant || e.Message == nil {
		return nil, false
	}
	for _, block := range e.Message.Content {
		if block.Type != "tool_use" || block.Name != ToolTodoWrite {
			continue
		}
		var input todoWriteInput
		if err := json.Unmarshal(block.Input, &input); err != nil {
			return nil, false
		}
		return input.Todos, true
	}
	return nil, false
}
