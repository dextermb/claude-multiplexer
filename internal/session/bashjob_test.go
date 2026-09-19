package session

import (
	"os"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func toolResultEvent(toolUseID, text string) protocol.Event {
	return protocol.Event{
		Type: protocol.TypeUser,
		Message: &protocol.Message{
			Role: "user",
			Content: protocol.Content{{
				Type:      "tool_result",
				ToolUseID: toolUseID,
				Content:   protocol.Content{{Type: "text", Text: text}},
			}},
		},
	}
}

// The current Claude Code carries a background bash job's whole output inline in
// the tool_result, with no "written to" marker and an empty output_file. The
// multiplexer writes it to the job's file, so the jobs dialog can read it.
func TestBackgroundBashInlineOutputCaptured(t *testing.T) {
	s := agentSession(t)
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID: "b1", ToolUseID: "toolu_1", TaskType: "local_bash",
		Description: "go test ./...",
	}))
	s.apply(taskEvent(protocol.SubtypeTaskNotification, &protocol.Task{
		TaskID: "b1", Status: "completed",
	}))
	s.apply(toolResultEvent("toolu_1", "ok  pkg/a\nok  pkg/b"))

	job := onlyJob(t, s)
	if job.Status != JobDone {
		t.Fatalf("status = %v, want done", job.Status)
	}
	body, err := ReadOutput(job)
	if err != nil {
		t.Fatalf("ReadOutput returned %v", err)
	}
	got := OutputLines(body)
	if len(got) != 2 || got[0] != "ok  pkg/a" || got[1] != "ok  pkg/b" {
		t.Fatalf("unexpected job output: %q", got)
	}
}

// A replay of the same result on restart must not stack the output, because the
// write truncates the file.
func TestBackgroundBashInlineOutputIsIdempotent(t *testing.T) {
	s := agentSession(t)
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID: "b1", ToolUseID: "toolu_1", TaskType: "local_bash",
	}))
	s.apply(toolResultEvent("toolu_1", "line one\nline two"))
	s.apply(toolResultEvent("toolu_1", "line one\nline two"))

	body, err := ReadOutput(onlyJob(t, s))
	if err != nil {
		t.Fatalf("ReadOutput returned %v", err)
	}
	if got := OutputLines(body); len(got) != 2 {
		t.Fatalf("a truncate write must not stack the output, got %q", got)
	}
}

// The older Claude Code names an external file in the tool_result. That path
// still wins, even when task_started arrives first, and no generated file holds.
func TestBackgroundBashMarkerSetsExternalPath(t *testing.T) {
	s := agentSession(t)
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID: "b1", ToolUseID: "toolu_1", TaskType: "local_bash",
	}))
	ext := "/tmp/claude/x/tasks/b1.output"
	s.apply(toolResultEvent("toolu_1",
		"Command running in background. Output is being written to: "+ext+". You will be notified."))

	job := onlyJob(t, s)
	if job.OutputPath != ext {
		t.Fatalf("OutputPath = %q, want %q", job.OutputPath, ext)
	}
	if _, err := os.Stat(s.generatedOutputPath("b1")); !os.IsNotExist(err) {
		t.Fatalf("a marker result must not write a generated file, stat err = %v", err)
	}
}

// A local agent's Task also ends with a top-level tool_result. It must not
// truncate the turn-by-turn file the agent captures itself.
func TestLocalAgentToolResultDoesNotOverwriteTurns(t *testing.T) {
	s := agentSession(t)
	startAgent(s, "a1", "toolu_1")
	s.CaptureAgentTurn("toolu_1", []string{"→ Read x.go", "← 40 lines"})
	s.apply(toolResultEvent("toolu_1", "the agent's final summary"))

	body, err := ReadOutput(onlyJob(t, s))
	if err != nil {
		t.Fatalf("ReadOutput returned %v", err)
	}
	got := OutputLines(body)
	if len(got) != 2 || got[0] != "→ Read x.go" {
		t.Fatalf("the agent turns must survive its tool_result, got %q", got)
	}
}

// A foreground bash also returns inline content, but it is not a job, so the
// multiplexer must not capture it.
func TestForegroundBashResultNotCaptured(t *testing.T) {
	s := agentSession(t)
	s.apply(toolResultEvent("toolu_1", "some inline output"))
	if got := len(s.jobList()); got != 0 {
		t.Fatalf("a foreground bash must add no job, got %d", got)
	}
}
