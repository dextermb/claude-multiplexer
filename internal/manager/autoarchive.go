package manager

import (
	"strconv"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

const autoArchiveTick = time.Hour

// StartAutoArchive starts the clock that archives a stopped session after it is
// idle for the configured number of days. Call it once, after StartScheduler.
// See docs/sessions/lifecycle.md.
func (m *Manager) StartAutoArchive() {
	if m.archiveStop != nil {
		return
	}
	m.archiveStop = make(chan struct{})
	m.archiveWG.Add(1)
	go m.archiveLoop()
}

func (m *Manager) archiveLoop() {
	defer m.archiveWG.Done()
	ticker := time.NewTicker(autoArchiveTick)
	defer ticker.Stop()
	for {
		select {
		case <-m.archiveStop:
			return
		case now := <-ticker.C:
			m.sweepArchive(now)
		}
	}
}

// sweepArchive archives every stopped session that is idle past the configured
// days. It reads the setting on each sweep, so a change takes effect on the next
// tick. A nil setting, or a value below 1, archives nothing.
func (m *Manager) sweepArchive(now time.Time) {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return
	}
	if cfg.AutoArchiveDays == nil || *cfg.AutoArchiveDays < 1 {
		return
	}
	days := *cfg.AutoArchiveDays
	for _, name := range dueForArchive(m.Stored(), days, now) {
		if err := m.Archive(name, true); err != nil {
			continue
		}
		m.notify(name, name+" was auto-archived after "+strconv.Itoa(days)+" days idle", true)
	}
}

// dueForArchive gives the names of the stored sessions the sweep must archive: a
// session that is not archived yet, has a real last-active time, and is idle for
// more than the given days.
func dueForArchive(metas []Meta, days int, now time.Time) []string {
	cutoff := time.Duration(days) * 24 * time.Hour
	var due []string
	for _, meta := range metas {
		if meta.Archived || meta.LastActiveAt.IsZero() {
			continue
		}
		if now.Sub(meta.LastActiveAt) >= cutoff {
			due = append(due, meta.Name)
		}
	}
	return due
}
