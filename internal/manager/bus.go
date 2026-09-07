package manager

import "sync"

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
	if buffer <= 0 {
		buffer = DefaultSubscriberBuffer
	}
	b.mu.Lock()
	defer b.mu.Unlock()
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
