package manager

import (
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/usage"
)

// evaluateReserve runs the reserve gate against a fresh poll. It trips when the
// percent remaining in the reserve window drops below the floor, and clears when
// it climbs back. On a change it pauses or resumes every hosted session. The
// reserve is read each poll, so a change to it takes effect on the next poll.
// See docs/peers.md.
func (m *Manager) evaluateReserve(u usage.Usage) {
	m.setHostingPaused(m.reserveTripped(u))
}

// reserveTripped reports whether the gate trips for a usage read. No reserve, or
// an unknown percent, never trips: an unknown header reads as unknown, never as
// zero. See docs/peers.md.
func (m *Manager) reserveTripped(u usage.Usage) bool {
	reserve := m.reserve()
	if reserve == nil {
		return false
	}
	window := u.FiveHour
	if reserve.Window == config.Window7d {
		window = u.Weekly
	}
	if !window.Known() {
		return false
	}
	return *window.Remaining < reserve.MinPercent
}

// reserve reads the configured reserve, or nil when peering has none.
func (m *Manager) reserve() *config.Reserve {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil || cfg.Peers == nil {
		return nil
	}
	return cfg.Peers.Reserve
}

// setHostingPaused records the gate state and, on a change, pauses or resumes
// every hosted session. A local session and a streamed session are untouched.
func (m *Manager) setHostingPaused(paused bool) {
	if m.hostingPaused.Swap(paused) == paused {
		return
	}
	m.mu.Lock()
	hosted := make([]*entry, 0, len(m.entries))
	for _, item := range m.entries {
		if item.metaCopy().Hosted {
			hosted = append(hosted, item)
		}
	}
	m.mu.Unlock()
	for _, item := range hosted {
		item.sess.SetPaused(paused)
	}
}

// HostingPaused reports whether the reserve gate is tripped, so the peer
// listener refuses a new hosted session. See docs/peers.md.
func (m *Manager) HostingPaused() bool { return m.hostingPaused.Load() }
