package render

import (
	"testing"
	"time"
)

func TestStampAtMarksOnlyBlockStarts(t *testing.T) {
	at := time.Unix(1_700_000_000, 0)
	lines := stampAt([]Line{
		{Class: ClassPrompt, Text: "› one"},
		{Class: ClassPrompt, Text: "  two", Cont: true},
		{Class: ClassBash, Text: "! ls"},
	}, at)

	if !lines[0].At.Equal(at) {
		t.Errorf("block-start line At = %v, want %v", lines[0].At, at)
	}
	if !lines[1].At.IsZero() {
		t.Errorf("continuation line carries %v, want zero", lines[1].At)
	}
	if !lines[2].At.Equal(at) {
		t.Errorf("second block-start line At = %v, want %v", lines[2].At, at)
	}
}
