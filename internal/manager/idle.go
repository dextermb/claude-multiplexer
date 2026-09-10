package manager

import (
	"context"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

// SetIdleAction arms a session to act on itself when it next goes idle. The
// action is a stop, and an archive after the stop when archive is true. A call
// with both false disarms. See docs/sessions.md.
func (m *Manager) SetIdleAction(name string, stop, archive bool) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	item.idleMu.Lock()
	item.stopWhenIdle = stop || archive
	item.archiveWhenIdle = archive
	item.idleFired = false
	item.idleMu.Unlock()
	return nil
}

// maybeIdleAction starts the armed idle action once the session is idle, its
// queue is empty, and it ran at least one turn. The turn guard stops a fresh
// session from acting before its first prompt. See docs/sessions.md.
func (m *Manager) maybeIdleAction(item *entry, snap session.Snapshot) {
	if snap.State != session.StateIdle || snap.Queued > 0 || snap.Turns == 0 {
		return
	}
	item.idleMu.Lock()
	if item.idleFired || (!item.stopWhenIdle && !item.archiveWhenIdle) {
		item.idleMu.Unlock()
		return
	}
	item.idleFired = true
	archive := item.archiveWhenIdle
	item.idleMu.Unlock()

	go m.idleAct(snap.Name, archive)
}

// idleAct stops the session, and archives it when asked. It runs in its own
// goroutine, not in the pump, because the stop closes stdin and the pump must
// keep draining events for the session to exit. See docs/sessions.md.
func (m *Manager) idleAct(name string, archive bool) {
	ctx, cancel := context.WithTimeout(context.Background(), session.DefaultStopGrace)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		m.notify(name, name+" failed to stop when idle: "+err.Error(), false)
		return
	}
	if archive {
		if err := m.Archive(name, true); err != nil {
			m.notify(name, name+" failed to archive when idle: "+err.Error(), false)
			return
		}
		m.notify(name, name+" archived itself when idle", true)
		return
	}
	m.notify(name, name+" stopped itself when idle", false)
}
