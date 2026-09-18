package manager

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// ErrNoLock is the failure when a lock label is empty once it is trimmed.
var ErrNoLock = errors.New("manager: a lock needs a label")

// Locks reads the labels a session holds, in the order it took them. It reads a
// live session from memory, and a stopped one from its metadata on disk. See
// docs/mcp/tools/locks.md.
func (m *Manager) Locks(name string) ([]string, error) {
	meta, err := m.anyMeta(name)
	if err != nil {
		return nil, err
	}
	return meta.Locks, nil
}

// SetLocks replaces the whole list of labels a session holds. The result keeps
// the given order, without a repeat.
func (m *Manager) SetLocks(name string, labels []string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	locks, err := cleanLocks(labels)
	if err != nil {
		return nil, err
	}
	return m.writeLocks(item, func([]string) []string { return locks })
}

// AddLock takes one lock. A label the session already holds is left as it is.
func (m *Manager) AddLock(name, label string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	lock, err := cleanLock(label)
	if err != nil {
		return nil, err
	}
	return m.writeLocks(item, func(held []string) []string {
		return appendUnique(held, lock)
	})
}

// RemoveLock releases one lock, and returns the labels the session still holds.
func (m *Manager) RemoveLock(name, label string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	lock, err := cleanLock(label)
	if err != nil {
		return nil, err
	}
	return m.writeLocks(item, func(held []string) []string {
		var kept []string
		for _, label := range held {
			if label != lock {
				kept = append(kept, label)
			}
		}
		return kept
	})
}

// ClearLocks releases every lock of a session, and reports whether it held any.
func (m *Manager) ClearLocks(name string) (bool, error) {
	item, err := m.entry(name)
	if err != nil {
		return false, err
	}
	if len(item.metaCopy().Locks) == 0 {
		return false, nil
	}
	if _, err := m.writeLocks(item, func([]string) []string { return nil }); err != nil {
		return false, err
	}
	return true, nil
}

// FindLocked reports the sessions that hold every named label. A session that
// runs on a peer carries no metadata here, so it never matches.
func (m *Manager) FindLocked(labels []string, live bool) ([]mcp.Session, error) {
	wanted, err := cleanLocks(labels)
	if err != nil {
		return nil, err
	}
	if len(wanted) == 0 {
		return nil, ErrNoLock
	}
	var out []mcp.Session
	for _, item := range m.List() {
		if live && !running(item) {
			continue
		}
		if holdsAll(item.Locks, wanted) {
			out = append(out, item)
		}
	}
	return out, nil
}

// running reports whether a row is a session the multiplexer runs now. A row
// that exited stays in the live table until it is evicted, so the state word
// decides, not the Live flag alone.
func running(item mcp.Session) bool {
	return item.Live && session.ParseState(item.State).Live()
}

func (m *Manager) writeLocks(item *entry, next func(held []string) []string) ([]string, error) {
	var locks []string
	if _, err := item.mutateMeta(func(meta *Meta) error {
		locks = next(meta.Locks)
		meta.Locks = locks
		return nil
	}); err != nil {
		return nil, err
	}
	return locks, nil
}

// anyMeta reads the metadata of a session whether it runs now or not.
func (m *Manager) anyMeta(name string) (Meta, error) {
	if item, err := m.entry(name); err == nil {
		return item.metaCopy(), nil
	}
	for _, meta := range m.Stored() {
		if meta.Name == name {
			return meta, nil
		}
	}
	return Meta{}, fmt.Errorf("%w: %s", ErrUnknownSession, name)
}

func holdsAll(held, wanted []string) bool {
	for _, lock := range wanted {
		if !slices.Contains(held, lock) {
			return false
		}
	}
	return true
}

func cleanLock(label string) (string, error) {
	lock := strings.TrimSpace(label)
	if lock == "" {
		return "", ErrNoLock
	}
	return lock, nil
}

func cleanLocks(labels []string) ([]string, error) {
	var out []string
	for _, label := range labels {
		lock, err := cleanLock(label)
		if err != nil {
			return nil, err
		}
		out = appendUnique(out, lock)
	}
	return out, nil
}
