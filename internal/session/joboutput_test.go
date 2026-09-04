package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func bashUse(id, input string) protocol.Event {
	return protocol.Event{
		Type: protocol.TypeAssistant,
		Message: &protocol.Message{Content: protocol.Content{{
			Type: "tool_use", Name: "Bash", ID: id, Input: []byte(input),
		}}},
	}
}

func bashResult(id, text string) protocol.Event {
	return protocol.Event{
		Type: protocol.TypeUser,
		Message: &protocol.Message{Content: protocol.Content{{
			Type: "tool_result", ToolUseID: id, Content: protocol.Content{{Type: "text", Text: text}},
		}}},
	}
}

func launchText(id, path string) string {
	return "Command running in background with ID: " + id +
		". Output is being written to: " + path + ". You will be notified when it completes."
}

func TestJobKeepsItsCommandAndOutputPath(t *testing.T) {
	s := &Session{}
	s.apply(bashUse("toolu_1", `{"command":"sh -c 'echo hi'","run_in_background":true}`))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID: "b1", ToolUseID: "toolu_1", Description: "say hi", TaskType: "local_bash",
	}))
	s.apply(bashResult("toolu_1", launchText("b1", "/tmp/x/tasks/b1.output")))

	job := s.jobList()[0]
	if job.Command != "sh -c 'echo hi'" {
		t.Fatalf("Command = %q", job.Command)
	}
	if job.OutputPath != "/tmp/x/tasks/b1.output" {
		t.Fatalf("OutputPath = %q", job.OutputPath)
	}
}

// The tool_result can reach the session before task_started names the job.
func TestJobTakesAPathThatArrivedBeforeItStarted(t *testing.T) {
	s := &Session{}
	s.apply(bashUse("toolu_1", `{"command":"sleep 1","run_in_background":true}`))
	s.apply(bashResult("toolu_1", launchText("b1", "/tmp/x/tasks/b1.output")))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "b1", ToolUseID: "toolu_1"}))

	if got := s.jobList()[0].OutputPath; got != "/tmp/x/tasks/b1.output" {
		t.Fatalf("OutputPath = %q", got)
	}
}

func TestForegroundBashDoesNotGiveAJobACommand(t *testing.T) {
	s := &Session{}
	s.apply(bashUse("toolu_1", `{"command":"ls"}`))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "b1", ToolUseID: "toolu_1"}))

	if got := s.jobList()[0].Command; got != "" {
		t.Fatalf("Command = %q, want empty", got)
	}
}

func TestJobResultWithoutAPathLeavesItEmpty(t *testing.T) {
	s := &Session{}
	s.apply(bashUse("toolu_1", `{"command":"sleep 1","run_in_background":true}`))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "b1", ToolUseID: "toolu_1"}))
	s.apply(bashResult("toolu_1", "This command requires approval"))

	if got := s.jobList()[0].OutputPath; got != "" {
		t.Fatalf("OutputPath = %q, want empty", got)
	}
}

func TestNotificationCarriesTheSummaryAndThePath(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "b1", ToolUseID: "toolu_1"}))
	s.apply(taskEvent(protocol.SubtypeTaskNotification, &protocol.Task{
		TaskID:     "b1",
		Status:     "completed",
		Summary:    `Background command "x" completed (exit code 0)`,
		OutputFile: "/tmp/x/tasks/b1.output",
	}))

	job := s.jobList()[0]
	if job.Status != JobDone {
		t.Fatalf("Status = %s", job.Status)
	}
	if !strings.Contains(job.Summary, "exit code 0") {
		t.Fatalf("Summary = %q", job.Summary)
	}
	if job.OutputPath != "/tmp/x/tasks/b1.output" {
		t.Fatalf("OutputPath = %q", job.OutputPath)
	}
}

// A killed job sends an empty output_file, so the parsed path must survive.
func TestAnEmptyOutputFileKeepsTheParsedPath(t *testing.T) {
	s := &Session{}
	s.apply(bashUse("toolu_1", `{"command":"sleep 99","run_in_background":true}`))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "b1", ToolUseID: "toolu_1"}))
	s.apply(bashResult("toolu_1", launchText("b1", "/tmp/x/tasks/b1.output")))
	s.apply(taskEvent(protocol.SubtypeTaskNotification, &protocol.Task{
		TaskID: "b1", Status: "stopped", OutputFile: "",
	}))

	if got := s.jobList()[0].OutputPath; got != "/tmp/x/tasks/b1.output" {
		t.Fatalf("OutputPath = %q", got)
	}
}

func writeOutput(t *testing.T, id, body string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+".output")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadOutputReturnsTheFile(t *testing.T) {
	path := writeOutput(t, "b1", "first\nsecond\n")
	got, err := ReadOutput(Job{ID: "b1", OutputPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if got != "first\nsecond\n" {
		t.Fatalf("ReadOutput = %q", got)
	}
}

func TestReadOutputReturnsTheTailOfALargeFile(t *testing.T) {
	body := strings.Repeat("a", MaxJobOutput) + "TAIL"
	path := writeOutput(t, "b1", body)
	got, err := ReadOutput(Job{ID: "b1", OutputPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxJobOutput {
		t.Fatalf("length = %d, want %d", len(got), MaxJobOutput)
	}
	if !strings.HasSuffix(got, "TAIL") {
		t.Fatal("the read must keep the end of the file")
	}
}

func TestReadOutputWithoutAPath(t *testing.T) {
	if _, err := ReadOutput(Job{ID: "b1"}); !errors.Is(err, ErrNoOutputPath) {
		t.Fatalf("err = %v, want ErrNoOutputPath", err)
	}
}

func TestReadOutputRefusesAPathOutsideTheJob(t *testing.T) {
	good := writeOutput(t, "b1", "hi")
	dir := filepath.Dir(good)
	cases := map[string]Job{
		"relative":     {ID: "b1", OutputPath: "tasks/b1.output"},
		"another job":  {ID: "b2", OutputPath: good},
		"another name": {ID: "b1", OutputPath: filepath.Join(dir, "b1.log")},
		"another dir":  {ID: "b1", OutputPath: filepath.Join(filepath.Dir(dir), "b1.output")},
		"escapes":      {ID: "b1", OutputPath: filepath.Join(dir, "..", "..", "etc", "b1.output")},
	}
	for name, job := range cases {
		if _, err := ReadOutput(job); !errors.Is(err, ErrBadOutputPath) {
			t.Fatalf("%s: err = %v, want ErrBadOutputPath", name, err)
		}
	}
}

func TestReadOutputTreatsAMissingFileAsNoOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks", "b1.output")
	got, err := ReadOutput(Job{ID: "b1", OutputPath: path})
	if err != nil || got != "" {
		t.Fatalf("ReadOutput = %q, %v, want empty and no error", got, err)
	}
}

func TestOutputLines(t *testing.T) {
	if got := OutputLines("a\nb\n"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("OutputLines = %q", got)
	}
	if got := OutputLines("\n"); got != nil {
		t.Fatalf("OutputLines = %q, want nil", got)
	}
}
