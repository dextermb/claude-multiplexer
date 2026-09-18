package manager

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
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

	skill render.SkillTracker

	metaMu sync.Mutex

	costMu  sync.Mutex
	written totals

	partialMu sync.Mutex
	partial   strings.Builder

	todoMu sync.Mutex
	tasks  protocol.TaskTracker

	snapMu sync.Mutex
	snap   session.Snapshot

	idleMu          sync.Mutex
	stopWhenIdle    bool
	archiveWhenIdle bool
	idleFired       bool

	ctxMu     sync.Mutex
	ctxLast   int
	ctxWarned bool
	ctxActed  bool
	ctxHeld   bool
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

// mutateMeta applies fn to the record under metaMu, persists it once, and
// returns the new copy. It holds metaMu across the write, so the change is
// atomic against the pump. See docs/manager.md.
func (e *entry) mutateMeta(fn func(*Meta) error) (Meta, error) {
	e.metaMu.Lock()
	defer e.metaMu.Unlock()
	next := e.meta
	if err := fn(&next); err != nil {
		return Meta{}, err
	}
	if err := writeMeta(e.path, next); err != nil {
		return Meta{}, err
	}
	e.meta = next
	return next, nil
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
	live.CacheReadTokens = cached.CacheReadTokens
	live.CacheWriteTokens = cached.CacheWriteTokens
	live.OutputTokens = cached.OutputTokens
	live.ContextTokens = cached.ContextTokens
	live.LastDuration = cached.LastDuration
	return live
}

// total adds the counters of the earlier runs to the counters of the running
// child, so every reader sees the lifetime figures. The pump is the one place
// that calls it. See docs/manager.md.
func (e *entry) total(snap session.Snapshot) session.Snapshot {
	snap.Turns += e.base.turns
	snap.Cost += e.base.cost
	snap.InputTokens += e.base.input
	snap.CacheReadTokens += e.base.cacheRead
	snap.CacheWriteTokens += e.base.cacheWrite
	snap.OutputTokens += e.base.output
	return snap
}

func (e *entry) todoList() []protocol.Todo {
	e.todoMu.Lock()
	defer e.todoMu.Unlock()
	return e.tasks.List()
}

type totals struct {
	turns      int
	cost       float64
	input      int
	cacheRead  int
	cacheWrite int
	output     int
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
