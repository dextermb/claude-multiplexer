package protocol

import (
	"strings"
	"testing"
)

func TestWriterSendsOnlyTheFieldsTheAPIAccepts(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)
	if err := w.SendUser("", "hello"); err != nil {
		t.Fatalf("SendUser: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	want := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello"}]},"parent_tool_use_id":null}`
	if got != want {
		t.Fatalf("line =\n%s\nwant\n%s", got, want)
	}
}

func TestWriterIncludesTheSessionIDWhenKnown(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)
	if err := w.SendUser("abc", "hello"); err != nil {
		t.Fatalf("SendUser: %v", err)
	}
	if !strings.Contains(buf.String(), `"session_id":"abc"`) {
		t.Fatalf("line = %s", buf.String())
	}
}

func TestWriterSetsTheModelWithAControlRequest(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)
	if err := w.SetModel("model-1", "sonnet"); err != nil {
		t.Fatalf("SetModel: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	want := `{"type":"control_request","request_id":"model-1","request":{"subtype":"set_model","model":"sonnet"}}`
	if got != want {
		t.Fatalf("line =\n%s\nwant\n%s", got, want)
	}
}

func TestWriterSetsThePermissionModeWithAControlRequest(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)
	if err := w.SetPermissionMode("mode-1", "plan"); err != nil {
		t.Fatalf("SetPermissionMode: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	want := `{"type":"control_request","request_id":"mode-1","request":{"subtype":"set_permission_mode","mode":"plan"}}`
	if got != want {
		t.Fatalf("line =\n%s\nwant\n%s", got, want)
	}
}

func TestWriterEndsEveryLineWithANewline(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)
	_ = w.SendUser("", "one")
	_ = w.SendUser("", "two")
	if lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n"); len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatal("the last line has no newline")
	}
}
