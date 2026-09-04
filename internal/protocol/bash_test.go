package protocol

import "testing"

func TestBackgroundBashReadsTheCommand(t *testing.T) {
	block := Block{
		Type:  "tool_use",
		Name:  "Bash",
		Input: []byte(`{"command":"sleep 20","run_in_background":true}`),
	}
	got, ok := block.BackgroundBash()
	if !ok || got != "sleep 20" {
		t.Fatalf("BackgroundBash = %q, %v", got, ok)
	}
}

func TestBackgroundBashRejectsAForegroundCall(t *testing.T) {
	block := Block{Type: "tool_use", Name: "Bash", Input: []byte(`{"command":"ls"}`)}
	if _, ok := block.BackgroundBash(); ok {
		t.Fatal("a foreground Bash call must not start a job")
	}
}

func TestBackgroundBashRejectsAnotherTool(t *testing.T) {
	block := Block{
		Type:  "tool_use",
		Name:  "Read",
		Input: []byte(`{"run_in_background":true}`),
	}
	if _, ok := block.BackgroundBash(); ok {
		t.Fatal("only a Bash call starts a job")
	}
}

func TestBackgroundBashRejectsBadInput(t *testing.T) {
	block := Block{Type: "tool_use", Name: "Bash", Input: []byte(`"not an object"`)}
	if _, ok := block.BackgroundBash(); ok {
		t.Fatal("input that does not decode must not start a job")
	}
}

// The text is the one Claude Code 2.1.176 returns; see docs/protocol.md.
func TestBackgroundOutputPathReadsTheRealResult(t *testing.T) {
	const text = "Command running in background with ID: b14gccuz1. " +
		"Output is being written to: /tmp/claude-501/x/sess/tasks/b14gccuz1.output. " +
		"You will be notified when it completes. To check interim output, use Read on that file path."
	want := "/tmp/claude-501/x/sess/tasks/b14gccuz1.output"
	if got := BackgroundOutputPath(text); got != want {
		t.Fatalf("BackgroundOutputPath = %q, want %q", got, want)
	}
}

func TestBackgroundOutputPathKeepsASpaceInThePath(t *testing.T) {
	const text = "Output is being written to: /tmp/my dir/tasks/b1.output. You will be notified."
	want := "/tmp/my dir/tasks/b1.output"
	if got := BackgroundOutputPath(text); got != want {
		t.Fatalf("BackgroundOutputPath = %q, want %q", got, want)
	}
}

func TestBackgroundOutputPathIsEmptyWithoutTheMarker(t *testing.T) {
	for _, text := range []string{"", "Command running in background with ID: b1.", "Output is being written to: /tmp/b1.log."} {
		if got := BackgroundOutputPath(text); got != "" {
			t.Fatalf("BackgroundOutputPath(%q) = %q, want empty", text, got)
		}
	}
}
