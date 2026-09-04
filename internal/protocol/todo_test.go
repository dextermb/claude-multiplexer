package protocol

import "testing"

const todoLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"TodoWrite","input":{"todos":[{"content":"Add the panel","status":"completed","activeForm":"Adding the panel"},{"content":"Wire the manager","status":"in_progress","activeForm":"Wiring the manager"},{"content":"Write the tests","status":"pending","activeForm":"Writing the tests"}]}}]}}`

func TestTodoWriteReadsTheList(t *testing.T) {
	ev, err := Decode([]byte(todoLine))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	todos, ok := ev.TodoWrite()
	if !ok {
		t.Fatal("TodoWrite must find the block")
	}
	if len(todos) != 3 {
		t.Fatalf("got %d todos, want 3", len(todos))
	}
	if todos[1].Content != "Wire the manager" || todos[1].Status != TodoInProgress {
		t.Fatalf("second todo wrong: %+v", todos[1])
	}
	if todos[1].ActiveForm != "Wiring the manager" {
		t.Fatalf("active form wrong: %+v", todos[1])
	}
}

func TestTodoWriteReadsAnEmptyList(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t","name":"TodoWrite","input":{"todos":[]}}]}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	todos, ok := ev.TodoWrite()
	if !ok {
		t.Fatal("an empty list still reads as a TodoWrite, so the panel can clear")
	}
	if len(todos) != 0 {
		t.Fatalf("got %d todos, want 0", len(todos))
	}
}

func TestTodoWriteIgnoresOtherMessages(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t","name":"Bash","input":{"command":"ls"}}]}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, ok := ev.TodoWrite(); ok {
		t.Fatal("a Bash tool_use must not read as a task list")
	}
}
