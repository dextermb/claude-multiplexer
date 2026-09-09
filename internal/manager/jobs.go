package manager

import (
	"fmt"
	"strconv"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// Jobs reports the background jobs of a live session. A stored session has no
// running jobs, so it returns an empty list. See docs/mcp/tools.md.
func (m *Manager) Jobs(name string) ([]mcp.Job, error) {
	item, err := m.entry(name)
	if err != nil {
		if _, metaErr := m.Meta(name); metaErr == nil {
			return nil, nil
		}
		return nil, err
	}
	jobs := item.sess.Snapshot().Jobs
	out := make([]mcp.Job, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, mcp.Job{
			ID:          job.ID,
			Description: job.Description,
			TaskType:    job.TaskType,
			Status:      job.Status.String(),
			Running:     job.Status.Running(),
		})
	}
	return out, nil
}

// StopJobFrom interrupts a session and queues an instruction to kill one
// background job, so the agent runs KillShell on the exact shell. The interrupt
// ends the turn at once, so the instruction runs next. See docs/mcp/tools.md.
func (m *Manager) StopJobFrom(target, from, jobID string) (int, error) {
	item, err := m.entry(target)
	if err != nil {
		return 0, err
	}
	job, ok := findJob(item.sess.Snapshot().Jobs, jobID)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknownJob, jobID)
	}
	if !job.Status.Running() {
		return 0, fmt.Errorf("%w: %s", ErrJobNotRunning, jobID)
	}
	lines := []render.Line{{Class: render.ClassMeta, Text: "← stop job " + jobID + " from " + from}}
	if err := item.sess.Send(killPrompt(job)); err != nil {
		return 0, err
	}
	_ = item.sess.Interrupt()
	snap := item.sess.Snapshot()
	m.bus.publishLines(item.lines, lines, Event{
		Session:  target,
		Kind:     session.KindState,
		Lines:    lines,
		Partial:  item.partialText(),
		Snapshot: snap,
	})
	return snap.Queued, nil
}

func findJob(jobs []session.Job, id string) (session.Job, bool) {
	for _, job := range jobs {
		if job.ID == id {
			return job, true
		}
	}
	return session.Job{}, false
}

func killPrompt(job session.Job) string {
	text := "Stop the background job now. Use the KillShell tool with shell_id " + strconv.Quote(job.ID) + "."
	if job.Description != "" {
		text += " The job is: " + job.Description + "."
	}
	return text
}
