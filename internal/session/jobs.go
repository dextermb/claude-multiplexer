package session

import "time"

// JobStatus is the state of a background job. A job starts as JobRunning and
// moves to one terminal status. See docs/sessions.md.
type JobStatus uint8

const (
	JobRunning JobStatus = iota
	JobDone
	JobFailed
	JobKilled
)

func (s JobStatus) String() string {
	switch s {
	case JobDone:
		return "done"
	case JobFailed:
		return "failed"
	case JobKilled:
		return "killed"
	default:
		return "running"
	}
}

func (s JobStatus) Running() bool { return s == JobRunning }

// ParseJobStatus is the inverse of String, so a job status that crossed the peer
// stream as a word reads back to a JobStatus. An unknown word reads as
// JobRunning.
func ParseJobStatus(word string) JobStatus {
	switch word {
	case "done":
		return JobDone
	case "failed":
		return JobFailed
	case "killed":
		return JobKilled
	default:
		return JobRunning
	}
}

// StatusWord maps a Claude Code task status to the job status word shown to the
// user, so the pane, the badge, and the modal use one vocabulary.
func StatusWord(status string) string {
	st, _ := classifyStatus(status)
	return st.String()
}

// Job is one background job that Claude Code runs for a session. The session
// derives it from the task events on the stream. See docs/sessions.md.
type Job struct {
	ID          string
	Description string
	TaskType    string
	Command     string
	Summary     string
	OutputPath  string
	Status      JobStatus
	StartedAt   time.Time
	EndedAt     time.Time
}

// classifyStatus maps a Claude Code task status to a JobStatus. The second
// return is true when the status is terminal. An unknown value stays running,
// so a new Claude Code status cannot lose a job. See docs/sessions.md.
func classifyStatus(status string) (JobStatus, bool) {
	switch status {
	case "completed":
		return JobDone, true
	case "failed":
		return JobFailed, true
	case "killed", "stopped":
		return JobKilled, true
	default:
		return JobRunning, false
	}
}
