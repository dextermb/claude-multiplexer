package manager

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStopsItselfWhenIdle(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.SetIdleAction(name, true, false); err != nil {
		t.Fatalf("SetIdleAction: %v", err)
	}

	runOneTurn(t, m, name, "hello")

	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && !snap.State.Live()
	})
}

func TestSessionArchivesItselfWhenIdle(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.SetIdleAction(name, false, true); err != nil {
		t.Fatalf("SetIdleAction: %v", err)
	}

	if err := m.Send(name, "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	path := filepath.Join(m.Root(), "sessions", name, "meta.json")
	waitFor(t, 10*time.Second, func() bool {
		if _, err := m.Snapshot(name); err == nil {
			return false
		}
		meta, err := ReadMeta(path)
		return err == nil && meta.Archived
	})
}

func TestIdleActionWaitsForTheFirstTurn(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.SetIdleAction(name, true, false); err != nil {
		t.Fatalf("SetIdleAction: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !snap.State.Live() {
		t.Fatalf("state = %v, want a live session before its first turn", snap.State)
	}
}
