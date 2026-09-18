package mcp

import (
	"slices"
	"strings"
	"testing"
)

// fakeLockPort fills only LockPort, so the pure lock handlers are tested through
// their port with six methods and no server.
type fakeLockPort struct {
	locks map[string][]string
	list  []Session
}

func newFakeLockPort() *fakeLockPort {
	return &fakeLockPort{locks: map[string][]string{}}
}

func (f *fakeLockPort) Locks(session string) ([]string, error) { return f.locks[session], nil }

func (f *fakeLockPort) SetLocks(labels []string, by string) ([]string, error) {
	f.locks[by] = slices.Clone(labels)
	return f.locks[by], nil
}

func (f *fakeLockPort) AddLock(label, by string) ([]string, error) {
	if !slices.Contains(f.locks[by], label) {
		f.locks[by] = append(f.locks[by], label)
	}
	return f.locks[by], nil
}

func (f *fakeLockPort) RemoveLock(label, by string) ([]string, error) {
	var kept []string
	for _, held := range f.locks[by] {
		if held != label {
			kept = append(kept, held)
		}
	}
	f.locks[by] = kept
	return kept, nil
}

func (f *fakeLockPort) ClearLocks(by string) (bool, error) {
	if len(f.locks[by]) == 0 {
		return false, nil
	}
	delete(f.locks, by)
	return true, nil
}

func (f *fakeLockPort) FindLocked(labels []string, live bool) ([]Session, error) {
	if len(labels) == 0 {
		return nil, ErrNoLock
	}
	var out []Session
	for _, item := range f.list {
		if live && !item.Live {
			continue
		}
		held := item.Locks
		if extra, ok := f.locks[item.Name]; ok {
			held = extra
		}
		match := true
		for _, label := range labels {
			if !slices.Contains(held, label) {
				match = false
				break
			}
		}
		if match {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestAddLockHandler(t *testing.T) {
	locks := newFakeLockPort()

	out, err := addLock(locks, "docs", lockIn{Lock: " deploy "})
	if err != nil {
		t.Fatalf("addLock: %v", err)
	}
	if !out.OK || len(out.Locks) != 1 || out.Locks[0] != "deploy" {
		t.Fatalf("out = %+v, want the trimmed lock", out)
	}
	if !strings.Contains(out.Message, "no other session") {
		t.Fatalf("message = %q, want it to say nobody else holds it", out.Message)
	}

	if _, err := addLock(locks, "docs", lockIn{Lock: "   "}); err != ErrNoLock {
		t.Fatalf("an empty label gave %v, want ErrNoLock", err)
	}
}

func TestAddLockHandlerNamesTheOtherHolders(t *testing.T) {
	locks := newFakeLockPort()
	locks.list = []Session{
		{Name: "api", Live: true, Locks: []string{"deploy"}},
		{Name: "docs", Live: true},
	}

	out, err := addLock(locks, "docs", lockIn{Lock: "deploy"})
	if err != nil {
		t.Fatalf("addLock: %v", err)
	}
	if !slices.Contains(out.Holders, "api") {
		t.Fatalf("holders = %v, want it to name api", out.Holders)
	}
	if !strings.Contains(out.Message, "api") {
		t.Fatalf("message = %q, want it to name the other holder", out.Message)
	}
}

func TestListLocksHandlerCountsThem(t *testing.T) {
	locks := &fakeLockPort{locks: map[string][]string{"docs": {"one", "two"}}}

	out, err := listLocks(locks, "docs", listLocksIn{})
	if err != nil {
		t.Fatalf("listLocks: %v", err)
	}
	if !strings.Contains(out.Message, "holds 2 locks") {
		t.Fatalf("message = %q, want the count", out.Message)
	}
}

func TestClearLocksHandlerReportsWhetherItChanged(t *testing.T) {
	locks := newFakeLockPort()

	out, err := clearLocks(locks, "docs")
	if err != nil {
		t.Fatalf("clearLocks: %v", err)
	}
	if out.Changed || !strings.Contains(out.Message, "held no lock") {
		t.Fatalf("out = %+v, want no change on a session with no lock", out)
	}

	if _, err := addLock(locks, "docs", lockIn{Lock: "deploy"}); err != nil {
		t.Fatalf("addLock: %v", err)
	}
	out, err = clearLocks(locks, "docs")
	if err != nil {
		t.Fatalf("clearLocks: %v", err)
	}
	if !out.Changed {
		t.Fatalf("out = %+v, want a change after a lock was taken", out)
	}
}

func TestFindLockedHandlerNeedsALabel(t *testing.T) {
	locks := newFakeLockPort()
	if _, err := findLocked(locks, findLockedIn{}); err != ErrNoLock {
		t.Fatalf("findLocked with no label gave %v, want ErrNoLock", err)
	}
}
