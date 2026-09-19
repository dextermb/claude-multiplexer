package session

import (
	"os"
	"path/filepath"
	"strings"
)

// The turns of a local agent do not belong in the session pane. The multiplexer
// captures them to the agent job's output file, which the jobs dialog reads the
// same way it reads a background bash job. Claude Code writes no file for an
// agent, so the multiplexer writes one. See docs/sessions/jobs.md.

// CaptureAgentTurn routes one rendered turn of a local agent to its job output
// file. It resolves the parent tool_use id to the job, and appends the turn to
// the job's file. A turn that arrives before task_started registers the job has
// no job yet, so it is held under the parent id until FlushAgentTurns drains it.
//
// The caller is the one pump goroutine, so every write to a job file is
// sequential and ordered. See docs/sessions/jobs.md.
func (s *Session) CaptureAgentTurn(parentToolUse string, lines []string) {
	if parentToolUse == "" || len(lines) == 0 {
		return
	}
	s.mu.Lock()
	taskID, known := s.jobByToolUse[parentToolUse]
	if !known {
		if s.pendingAgentOut == nil {
			s.pendingAgentOut = make(map[string][]string)
		}
		s.pendingAgentOut[parentToolUse] = append(s.pendingAgentOut[parentToolUse], lines...)
		s.mu.Unlock()
		return
	}
	path := s.jobPath(taskID)
	s.mu.Unlock()
	appendJobOutput(path, lines)
}

// FlushAgentTurns writes the turns that arrived before the agent job registered,
// once task_started names the job. The caller is the pump goroutine, on the
// task_started event, so the flush and the later direct writes stay in order.
func (s *Session) FlushAgentTurns(parentToolUse string) {
	if parentToolUse == "" {
		return
	}
	s.mu.Lock()
	buffered := s.pendingAgentOut[parentToolUse]
	delete(s.pendingAgentOut, parentToolUse)
	if len(buffered) == 0 {
		s.mu.Unlock()
		return
	}
	taskID := s.jobByToolUse[parentToolUse]
	path := s.jobPath(taskID)
	s.mu.Unlock()
	appendJobOutput(path, buffered)
}

// jobPath returns the output path of a job, or "" when there is none. The caller
// holds s.mu.
func (s *Session) jobPath(taskID string) string {
	if job := s.jobs[taskID]; job != nil {
		return job.OutputPath
	}
	return ""
}

// appendJobOutput appends rendered turns to the job output file, one line each.
// It is best-effort, the same as ReadOutput, because a job that cannot write its
// output must not end the session. An empty path drops the turns, which is how a
// session with no state directory behaves.
func appendJobOutput(path string, lines []string) {
	if path == "" || len(lines) == 0 {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(strings.Join(lines, "\n") + "\n")
}
