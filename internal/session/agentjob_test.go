package session

import (
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func agentSession(t *testing.T) *Session {
	t.Helper()
	dir := t.TempDir()
	return &Session{cfg: Config{TranscriptPath: filepath.Join(dir, "transcript.jsonl")}}
}

func startAgent(s *Session, taskID, toolUse string) {
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID:    taskID,
		ToolUseID: toolUse,
		TaskType:  protocol.TaskTypeLocalAgent,
	}))
}

func onlyJob(t *testing.T, s *Session) Job {
	t.Helper()
	jobs := s.jobList()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	return jobs[0]
}

func TestAgentJobGetsOutputPath(t *testing.T) {
	s := agentSession(t)
	startAgent(s, "a1", "toolu_1")
	job := onlyJob(t, s)
	if job.OutputPath == "" {
		t.Fatal("a local agent job must get an output path")
	}
	if !validOutputPath(job.OutputPath, job.ID) {
		t.Fatalf("output path %q is not a job output file", job.OutputPath)
	}
}

func TestAgentTurnCapturedAfterStart(t *testing.T) {
	s := agentSession(t)
	startAgent(s, "a1", "toolu_1")
	s.CaptureAgentTurn("toolu_1", []string{"→ Read render.go", "← 40 lines"})

	body, err := ReadOutput(onlyJob(t, s))
	if err != nil {
		t.Fatalf("ReadOutput returned %v", err)
	}
	if got := OutputLines(body); len(got) != 2 || got[0] != "→ Read render.go" || got[1] != "← 40 lines" {
		t.Fatalf("unexpected job output: %q", got)
	}
}

// A turn can arrive before task_started registers the job. The session holds it
// under the parent id, and the flush writes it in order, before a later turn.
func TestAgentTurnBeforeStartIsFlushedInOrder(t *testing.T) {
	s := agentSession(t)
	s.CaptureAgentTurn("toolu_1", []string{"first"})
	s.CaptureAgentTurn("toolu_1", []string{"second"})

	if body, _ := ReadOutput(Job{ID: "a1", OutputPath: s.agentOutputPath("a1")}); body != "" {
		t.Fatalf("nothing must be written before the job registers, got %q", body)
	}

	startAgent(s, "a1", "toolu_1")
	s.FlushAgentTurns("toolu_1")
	s.CaptureAgentTurn("toolu_1", []string{"third"})

	body, err := ReadOutput(onlyJob(t, s))
	if err != nil {
		t.Fatalf("ReadOutput returned %v", err)
	}
	got := OutputLines(body)
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("output = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
}

// An empty parent id or empty turn is a no-op, so a top-level turn never reaches
// a job file.
func TestCaptureIgnoresEmptyInput(t *testing.T) {
	s := agentSession(t)
	startAgent(s, "a1", "toolu_1")
	s.CaptureAgentTurn("", []string{"stray"})
	s.CaptureAgentTurn("toolu_1", nil)

	if body, _ := ReadOutput(onlyJob(t, s)); body != "" {
		t.Fatalf("no output expected, got %q", body)
	}
}

// A background bash job is not an agent, so it keeps the path from its stream and
// gets no agent output file.
func TestBashJobKeepsStreamPath(t *testing.T) {
	s := agentSession(t)
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID: "b1", ToolUseID: "toolu_b", TaskType: "local_bash",
	}))
	if got := onlyJob(t, s).OutputPath; got != "" {
		t.Fatalf("a bash job with no stream path must have no path, got %q", got)
	}
}
