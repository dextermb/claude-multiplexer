package mcp

import (
	"testing"
)

func TestBarDefaultsReturnsBothBars(t *testing.T) {
	out, err := barDefaults()
	if err != nil {
		t.Fatalf("barDefaults: %v", err)
	}
	if len(out.Session.Left) == 0 || len(out.Session.Right) == 0 {
		t.Fatalf("the session default has empty sides: %+v", out.Session)
	}
	if len(out.Status.Left) == 0 || len(out.Status.Right) == 0 {
		t.Fatalf("the status default has empty sides: %+v", out.Status)
	}
	if out.Note == "" {
		t.Fatal("the note is empty, want set_config guidance")
	}
}
