package tui

import "time"

const burstGap = 8 * time.Millisecond

// burst marks the keys a terminal delivers too fast to be typed; see docs/tui/input.md.
type burst struct {
	last time.Time
}

func (b *burst) key(at time.Time) bool {
	previous := b.last
	b.last = at
	if previous.IsZero() {
		return false
	}
	return at.Sub(previous) < burstGap
}
