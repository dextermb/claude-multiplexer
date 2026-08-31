package protocol

import "testing"

const askLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"AskUserQuestion","input":{"questions":[{"question":"Which colour do you prefer?","header":"Colour","options":[{"label":"Red","description":"You prefer red."},{"label":"Blue","description":"You prefer blue."}],"multiSelect":false}]}}]}}`

func TestAskUserQuestionReadsTheBlock(t *testing.T) {
	ev, err := Decode([]byte(askLine))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	id, questions, ok := ev.AskUserQuestion()
	if !ok {
		t.Fatal("AskUserQuestion must find the block")
	}
	if id != "toolu_1" {
		t.Fatalf("id = %q, want toolu_1", id)
	}
	if len(questions) != 1 {
		t.Fatalf("got %d questions, want 1", len(questions))
	}
	q := questions[0]
	if q.Header != "Colour" || q.MultiSelect {
		t.Fatalf("question header/multiSelect wrong: %+v", q)
	}
	if len(q.Options) != 2 || q.Options[0].Label != "Red" || q.Options[1].Label != "Blue" {
		t.Fatalf("options wrong: %+v", q.Options)
	}
}

func TestAskUserQuestionIgnoresOtherMessages(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t","name":"Bash","input":{"command":"ls"}}]}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, _, ok := ev.AskUserQuestion(); ok {
		t.Fatal("a Bash tool_use must not read as a question")
	}
}
