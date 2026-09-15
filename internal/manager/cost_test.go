package manager

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func readLedger(t *testing.T, root string) []CostEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, LedgerFile))
	if err != nil {
		t.Fatalf("read the ledger: %v", err)
	}
	var out []CostEntry
	for _, line := range splitLines(string(data)) {
		var entry CostEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("the ledger holds a line that is not an entry: %q", line)
		}
		out = append(out, entry)
	}
	return out
}

func splitLines(text string) []string {
	var out []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '\n' {
			continue
		}
		if i > start {
			out = append(out, text[start:i])
		}
		start = i + 1
	}
	if start < len(text) {
		out = append(out, text[start:])
	}
	return out
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write the settings: %v", err)
	}
	return path
}

func TestEachTurnWritesOneLedgerLineOfItsOwnCost(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "spend", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "first")
	runOneTurn(t, m, name, "second")
	waitFor(t, 10*time.Second, func() bool { return len(readLedger(t, m.Root())) == 2 })

	entries := readLedger(t, m.Root())
	for i, entry := range entries {
		if entry.Session != name {
			t.Fatalf("line %d names %q, want %q", i, entry.Session, name)
		}
		if entry.Cost != 0.25 {
			t.Fatalf("line %d holds %v, want the cost of one turn and not a total", i, entry.Cost)
		}
	}
}

func TestAResumedSessionLogsOnlyItsNewTurns(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "again", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "first")
	waitForMeta(t, m, name)
	retire(t, m, name)
	waitFor(t, 10*time.Second, func() bool { return len(m.Stored()) == 1 })

	resumed, err := m.Resume(context.Background(), m.Stored()[0])
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	runOneTurn(t, m, resumed, "second")
	waitFor(t, 10*time.Second, func() bool { return len(readLedger(t, m.Root())) == 2 })

	for i, entry := range readLedger(t, m.Root()) {
		if entry.Cost != 0.25 {
			t.Fatalf("line %d holds %v, want one turn of cost", i, entry.Cost)
		}
	}
}

func TestTheWindowCountsTodayAndLeavesTheDaysBefore(t *testing.T) {
	m := newTestManager(t)
	old := CostEntry{At: time.Now().UTC().AddDate(0, 0, -3), Session: "old", Cost: 5}
	if err := m.ledger.append(old); err != nil {
		t.Fatalf("append: %v", err)
	}
	name, err := m.Spawn(context.Background(), Spec{Name: "today", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	waitFor(t, 10*time.Second, func() bool { return len(readLedger(t, m.Root())) == 2 })

	total, window := m.WindowCost()
	if window != "1d" {
		t.Fatalf("window = %q, want the 1d default", window)
	}
	if total != 0.25 {
		t.Fatalf("total = %v, want today's turn alone", total)
	}
}

func TestTheAllWindowCountsEverySessionAndNamesNoWindow(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{writeConfig(t, `{"costWindow":"all"}`)}
	if err := m.ledger.append(CostEntry{
		At: time.Now().UTC().AddDate(0, 0, -3), Session: "old", Cost: 5,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	name, err := m.Spawn(context.Background(), Spec{Name: "live", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")

	total, window := m.WindowCost()
	if window != "" {
		t.Fatalf("window = %q, want no label under all", window)
	}
	if total != 0.25 {
		t.Fatalf("total = %v, want the live session and not the ledger", total)
	}
}

func TestTheBackfillDatesAStoredSessionAtItsLastActivity(t *testing.T) {
	root := t.TempDir()
	name := "before"
	dir := filepath.Join(root, "sessions", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	active := time.Now().UTC().AddDate(0, 0, -2)
	meta := Meta{Name: name, Cost: 1.25, Turns: 3, LastActiveAt: active}
	if err := writeMeta(filepath.Join(dir, "meta.json"), meta); err != nil {
		t.Fatalf("writeMeta: %v", err)
	}

	l := newLedger(root)
	if err := l.backfill([]Meta{meta}); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	entries := readLedger(t, root)
	if len(entries) != 1 {
		t.Fatalf("entries = %+v, want one", entries)
	}
	if !entries[0].Backfilled || entries[0].Cost != 1.25 || !entries[0].At.Equal(active) {
		t.Fatalf("entry = %+v, want the lifetime cost dated at the last activity", entries[0])
	}

	if err := l.backfill([]Meta{meta}); err != nil {
		t.Fatalf("the second backfill: %v", err)
	}
	if entries := readLedger(t, root); len(entries) != 1 {
		t.Fatalf("the backfill ran twice: %+v", entries)
	}
}

func TestThePruneDropsTheLinesPastTheKeep(t *testing.T) {
	root := t.TempDir()
	l := newLedger(root)
	stale := CostEntry{At: time.Now().UTC().Add(-2 * ledgerKeep), Session: "stale", Cost: 9}
	fresh := CostEntry{At: time.Now().UTC(), Session: "fresh", Cost: 1}
	for _, entry := range []CostEntry{stale, fresh} {
		if err := l.append(entry); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	if err := l.prune(time.Now().Add(-ledgerKeep)); err != nil {
		t.Fatalf("prune: %v", err)
	}
	entries := readLedger(t, root)
	if len(entries) != 1 || entries[0].Session != "fresh" {
		t.Fatalf("entries = %+v, want the fresh line alone", entries)
	}
}
