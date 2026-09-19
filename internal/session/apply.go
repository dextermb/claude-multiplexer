package session

import (
	"time"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func (s *Session) apply(ev protocol.Event) {
	if _, _, ok := ev.AskUserQuestion(); ok {
		s.mu.Lock()
		s.askedQuestion = true
		s.mu.Unlock()
		_ = s.Interrupt()
	}
	s.applyJobBlocks(ev)
	switch {
	case ev.Type == protocol.TypeSystem && ev.Task != nil:
		s.applyTask(ev.Subtype, ev.Task)
	case ev.IsInit() && ev.Init != nil:
		s.mu.Lock()
		s.claudeSessionID = ev.Init.SessionID
		if s.model == "" {
			s.model = ev.Init.Model
		}
		if ev.Init.PermissionMode != "" {
			s.permissionMode = ev.Init.PermissionMode
		}
		s.mu.Unlock()
		s.setStateIf(StateStarting, StateIdle)
	case ev.Type == protocol.TypeAssistant && ev.Message != nil && ev.Message.Usage != nil:
		u := ev.Message.Usage
		s.mu.Lock()
		s.contextTokens = u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens + u.OutputTokens
		s.mu.Unlock()
	case ev.Type == protocol.TypeResult && ev.Result != nil:
		s.mu.Lock()
		s.cost += ev.Result.TotalCostUSD
		s.turns += ev.Result.NumTurns
		s.lastDuration = time.Duration(ev.Result.DurationMS) * time.Millisecond
		if usage := ev.Result.Usage; usage != nil {
			s.inputTokens += usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
			s.cacheReadTokens += usage.CacheReadInputTokens
			s.cacheWriteTokens += usage.CacheCreationInputTokens
			s.outputTokens += usage.OutputTokens
		}
		if ev.Result.SessionID != "" {
			s.claudeSessionID = ev.Result.SessionID
		}
		asked := s.askedQuestion
		s.askedQuestion = false
		s.mu.Unlock()
		if asked {
			s.setState(StateWaiting)
		} else {
			s.setState(StateIdle)
		}
	}
}

// jobWrite is the output the multiplexer must write to a job file, once the lock
// is released. See docs/sessions/jobs.md.
type jobWrite struct {
	path string
	text string
}

// applyJobBlocks reads the two message blocks that a background job needs: the
// Bash call that starts it, and the tool_result that carries its output. See
// docs/sessions/jobs.md.
func (s *Session) applyJobBlocks(ev protocol.Event) {
	if ev.Message == nil {
		return
	}
	var writes []jobWrite
	s.mu.Lock()
	for _, block := range ev.Message.Content {
		switch block.Type {
		case "tool_use":
			command, ok := block.BackgroundBash()
			if !ok || block.ID == "" {
				continue
			}
			if s.pendingBash == nil {
				s.pendingBash = make(map[string]string)
			}
			s.pendingBash[block.ID] = command
		case "tool_result":
			if w, ok := s.applyLaunchResult(block); ok {
				writes = append(writes, w)
			}
		}
	}
	s.mu.Unlock()
	for _, w := range writes {
		writeJobOutput(w.path, w.text)
	}
}

// applyLaunchResult reads the tool_result of a background job. A result that
// names an external file (the older Claude Code shape) sets the job's output
// path. A result that carries the output inline (the current shape) returns a
// write, for the caller to run once the lock is released. A local agent is
// skipped: its Task tool_result would truncate the turns it captures itself.
// See docs/sessions/jobs.md.
func (s *Session) applyLaunchResult(block protocol.Block) (jobWrite, bool) {
	id := block.ToolUseID
	if id == "" {
		return jobWrite{}, false
	}
	_, pending := s.pendingBash[id]
	taskID, started := s.jobByToolUse[id]
	if !pending && !started {
		return jobWrite{}, false
	}
	text := block.Content.Text()
	if path := protocol.BackgroundOutputPath(text); path != "" {
		if started {
			if job := s.jobs[taskID]; job != nil && job.OutputPath == "" {
				job.OutputPath = path
			}
			return jobWrite{}, false
		}
		if s.pendingPath == nil {
			s.pendingPath = make(map[string]string)
		}
		s.pendingPath[id] = path
		return jobWrite{}, false
	}
	if !started || text == "" {
		return jobWrite{}, false
	}
	job := s.jobs[taskID]
	if job == nil || job.TaskType == protocol.TaskTypeLocalAgent {
		return jobWrite{}, false
	}
	if job.OutputPath == "" {
		job.OutputPath = s.generatedOutputPath(taskID)
	}
	if job.OutputPath == "" {
		return jobWrite{}, false
	}
	return jobWrite{path: job.OutputPath, text: text}, true
}

func (s *Session) applyTask(subtype string, task *protocol.Task) {
	if task.TaskID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch subtype {
	case protocol.SubtypeTaskStarted:
		if s.jobs == nil {
			s.jobs = make(map[string]*Job)
		}
		if _, ok := s.jobs[task.TaskID]; ok {
			return
		}
		job := &Job{
			ID:          task.TaskID,
			Description: task.Description,
			TaskType:    task.TaskType,
			Command:     s.pendingBash[task.ToolUseID],
			OutputPath:  s.pendingPath[task.ToolUseID],
			Status:      JobRunning,
			StartedAt:   time.Now(),
		}
		if task.TaskType == protocol.TaskTypeLocalAgent {
			if path := s.generatedOutputPath(task.TaskID); path != "" {
				job.OutputPath = path
			}
		}
		s.jobs[task.TaskID] = job
		s.jobOrder = append(s.jobOrder, task.TaskID)
		if task.ToolUseID != "" {
			if s.jobByToolUse == nil {
				s.jobByToolUse = make(map[string]string)
			}
			s.jobByToolUse[task.ToolUseID] = task.TaskID
			delete(s.pendingBash, task.ToolUseID)
			delete(s.pendingPath, task.ToolUseID)
		}
	case protocol.SubtypeTaskUpdated:
		job := s.jobs[task.TaskID]
		if job == nil || task.Patch == nil {
			return
		}
		s.setJobStatus(job, task.Patch.Status)
	case protocol.SubtypeTaskNotification:
		job := s.jobs[task.TaskID]
		if job == nil {
			return
		}
		if task.Summary != "" {
			job.Summary = task.Summary
		}
		if task.OutputFile != "" {
			job.OutputPath = task.OutputFile
		}
		s.setJobStatus(job, task.Status)
	}
}

func (s *Session) setJobStatus(job *Job, status string) {
	if !job.Status.Running() {
		return
	}
	next, terminal := classifyStatus(status)
	if !terminal {
		return
	}
	job.Status = next
	job.EndedAt = time.Now()
}
