package protocol

import "testing"

const (
	taskCreateLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"c1","name":"TaskCreate","input":{"subject":"Add the panel","activeForm":"Adding the panel"}},{"type":"tool_use","id":"c2","name":"TaskCreate","input":{"subject":"Wire the manager","activeForm":"Wiring the manager"}}]}}`
	taskUpdateLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"u1","name":"TaskUpdate","input":{"taskId":"1","status":"completed"}},{"type":"tool_use","id":"u2","name":"TaskUpdate","input":{"taskId":"2","status":"in_progress"}}]}}`
)

func decode(t *testing.T, line string) Event {
	t.Helper()
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return ev
}

func TestTaskCreateReadsTheBlocks(t *testing.T) {
	todos := decode(t, taskCreateLine).TaskCreate()
	if len(todos) != 2 {
		t.Fatalf("want 2 created tasks, got %d", len(todos))
	}
	if todos[0].Content != "Add the panel" || todos[0].ActiveForm != "Adding the panel" {
		t.Errorf("first task wrong: %+v", todos[0])
	}
	if todos[0].Status != TodoPending {
		t.Errorf("a new task must start pending, got %q", todos[0].Status)
	}
}

func TestTaskUpdateReadsTheBlocks(t *testing.T) {
	edits := decode(t, taskUpdateLine).TaskUpdate()
	if len(edits) != 2 {
		t.Fatalf("want 2 edits, got %d", len(edits))
	}
	if edits[0] != (TaskEdit{ID: "1", Status: TodoCompleted}) {
		t.Errorf("first edit wrong: %+v", edits[0])
	}
	if edits[1] != (TaskEdit{ID: "2", Status: TodoInProgress}) {
		t.Errorf("second edit wrong: %+v", edits[1])
	}
}

func TestTaskCreateIgnoresOtherMessages(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}`
	if todos := decode(t, line).TaskCreate(); todos != nil {
		t.Errorf("a text message holds no tasks, got %+v", todos)
	}
	if edits := decode(t, line).TaskUpdate(); edits != nil {
		t.Errorf("a text message holds no edits, got %+v", edits)
	}
}

func TestTaskTrackerAccumulatesCreatesThenUpdates(t *testing.T) {
	var tasks TaskTracker
	tasks.Apply(decode(t, taskCreateLine))
	tasks.Apply(decode(t, taskUpdateLine))

	list := tasks.List()
	if len(list) != 2 {
		t.Fatalf("want 2 tasks, got %d: %+v", len(list), list)
	}
	if list[0].Content != "Add the panel" || list[0].Status != TodoCompleted {
		t.Errorf("first task wrong after update: %+v", list[0])
	}
	if list[1].Content != "Wire the manager" || list[1].Status != TodoInProgress {
		t.Errorf("second task wrong after update: %+v", list[1])
	}
}

func TestTaskTrackerKeepsCreationOrderAcrossMessages(t *testing.T) {
	create := func(subject string) Event {
		line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"c","name":"TaskCreate","input":{"subject":"` + subject + `"}}]}}`
		return decode(t, line)
	}
	var tasks TaskTracker
	tasks.Apply(create("one"))
	tasks.Apply(create("two"))
	tasks.Apply(create("three"))

	update := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"u","name":"TaskUpdate","input":{"taskId":"3","status":"in_progress"}}]}}`
	tasks.Apply(decode(t, update))

	list := tasks.List()
	if len(list) != 3 || list[2].Content != "three" || list[2].Status != TodoInProgress {
		t.Fatalf("the third task must take the edit: %+v", list)
	}
	if list[0].Status != TodoPending || list[1].Status != TodoPending {
		t.Errorf("the other tasks must stay pending: %+v", list)
	}
}

func TestTaskTrackerStillReadsLegacyTodoWrite(t *testing.T) {
	var tasks TaskTracker
	tasks.Apply(decode(t, todoLine))

	list := tasks.List()
	if len(list) != 3 {
		t.Fatalf("want 3 tasks from TodoWrite, got %d", len(list))
	}
	if list[1].Content != "Wire the manager" || list[1].Status != TodoInProgress {
		t.Errorf("legacy task wrong: %+v", list[1])
	}
}

func TestTaskTrackerReplacesOnLegacyTodoWrite(t *testing.T) {
	var tasks TaskTracker
	tasks.Apply(decode(t, taskCreateLine))
	empty := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t","name":"TodoWrite","input":{"todos":[]}}]}}`
	tasks.Apply(decode(t, empty))

	if list := tasks.List(); list != nil {
		t.Errorf("an empty TodoWrite must clear the list, got %+v", list)
	}
}
