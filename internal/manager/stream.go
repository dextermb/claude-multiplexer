package manager

import (
	"context"

	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// sessionBuffer returns the line buffer of a live local session or a streamed
// session, or nil when the name has no live buffer. The buffer carries the
// watermark a stream reads. See docs/peers.md.
func (m *Manager) sessionBuffer(name string) *lineBuffer {
	if re := m.remote(name); re != nil {
		return re.lines
	}
	if item, err := m.entry(name); err == nil {
		return item.lines
	}
	return nil
}

// streamSession replays the session's current lines, then tails its live events,
// converting each to the wire form. It reads the lines and a watermark together
// with the subscription, then drops the lines of a live event already in the
// replay, so the boundary never doubles the output. The channel closes when the
// context is done. See docs/peers.md.
func (m *Manager) streamSession(ctx context.Context, name string) <-chan wire.Event {
	out := make(chan wire.Event, 64)

	var (
		sub       *Subscription
		lines     []render.Line
		watermark uint64
	)
	if buf := m.sessionBuffer(name); buf != nil {
		sub, lines, watermark = m.bus.subscribeAt(buf, DefaultSubscriberBuffer)
	} else {
		sub = m.bus.Subscribe(DefaultSubscriberBuffer)
		lines = m.Lines(name)
	}

	go func() {
		defer close(out)
		defer sub.Close()

		replay := wire.Event{Session: name, Lines: lines}
		if snap, err := m.Snapshot(name); err == nil {
			replay.Snapshot = wireSnapshot(snap)
		}
		select {
		case out <- replay:
		case <-ctx.Done():
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.C:
				if !ok {
					return
				}
				if ev.Session != name {
					continue
				}
				if ev.Seq != 0 && ev.Seq <= watermark {
					ev.Lines = nil
				}
				select {
				case out <- wireEvent(ev):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// wireEvent converts a manager event to its wire form.
func wireEvent(ev Event) wire.Event {
	return wire.Event{
		Session:    ev.Session,
		Kind:       int(ev.Kind),
		Lines:      ev.Lines,
		Partial:    ev.Partial,
		Snapshot:   wireSnapshot(ev.Snapshot),
		Closed:     ev.Closed,
		Questions:  ev.Questions,
		QuestionID: ev.QuestionID,
		Todos:      ev.Todos,
	}
}

// wireSnapshot converts a session snapshot to its wire form, with the error as a
// string and the duration in milliseconds.
func wireSnapshot(s session.Snapshot) wire.Snapshot {
	out := wire.Snapshot{
		Name:            s.Name,
		Title:           s.Title,
		Dir:             s.Dir,
		Model:           s.Model,
		PermissionMode:  s.PermissionMode,
		Effort:          s.Effort,
		State:           s.State.String(),
		ClaudeSessionID: s.ClaudeSessionID,
		Cost:            s.Cost,
		Turns:           s.Turns,
		Queued:          s.Queued,
		LastDurationMS:  s.LastDuration.Milliseconds(),
		InputTokens:     s.InputTokens,
		OutputTokens:    s.OutputTokens,
		ContextTokens:   s.ContextTokens,
	}
	if s.Err != nil {
		out.Err = s.Err.Error()
	}
	for _, j := range s.Jobs {
		out.Jobs = append(out.Jobs, wire.Job{
			ID:          j.ID,
			Description: j.Description,
			TaskType:    j.TaskType,
			Status:      j.Status.String(),
			Running:     j.Status.Running(),
		})
	}
	return out
}
