package tui

// totalCost is the cost of every session this host holds, live and stored, an
// archived session included. A remote session is left out, because the peer
// account pays for it. See docs/tui/sessions/bars.md.
func (m Model) totalCost() float64 {
	var total float64
	for _, snap := range m.mgr.Snapshots() {
		total += snap.Cost
	}
	for _, meta := range m.stored {
		total += meta.Cost
	}
	return total
}
