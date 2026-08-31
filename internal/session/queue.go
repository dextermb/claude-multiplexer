package session

import "sync"

type queue struct {
	mu    sync.Mutex
	items []string
	sig   chan struct{}
}

func newQueue() *queue {
	return &queue{sig: make(chan struct{}, 1)}
}

func (q *queue) push(item string) int {
	q.mu.Lock()
	q.items = append(q.items, item)
	n := len(q.items)
	q.mu.Unlock()
	select {
	case q.sig <- struct{}{}:
	default:
	}
	return n
}

func (q *queue) pop() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return "", false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *queue) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
