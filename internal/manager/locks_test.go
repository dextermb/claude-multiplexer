package manager

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func spawnFor(t *testing.T, m *Manager, name string) string {
	t.Helper()
	got, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: name})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	return got
}

func TestLocksAddAndRemove(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")

	locks, err := m.AddLock(name, "deploy")
	if err != nil {
		t.Fatalf("AddLock: %v", err)
	}
	if len(locks) != 1 || locks[0] != "deploy" {
		t.Fatalf("locks = %v, want [deploy]", locks)
	}
	if locks, err = m.AddLock(name, "deploy"); err != nil || len(locks) != 1 {
		t.Fatalf("a repeat add gave %v (err %v), want one lock", locks, err)
	}
	if locks, err = m.AddLock(name, " repo:cmux "); err != nil || len(locks) != 2 {
		t.Fatalf("AddLock = %v (err %v), want two locks", locks, err)
	}
	if locks[1] != "repo:cmux" {
		t.Fatalf("the label kept its outer space: %q", locks[1])
	}

	if _, err := m.AddLock(name, "   "); !errors.Is(err, ErrNoLock) {
		t.Fatalf("an empty label gave %v, want ErrNoLock", err)
	}

	if locks, err = m.RemoveLock(name, "deploy"); err != nil || len(locks) != 1 {
		t.Fatalf("RemoveLock = %v (err %v), want one lock left", locks, err)
	}
	if locks, err = m.RemoveLock(name, "deploy"); err != nil || len(locks) != 1 {
		t.Fatalf("removing a lock it does not hold gave %v (err %v)", locks, err)
	}
}

func TestLocksSurviveOnDisk(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")

	if _, err := m.SetLocks(name, []string{"one", "two", "one"}); err != nil {
		t.Fatalf("SetLocks: %v", err)
	}
	meta, err := ReadMeta(metaPath(m.opts.Root, name))
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if len(meta.Locks) != 2 || meta.Locks[0] != "one" || meta.Locks[1] != "two" {
		t.Fatalf("meta.Locks = %v, want [one two] with the repeat dropped", meta.Locks)
	}
}

func TestClearLocksReportsWhetherItChangedAnything(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")

	if changed, err := m.ClearLocks(name); err != nil || changed {
		t.Fatalf("ClearLocks = %v (err %v), want false on a session with no lock", changed, err)
	}
	if _, err := m.AddLock(name, "deploy"); err != nil {
		t.Fatalf("AddLock: %v", err)
	}
	changed, err := m.ClearLocks(name)
	if err != nil || !changed {
		t.Fatalf("ClearLocks = %v (err %v), want true", changed, err)
	}
	if locks, _ := m.Locks(name); len(locks) != 0 {
		t.Fatalf("the session holds %v after a clear", locks)
	}
}

func TestLocksOfAnUnknownSession(t *testing.T) {
	m := withMCP(t)
	if _, err := m.Locks("nowhere"); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("Locks = %v, want ErrUnknownSession", err)
	}
}

func TestFindLockedMatchesEveryLabel(t *testing.T) {
	m := withMCP(t)
	api := spawnFor(t, m, "api")
	docs := spawnFor(t, m, "docs")

	if _, err := m.SetLocks(api, []string{"deploy", "repo:cmux"}); err != nil {
		t.Fatalf("SetLocks: %v", err)
	}
	if _, err := m.SetLocks(docs, []string{"deploy"}); err != nil {
		t.Fatalf("SetLocks: %v", err)
	}

	found, err := m.FindLocked([]string{"deploy"}, false)
	if err != nil {
		t.Fatalf("FindLocked: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("found %d sessions for one label, want 2", len(found))
	}

	if found, err = m.FindLocked([]string{"deploy", "repo:cmux"}, false); err != nil || len(found) != 1 {
		t.Fatalf("found %v (err %v), want only the session holding both", found, err)
	}
	if found[0].Name != api {
		t.Fatalf("found %q, want %q", found[0].Name, api)
	}

	if found, err = m.FindLocked([]string{"nothing"}, false); err != nil || len(found) != 0 {
		t.Fatalf("found %v (err %v) for a label nobody holds", found, err)
	}
	if _, err = m.FindLocked(nil, false); !errors.Is(err, ErrNoLock) {
		t.Fatalf("FindLocked with no label gave %v, want ErrNoLock", err)
	}
}

func TestLockToolsNotifyTheHuman(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")
	sub := m.Subscribe(256)
	defer sub.Close()

	tools := &bridge{m: m}
	if _, err := tools.AddLock("deploy", name); err != nil {
		t.Fatalf("AddLock: %v", err)
	}
	if ev := awaitNotice(t, sub); !strings.Contains(ev.Notice, "api took the lock deploy") {
		t.Fatalf("notice = %q", ev.Notice)
	}

	if _, err := tools.RemoveLock("deploy", name); err != nil {
		t.Fatalf("RemoveLock: %v", err)
	}
	if ev := awaitNotice(t, sub); !strings.Contains(ev.Notice, "api released the lock deploy") {
		t.Fatalf("notice = %q", ev.Notice)
	}

	if _, err := tools.SetLocks([]string{"one", "two"}, name); err != nil {
		t.Fatalf("SetLocks: %v", err)
	}
	if ev := awaitNotice(t, sub); !strings.Contains(ev.Notice, "api holds 2 locks") {
		t.Fatalf("notice = %q", ev.Notice)
	}

	changed, err := tools.ClearLocks(name)
	if err != nil || !changed {
		t.Fatalf("ClearLocks = %v (err %v)", changed, err)
	}
	if ev := awaitNotice(t, sub); !strings.Contains(ev.Notice, "api released every lock") {
		t.Fatalf("notice = %q", ev.Notice)
	}
}

func TestLocksOutliveTheSession(t *testing.T) {
	m := withMCP(t)
	name := spawnFor(t, m, "api")

	if _, err := m.AddLock(name, "deploy"); err != nil {
		t.Fatalf("AddLock: %v", err)
	}
	if err := m.Stop(context.Background(), name); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	locks, err := m.Locks(name)
	if err != nil {
		t.Fatalf("Locks after a stop: %v", err)
	}
	if len(locks) != 1 || locks[0] != "deploy" {
		t.Fatalf("locks = %v, want [deploy] on the stopped session", locks)
	}

	found, err := m.FindLocked([]string{"deploy"}, false)
	if err != nil || len(found) != 1 {
		t.Fatalf("found %v (err %v), want the stopped session", found, err)
	}
	if found, err = m.FindLocked([]string{"deploy"}, true); err != nil || len(found) != 0 {
		t.Fatalf("found %v (err %v), want none when live is true", found, err)
	}
}
