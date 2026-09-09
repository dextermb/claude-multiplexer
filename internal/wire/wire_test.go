package wire

import (
	"encoding/json"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestEventRoundTripsThroughJSON(t *testing.T) {
	ev := Event{
		Session: "remote",
		Kind:    1,
		Lines: []render.Line{
			{Class: render.ClassText, Text: "hello"},
			{Class: render.ClassToolUse, Text: "read", Cont: true},
		},
		Partial:    "typing",
		Snapshot:   Snapshot{Name: "remote", State: "busy", Cost: 0.5, Turns: 3, Err: "boom"},
		Todos:      []protocol.Todo{{Content: "do it", Status: protocol.TodoPending}},
		Questions:  []protocol.Question{{Question: "which?", Header: "pick"}},
		QuestionID: "q1",
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Session != "remote" || got.Kind != 1 || got.Partial != "typing" {
		t.Errorf("scalars did not round trip: %+v", got)
	}
	if len(got.Lines) != 2 || got.Lines[0].Text != "hello" || !got.Lines[1].Cont {
		t.Errorf("lines did not round trip: %+v", got.Lines)
	}
	if got.Snapshot.State != "busy" || got.Snapshot.Cost != 0.5 || got.Snapshot.Err != "boom" {
		t.Errorf("snapshot did not round trip: %+v", got.Snapshot)
	}
	if len(got.Todos) != 1 || got.Todos[0].Content != "do it" {
		t.Errorf("todos did not round trip: %+v", got.Todos)
	}
	if len(got.Questions) != 1 || got.Questions[0].Header != "pick" {
		t.Errorf("questions did not round trip: %+v", got.Questions)
	}
}
