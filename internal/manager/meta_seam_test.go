package manager

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestConcurrentAddLockKeepsEveryLabel proves the read-modify-write of the
// record is atomic. Every AddLock reads the current locks and appends one, so
// two adds that overlap must both survive. Before the mutateMeta seam, each add
// read a copy outside the lock, so one add could overwrite the other.
func TestConcurrentAddLockKeepsEveryLabel(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")

	const n = 24
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			if _, err := m.AddLock(name, fmt.Sprintf("lock-%02d", i)); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("AddLock: %v", err)
	}

	locks, err := m.Locks(name)
	if err != nil {
		t.Fatalf("Locks: %v", err)
	}
	if len(locks) != n {
		t.Fatalf("the session holds %d locks, want %d: a concurrent add was lost", len(locks), n)
	}
}

// TestTurnWritesDoNotClobberLocks runs the pump's per-turn record write against
// the lock tool at the same time. The pump writes the turn totals while the tool
// writes a lock, and neither may lose the other's change. Run under -race, it
// also proves the two writers no longer race on the record.
func TestTurnWritesDoNotClobberLocks(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 8; i++ {
			_ = m.Send(name, fmt.Sprintf("prompt %d", i))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 24; i++ {
			if _, err := m.AddLock(name, fmt.Sprintf("lock-%02d", i)); err != nil {
				t.Errorf("AddLock: %v", err)
				return
			}
		}
	}()
	wg.Wait()

	locks, err := m.Locks(name)
	if err != nil {
		t.Fatalf("Locks: %v", err)
	}
	if len(locks) != 24 {
		t.Fatalf("the session holds %d locks, want 24: a turn write clobbered a lock", len(locks))
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns > 0
	})
}
