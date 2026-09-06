package manager

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func (m *Manager) rememberSession(item *entry, snap session.Snapshot) {
	if snap.Turns == 0 {
		return
	}
	item.metaMu.Lock()
	next := item.meta
	next.ClaudeSessionID = snap.ClaudeSessionID
	next.Title = snap.Title
	next.Model = snap.Model
	next.PermissionMode = snap.PermissionMode
	next.Effort = snap.Effort
	next.Control = item.control
	next.Turns = item.base.turns + snap.Turns
	next.Cost = item.base.cost + snap.Cost
	next.InputTokens = item.base.input + snap.InputTokens
	next.OutputTokens = item.base.output + snap.OutputTokens
	if next.sameAs(item.meta) && !item.meta.LastActiveAt.IsZero() {
		item.metaMu.Unlock()
		return
	}
	next.LastActiveAt = time.Now()
	item.meta = next
	item.metaMu.Unlock()

	_ = writeMeta(item.path, next)
}

func (m *Manager) Stored() []Meta {
	metas, err := scanMeta(m.opts.Root)
	if err != nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Meta, 0, len(metas))
	for _, meta := range metas {
		if _, live := m.entries[meta.Name]; live {
			continue
		}
		out = append(out, meta)
	}
	return out
}

func (m *Manager) Replay(name string) []render.Line {
	file, err := os.Open(transcriptPath(m.opts.Root, name))
	if err != nil {
		return nil
	}
	defer file.Close()

	reader := protocol.NewReader(file)
	var lines []render.Line
	for {
		ev, err := reader.Next()
		if errors.Is(err, protocol.ErrNotJSON) {
			continue
		}
		if err != nil {
			break
		}
		lines = append(lines, m.opts.Renderer.Lines(session.Event{
			Kind:     session.KindProtocol,
			Protocol: ev,
		})...)
	}
	if len(lines) > m.opts.MaxLines {
		lines = lines[len(lines)-m.opts.MaxLines:]
	}
	return lines
}

// Todos returns the current task list of a session. A live session holds the
// list in memory; a stored session gets it rebuilt from the transcript. See
// docs/tui/tasks.md.
func (m *Manager) Todos(name string) []protocol.Todo {
	if item, err := m.entry(name); err == nil {
		return item.todoList()
	}
	tasks := m.tasksFromTranscript(name)
	return tasks.List()
}

func (m *Manager) tasksFromTranscript(name string) protocol.TaskTracker {
	var tasks protocol.TaskTracker
	file, err := os.Open(transcriptPath(m.opts.Root, name))
	if err != nil {
		return tasks
	}
	defer file.Close()

	reader := protocol.NewReader(file)
	for {
		ev, err := reader.Next()
		if errors.Is(err, protocol.ErrNotJSON) {
			continue
		}
		if err != nil {
			break
		}
		tasks.Apply(ev)
	}
	return tasks
}

func (m *Manager) Archive(name string, archived bool) error {
	m.mu.Lock()
	item, live := m.entries[name]
	m.mu.Unlock()
	if live && item.sess.State().Live() {
		return fmt.Errorf("%w: %s", ErrStillLive, name)
	}

	path := metaPath(m.opts.Root, name)
	meta, err := ReadMeta(path)
	if err != nil {
		return err
	}
	meta.Archived = archived
	if archived {
		meta.ArchivedAt = time.Now()
	} else {
		meta.ArchivedAt = time.Time{}
	}
	if err := writeMeta(path, meta); err != nil {
		return err
	}

	if live {
		m.mu.Lock()
		delete(m.entries, name)
		m.order = removeName(m.order, name)
		m.mu.Unlock()
	}
	return nil
}

func (m *Manager) Meta(name string) (Meta, error) {
	return ReadMeta(metaPath(m.opts.Root, name))
}

func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}

func (m *Manager) Snapshots() []session.Snapshot {
	m.mu.Lock()
	items := make([]*entry, 0, len(m.order))
	for _, name := range m.order {
		items = append(items, m.entries[name])
	}
	m.mu.Unlock()

	out := make([]session.Snapshot, 0, len(items))
	for _, item := range items {
		out = append(out, item.view())
	}
	return out
}

func (m *Manager) Snapshot(name string) (session.Snapshot, error) {
	item, err := m.entry(name)
	if err != nil {
		return session.Snapshot{}, err
	}
	return item.view(), nil
}

func (m *Manager) Lines(name string) []render.Line {
	item, err := m.entry(name)
	if err != nil {
		return nil
	}
	return item.lines.all()
}

func (m *Manager) AppendLines(name string, lines []render.Line) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	item.lines.append(lines)
	return nil
}

func (m *Manager) Stderr(name string) []string {
	item, err := m.entry(name)
	if err != nil {
		return nil
	}
	return item.sess.Stderr()
}

func (m *Manager) TotalCost() float64 {
	var total float64
	for _, snap := range m.Snapshots() {
		total += snap.Cost
	}
	return total
}
