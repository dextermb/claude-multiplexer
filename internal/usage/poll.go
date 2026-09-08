package usage

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Fetch makes one request that returns the unified rate-limit headers. It is
// injected, so the poller does not hold the endpoint or the credential. A read
// of usage costs a little of the account's usage, so the poller calls Fetch on
// an interval and caches the result. See docs/peers.md.
type Fetch func(context.Context) (http.Header, error)

// Poller keeps a cached Usage, and refreshes it on an interval. A read never
// makes a network call.
type Poller struct {
	fetch    Fetch
	interval time.Duration
	now      func() time.Time
	onUpdate func(Usage)

	mu   sync.Mutex
	last Usage
}

// NewPoller builds a poller. A zero interval takes the default.
func NewPoller(fetch Fetch, interval time.Duration) *Poller {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Poller{fetch: fetch, interval: interval, now: time.Now}
}

// OnUpdate registers a callback the poller runs after each refresh, so the
// reserve gate re-evaluates on every poll. See docs/peers.md.
func (p *Poller) OnUpdate(fn func(Usage)) { p.onUpdate = fn }

// Usage returns the last cached read. It is safe to call from any goroutine.
func (p *Poller) Usage() Usage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last
}

// Refresh runs one Fetch and updates the cache. A failed fetch keeps the
// last-known windows and records the error, so a read never loses the last good
// values to a transient failure.
func (p *Poller) Refresh(ctx context.Context) Usage {
	header, err := p.fetch(ctx)
	p.mu.Lock()
	if err != nil {
		p.last.OK = false
		p.last.Error = err.Error()
		p.last.FetchedAt = p.now()
		result := p.last
		p.mu.Unlock()
		p.notify(result)
		return result
	}
	next := Parse(header)
	next.FetchedAt = p.now()
	p.last = next
	p.mu.Unlock()
	p.notify(next)
	return next
}

// notify runs the update callback outside the lock, so it can read the cache.
func (p *Poller) notify(u Usage) {
	if p.onUpdate != nil {
		p.onUpdate(u)
	}
}

// Run refreshes at once, and then on the interval, until the context is done.
func (p *Poller) Run(ctx context.Context) {
	p.Refresh(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.Refresh(ctx)
		}
	}
}
