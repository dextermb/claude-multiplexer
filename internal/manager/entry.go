package manager

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

type entry struct {
	sess    *session.Session
	lines   *lineBuffer
	meta    Meta
	path    string
	base    totals
	token   string
	control bool

	metaMu sync.Mutex

	partialMu sync.Mutex
	partial   strings.Builder

	todoMu sync.Mutex
	tasks  protocol.TaskTracker

	snapMu sync.Mutex
	snap   session.Snapshot
}

// metaCopy and setMeta guard the record of a live session, because the pump
// goroutine writes it while the interface and the tools read it.
func (e *entry) metaCopy() Meta {
	e.metaMu.Lock()
	defer e.metaMu.Unlock()
	return e.meta
}

func (e *entry) setMeta(meta Meta) {
	e.metaMu.Lock()
	defer e.metaMu.Unlock()
	e.meta = meta
}

func (e *entry) partialText() string {
	e.partialMu.Lock()
	defer e.partialMu.Unlock()
	return e.partial.String()
}

func (e *entry) setSnapshot(snap session.Snapshot) {
	e.snapMu.Lock()
	defer e.snapMu.Unlock()
	e.snap = snap
}

// view returns the live session snapshot, but with the totals the stream feeds
// taken from the last snapshot the pump saw. So the turn count never leads the
// buffered lines, and the title, the queue, and the state stay live. See
// docs/manager.md.
func (e *entry) view() session.Snapshot {
	live := e.sess.Snapshot()
	e.snapMu.Lock()
	cached := e.snap
	e.snapMu.Unlock()
	live.Turns = cached.Turns
	live.Cost = cached.Cost
	live.InputTokens = cached.InputTokens
	live.OutputTokens = cached.OutputTokens
	live.ContextTokens = cached.ContextTokens
	live.LastDuration = cached.LastDuration
	return live
}

func (e *entry) todoList() []protocol.Todo {
	e.todoMu.Lock()
	defer e.todoMu.Unlock()
	return e.tasks.List()
}

type totals struct {
	turns  int
	cost   float64
	input  int
	output int
}

func (m *Manager) entry(name string) (*entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.entries[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSession, name)
	}
	return item, nil
}
