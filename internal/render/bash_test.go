package render

import (
	"strings"
	"testing"
)

func TestBashLinesHeaderAndBody(t *testing.T) {
	got := BashLines("echo hi", "hi\nthere\n", nil)
	if len(got) != 3 {
		t.Fatalf("lines = %v, want 3", Text(got))
	}
	if got[0].Class != ClassBash || got[0].Text != "! echo hi" {
		t.Fatalf("header = %+v", got[0])
	}
	if got[1].Text != "hi" || got[2].Text != "there" {
		t.Fatalf("body = %v", Text(got[1:]))
	}
	for _, line := range got {
		if line.Class != ClassBash {
			t.Fatalf("every line must be bash, got %v in %v", line.Class, Text(got))
		}
	}
}

func TestBashLinesNoOutput(t *testing.T) {
	got := BashLines("true", "", nil)
	if len(got) != 1 || got[0].Text != "! true" {
		t.Fatalf("lines = %v, want only the header", Text(got))
	}
}

func TestBashLinesAddsAnErrorLine(t *testing.T) {
	got := BashLines("false", "", errTest)
	last := got[len(got)-1]
	if last.Class != ClassError || !strings.Contains(last.Text, "boom") {
		t.Fatalf("last line = %+v, want an error line", last)
	}
}
