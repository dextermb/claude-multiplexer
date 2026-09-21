package mcp

import (
	"encoding/json"
	"testing"
)

func TestBarDefaultsReturnsBothBars(t *testing.T) {
	out, err := barDefaults()
	if err != nil {
		t.Fatalf("barDefaults: %v", err)
	}
	var session struct {
		Left  []any `json:"left"`
		Right []any `json:"right"`
	}
	if err := json.Unmarshal(out.Session, &session); err != nil {
		t.Fatalf("the session default is not valid JSON: %v", err)
	}
	if len(session.Left) == 0 || len(session.Right) == 0 {
		t.Fatalf("the session default has empty sides: %+v", session)
	}
	if len(out.Status) == 0 {
		t.Fatal("the status default is empty")
	}
	if out.Note == "" {
		t.Fatal("the note is empty, want set_config guidance")
	}
}
