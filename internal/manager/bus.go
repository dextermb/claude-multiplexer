package manager

import (
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

const DefaultSubscriberBuffer = 256

type Bus struct {
	mu   sync.Mutex
	subs map[int]*Subscription
	next int
	seq  uint64
}

type Subscription struct {
	C       chan Event
	bus     *Bus
	id      int
	dropped int64
	closed  bool
}

func NewBus() *Bus {
	return &Bus{subs: make(map[int]*Subscription)}
}

func (b *Bus) Subscribe(buffer int) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.subscribeLocked(buffer)
}

// subscribeAt registers a subscription and reads the buffer's lines and
// watermark under the bus lock, so no event straddles the boundary: an event
// already in the returned lines is never also delivered live, and an event
// delivered live is never in the lines. See docs/peers.md.
func (b *Bus) subscribeAt(buf *lineBuffer, buffer int) (*Subscription, []render.Line, uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := b.subscribeLocked(buffer)
	lines, seq := buf.snapshot()
	return sub, lines, seq
}

func (b *Bus) subscribeLocked(buffer int) *Subscription {
	if buffer <= 0 {
		buffer = DefaultSubscriberBuffer
	}
	sub := &Subscription{C: make(chan Event, buffer), bus: b, id: b.next}
	b.next++
	b.subs[sub.id] = sub
	return sub
}

func (b *Bus) Publish(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	ev.Seq = b.seq
	b.deliver(ev)
}

// publishLines appends the lines to the buffer and publishes the event under one
// hold of the bus lock, so the buffer's watermark stays exact against the events
// the bus delivers. See docs/peers.md.
func (b *Bus) publishLines(buf *lineBuffer, lines []render.Line, ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	ev.Seq = b.seq
	buf.appendAt(lines, b.seq)
	b.deliver(ev)
}

// publishReset replaces the buffer and publishes the event under one hold of the
// bus lock, for a streamed session whose connection replaces its lines.
func (b *Bus) publishReset(buf *lineBuffer, lines []render.Line, ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	ev.Seq = b.seq
	buf.resetAt(lines, b.seq)
	b.deliver(ev)
}

func (b *Bus) deliver(ev Event) {
	for _, sub := range b.subs {
		select {
		case sub.C <- ev:
		default:
			select {
			case <-sub.C:
				sub.dropped++
			default:
			}
			select {
			case sub.C <- ev:
			default:
				sub.dropped++
			}
		}
	}
}

func (b *Bus) Subscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// notify publishes a change that no session event follows, so the interface
// learns of it without a timer. See docs/mcp/notices.md.
func (m *Manager) notify(name, notice string, reload bool) {
	m.bus.Publish(Event{Session: name, Notice: notice, Reload: reload})
}

func (s *Subscription) Close() {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	delete(s.bus.subs, s.id)
	close(s.C)
}

func (s *Subscription) Dropped() int64 {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	return s.dropped
}
