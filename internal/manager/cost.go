package manager

import (
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// startLedger loads the ledger, writes the one-time backfill when there is no
// file, and drops the lines that are too old to reach any window.
func (m *Manager) startLedger() {
	metas, err := scanMeta(m.opts.Root)
	if err == nil {
		_ = m.ledger.backfill(metas)
	}
	m.ledger.load()
	_ = m.ledger.prune(time.Now().Add(-ledgerKeep))
}

// WindowCost is the total the status bar shows, and the label that names the
// window it covers. An "all" window sums every session this host holds, and any
// other window sums the ledger lines inside it. The label is empty for "all".
// See docs/cost.md.
func (m *Manager) WindowCost() (float64, string) {
	window := config.DefaultCostWindow
	if cfg, err := config.Load(m.opts.ConfigPaths...); err == nil {
		window = config.CostWindow(cfg.CostWindow)
	}
	start, ok := config.WindowStart(window, time.Now())
	if !ok {
		return m.lifetimeCost(), ""
	}
	return m.ledger.sumSince(start), window
}

// lifetimeCost adds the live sessions and the stored records. A remote session
// is left out, because the peer account pays for it.
func (m *Manager) lifetimeCost() float64 {
	var total float64
	for _, snap := range m.Snapshots() {
		total += snap.Cost
	}
	for _, meta := range m.Stored() {
		total += meta.Cost
	}
	return total
}

// costEntry gives the ledger line for one snapshot, and reports whether the
// snapshot moved any counter since the last line. The snapshot holds lifetime
// totals, so the line carries the difference. See docs/cost.md.
func (e *entry) costEntry(snap session.Snapshot, at time.Time) (CostEntry, bool) {
	e.costMu.Lock()
	defer e.costMu.Unlock()
	entry := CostEntry{
		At:               at.UTC(),
		Session:          snap.Name,
		Cost:             snap.Cost - e.written.cost,
		InputTokens:      snap.InputTokens - e.written.input,
		OutputTokens:     snap.OutputTokens - e.written.output,
		CacheReadTokens:  snap.CacheReadTokens - e.written.cacheRead,
		CacheWriteTokens: snap.CacheWriteTokens - e.written.cacheWrite,
	}
	if entry.Cost <= 0 && entry.InputTokens <= 0 && entry.OutputTokens <= 0 {
		return CostEntry{}, false
	}
	e.written = totals{
		turns:      snap.Turns,
		cost:       snap.Cost,
		input:      snap.InputTokens,
		cacheRead:  snap.CacheReadTokens,
		cacheWrite: snap.CacheWriteTokens,
		output:     snap.OutputTokens,
	}
	return entry, true
}
