package manager

import (
	"strconv"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func (b *bridge) Locks(session string) ([]string, error) { return b.m.Locks(session) }

func (b *bridge) SetLocks(labels []string, by string) ([]string, error) {
	locks, err := b.m.SetLocks(by, labels)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" holds "+lockSummary(locks), true)
	return locks, nil
}

func (b *bridge) AddLock(label, by string) ([]string, error) {
	locks, err := b.m.AddLock(by, label)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" took the lock "+label, true)
	return locks, nil
}

func (b *bridge) RemoveLock(label, by string) ([]string, error) {
	locks, err := b.m.RemoveLock(by, label)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" released the lock "+label, true)
	return locks, nil
}

func (b *bridge) ClearLocks(by string) (bool, error) {
	changed, err := b.m.ClearLocks(by)
	if err != nil {
		return false, err
	}
	if changed {
		b.m.notify(by, by+" released every lock it held", true)
	}
	return changed, nil
}

func (b *bridge) FindLocked(labels []string, live bool) ([]mcp.Session, error) {
	return b.m.FindLocked(labels, live)
}

func lockSummary(locks []string) string {
	if len(locks) == 0 {
		return "no locks"
	}
	if len(locks) == 1 {
		return "1 lock"
	}
	return strconv.Itoa(len(locks)) + " locks"
}
