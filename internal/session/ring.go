package session

import "sync"

type ring struct {
	mu    sync.Mutex
	items []string
	max   int
}

func newRing(max int) *ring {
	return &ring{max: max}
}

func (r *ring) add(item string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, item)
	if len(r.items) > r.max {
		r.items = r.items[len(r.items)-r.max:]
	}
}

func (r *ring) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.items))
	copy(out, r.items)
	return out
}
