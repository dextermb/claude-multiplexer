package manager

import "github.com/dextermb/claude-multiplexer/internal/mcp"

// List describes every session the manager knows, live rows first.
func (m *Manager) List() []mcp.Session {
	m.mu.Lock()
	items := make([]*entry, 0, len(m.order))
	for _, name := range m.order {
		items = append(items, m.entries[name])
	}
	m.mu.Unlock()

	out := make([]mcp.Session, 0, len(items))
	for _, item := range items {
		snap := item.sess.Snapshot()
		out = append(out, mcp.Session{
			Name:    snap.Name,
			Title:   snap.Title,
			Dir:     snap.Dir,
			State:   snap.State.String(),
			Model:   snap.Model,
			Live:    true,
			Control: item.control,
			Owner:   item.metaCopy().Owner,
			Queued:  snap.Queued,
			Turns:   snap.Turns,
			Cost:    snap.Cost,
		})
	}
	for _, meta := range m.Stored() {
		state := "stored"
		if meta.Archived {
			state = "archived"
		}
		out = append(out, mcp.Session{
			Name:     meta.Name,
			Title:    meta.Title,
			Dir:      meta.Dir,
			State:    state,
			Model:    meta.Model,
			Archived: meta.Archived,
			Control:  meta.Control,
			Owner:    meta.Owner,
			Turns:    meta.Turns,
			Cost:     meta.Cost,
		})
	}
	return out
}

// Parents reports the control session that created each live session, for the
// sessions that a control session created. See docs/manager.md.
func (m *Manager) Parents() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		if parent := item.metaCopy().Parent; parent != "" {
			out[name] = parent
		}
	}
	return out
}

// Schedules reports the schedule that spawned each live session that a schedule
// started. It maps a session name to the schedule name.
func (m *Manager) Schedules() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		if sched := item.metaCopy().Scheduled; sched != "" {
			out[name] = sched
		}
	}
	return out
}

// WorkingDirs reports the working directory of each live session that set one.
func (m *Manager) WorkingDirs() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		if dir := item.metaCopy().WorkingDir; dir != "" {
			out[name] = dir
		}
	}
	return out
}

// Projects reports the project directories of each live session that has a
// project. The interface groups the diff by these directories.
func (m *Manager) Projects() map[string][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string][]string, len(m.entries))
	for name, item := range m.entries {
		if dirs := item.metaCopy().WorkingDirs; len(dirs) > 0 {
			out[name] = dirs
		}
	}
	return out
}

// Grants reports which live sessions may drive their neighbours.
func (m *Manager) Grants() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]bool, len(m.entries))
	for name, item := range m.entries {
		out[name] = item.control
	}
	return out
}
