package tui

import "fmt"

// costSeg is the total the status bar draws. The manager sums it, so the label
// names the window the figure covers, and an empty label means the whole
// history. See docs/cost.md.
func (m Model) costSeg() string {
	if m.costWindow == "" {
		return fmt.Sprintf("$%.4f", m.cost)
	}
	return fmt.Sprintf("$%.4f %s", m.cost, m.costWindow)
}
