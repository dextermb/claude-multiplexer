package manager

import (
	"context"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	schedulerTick   = 30 * time.Second
	maxSpawnPerTick = 3
)

// StartScheduler starts the clock that fires the schedules. Call it once, after
// StartMCP, because a schedule spawns a session and a session needs the MCP
// address. See docs/scheduler.md.
func (m *Manager) StartScheduler() {
	if m.schedStop != nil {
		return
	}
	m.schedStop = make(chan struct{})
	m.schedWG.Add(1)
	go m.scheduleLoop()
}

func (m *Manager) scheduleLoop() {
	defer m.schedWG.Done()
	ticker := time.NewTicker(schedulerTick)
	defer ticker.Stop()
	for {
		select {
		case <-m.schedStop:
			return
		case now := <-ticker.C:
			m.runDue(now)
		}
	}
}

// runDue fires every schedule that is due at now. A tick spawns at most
// maxSpawnPerTick fresh sessions, so a backlog does not load the machine at once.
// See docs/scheduler.md.
func (m *Manager) runDue(now time.Time) {
	spawned := 0
	for _, s := range m.dueSchedules(now) {
		if s.Session == "" {
			if spawned >= maxSpawnPerTick {
				continue
			}
			if m.sessionLive(s.LastSession) {
				m.notify(s.LastSession, s.Name+" skipped a run, its last session is still live", true)
				continue
			}
		}
		session, err := m.fireSchedule(s)
		if err != nil {
			m.notify(s.Name, s.Name+" failed to run: "+err.Error(), false)
			continue
		}
		m.markRun(s.Name, now, session)
		if s.Session == "" {
			spawned++
		}
		m.notify(session, s.Name+" ran", true)
	}
}

// dueSchedules gives a copy of each enabled schedule whose next fire is at or
// before now. It computes the next fire from the last run, so a slot missed
// during downtime fires once, and then the last run moves forward.
func (m *Manager) dueSchedules(now time.Time) []Schedule {
	m.schedMu.Lock()
	defer m.schedMu.Unlock()
	var due []Schedule
	for _, s := range m.schedules {
		if !s.Enabled {
			continue
		}
		parsed, err := cron.ParseStandard(s.Cron)
		if err != nil {
			continue
		}
		base := s.LastRun
		if base.IsZero() {
			base = s.CreatedAt
		}
		if !now.Before(parsed.Next(base)) {
			due = append(due, *s)
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i].Name < due[j].Name })
	return due
}

// fireSchedule runs one schedule. In spawn mode it starts a fresh session; in
// reuse mode it queues the prompt into the named session, and starts that session
// first when it is not live. It returns the session name.
func (m *Manager) fireSchedule(s Schedule) (string, error) {
	from := "schedule:" + s.Name
	if s.Session == "" {
		name, err := m.Spawn(context.Background(), Spec{
			Dir:            s.Dir,
			Model:          s.Model,
			PermissionMode: s.PermissionMode,
			Effort:         s.Effort,
			Control:        s.Control,
			Scheduled:      s.Name,
		})
		if err != nil {
			return "", err
		}
		if _, err := m.SendFrom(name, from, s.Prompt); err != nil {
			return name, err
		}
		return name, nil
	}

	if !m.sessionLive(s.Session) {
		if err := m.startReuse(s); err != nil {
			return "", err
		}
	}
	if _, err := m.SendFrom(s.Session, from, s.Prompt); err != nil {
		return s.Session, err
	}
	return s.Session, nil
}

// startReuse brings the reuse session of a schedule back. It resumes the stored
// session when there is one, so the session keeps its memory across fires; else
// it starts a fresh session with the same name.
func (m *Manager) startReuse(s Schedule) error {
	if meta, err := m.Meta(s.Session); err == nil && meta.ClaudeSessionID != "" {
		_, err := m.Resume(context.Background(), meta)
		return err
	}
	_, err := m.Spawn(context.Background(), Spec{
		Name:           s.Session,
		Dir:            s.Dir,
		Model:          s.Model,
		PermissionMode: s.PermissionMode,
		Effort:         s.Effort,
		Control:        s.Control,
		Scheduled:      s.Name,
	})
	return err
}

func (m *Manager) sessionLive(name string) bool {
	if name == "" {
		return false
	}
	item, err := m.entry(name)
	if err != nil {
		return false
	}
	return item.sess.State().Live()
}

func (m *Manager) markRun(name string, now time.Time, session string) {
	m.schedMu.Lock()
	s, ok := m.schedules[name]
	if !ok {
		m.schedMu.Unlock()
		return
	}
	s.LastRun = now
	if session != "" {
		s.LastSession = session
	}
	snapshot := *s
	m.schedMu.Unlock()
	_ = writeSchedule(schedulePath(m.opts.Root, name), snapshot)
}
