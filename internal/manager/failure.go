package manager

import (
	"errors"
	"os/exec"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

const failureStderrLines = 20

// rememberFailure records why a session ended, on the record of the session
// itself. A notice reaches a subscriber that listens at that moment and nothing
// else, so without this a stopped session cannot say why it stopped. See
// docs/sessions/lifecycle.md.
func (m *Manager) rememberFailure(item *entry, snap session.Snapshot, stderr []string) {
	record := failureOf(snap, stderr)
	_, _ = item.mutateMeta(func(meta *Meta) error {
		meta.Error = record.Error
		meta.ExitCode = record.ExitCode
		meta.Stderr = record.Stderr
		return nil
	})
}

type failure struct {
	Error    string
	ExitCode *int
	Stderr   []string
}

// failureOf reads the failure of a final snapshot. A session that ended in a
// clean way gives an empty record, which clears the record of an earlier run.
func failureOf(snap session.Snapshot, stderr []string) failure {
	if snap.Err == nil {
		return failure{}
	}
	out := failure{
		Error:  snap.Err.Error(),
		Stderr: lastLines(stderr, failureStderrLines),
	}
	var exit *exec.ExitError
	if errors.As(snap.Err, &exit) {
		code := exit.ExitCode()
		out.ExitCode = &code
	}
	return out
}

func lastLines(lines []string, limit int) []string {
	if len(lines) == 0 {
		return nil
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	out := make([]string, len(lines))
	copy(out, lines)
	return out
}

// clearFailure drops the failure record of a session that starts again, so a
// record that describes a run which is over does not read as the run that is
// live now.
func clearFailure(path string) {
	_ = mutateStoredMeta(path, func(meta *Meta) error {
		meta.Error = ""
		meta.ExitCode = nil
		meta.Stderr = nil
		return nil
	})
}
