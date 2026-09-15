package manager

import (
	"context"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestTotalAddsEveryLifetimeCounterAndLeavesTheContext(t *testing.T) {
	item := &entry{base: totals{
		turns:      3,
		cost:       1.5,
		input:      400,
		cacheRead:  300,
		cacheWrite: 80,
		output:     20,
	}}
	got := item.total(session.Snapshot{
		Turns:            1,
		Cost:             0.5,
		InputTokens:      100,
		CacheReadTokens:  60,
		CacheWriteTokens: 30,
		OutputTokens:     10,
		ContextTokens:    12000,
	})
	if got.Turns != 4 || got.Cost != 2.0 {
		t.Fatalf("total gave turns %d cost %v, want 4 and 2", got.Turns, got.Cost)
	}
	if got.InputTokens != 500 || got.OutputTokens != 30 {
		t.Fatalf("total gave %d in %d out, want 500 and 30", got.InputTokens, got.OutputTokens)
	}
	if got.CacheReadTokens != 360 || got.CacheWriteTokens != 110 {
		t.Fatalf("total gave cache %d read %d write, want 360 and 110",
			got.CacheReadTokens, got.CacheWriteTokens)
	}
	if got.ContextTokens != 12000 {
		t.Fatalf("the context fill is not a lifetime counter, and total changed it to %d",
			got.ContextTokens)
	}
}

func TestResumedSessionReportsTheLifetimeTotals(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "carry", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "first")
	waitForMeta(t, m, name)
	retire(t, m, name)
	waitFor(t, 10*time.Second, func() bool { return len(m.Stored()) == 1 })

	stored := m.Stored()[0]
	resumed, err := m.Resume(context.Background(), stored)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	snap, err := m.Snapshot(resumed)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Cost != stored.Cost || snap.Turns != stored.Turns {
		t.Fatalf("a resumed session reports cost %v turns %d, want the stored %v and %d",
			snap.Cost, snap.Turns, stored.Cost, stored.Turns)
	}

	runOneTurn(t, m, resumed, "second")
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(resumed)
		return err == nil && snap.Turns == 2
	})

	snap, err = m.Snapshot(resumed)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Cost != 0.5 {
		t.Fatalf("cost = %v, want the two turns summed to 0.5", snap.Cost)
	}
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(metaPath(m.Root(), resumed))
		return err == nil && meta.Turns == snap.Turns
	})
	meta := waitForMeta(t, m, resumed)
	if meta.Cost != snap.Cost || meta.Turns != snap.Turns {
		t.Fatalf("the meta %+v and the snapshot %+v disagree", meta, snap)
	}

	for _, listed := range m.List() {
		if listed.Name == resumed && listed.Cost != snap.Cost {
			t.Fatalf("list reports cost %v, want %v", listed.Cost, snap.Cost)
		}
	}
}

func TestAResumedSessionThatArmsTheIdleStopStillRunsItsPrompt(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "armed", Dir: t.TempDir()})
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
	if err := m.SetIdleAction(resumed, true, false); err != nil {
		t.Fatalf("SetIdleAction: %v", err)
	}

	runOneTurn(t, m, resumed, "second")
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(metaPath(m.Root(), resumed))
		return err == nil && meta.Turns == 2
	})
}
