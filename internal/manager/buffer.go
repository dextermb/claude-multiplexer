package manager

import (
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

type lineBuffer struct {
	mu    sync.Mutex
	lines []render.Line
	max   int
}

func newLineBuffer(max int) *lineBuffer {
	return &lineBuffer{max: max}
}

func (b *lineBuffer) append(lines []render.Line) {
	if len(lines) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, lines...)
	if len(b.lines) > b.max {
		b.lines = append([]render.Line(nil), b.lines[len(b.lines)-b.max:]...)
	}
}

// reset replaces the whole buffer, so a streamed session's first event of a
// connection sets the buffer instead of appending a second copy of it.
func (b *lineBuffer) reset(lines []render.Line) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append([]render.Line(nil), lines...)
	if len(b.lines) > b.max {
		b.lines = append([]render.Line(nil), b.lines[len(b.lines)-b.max:]...)
	}
}

func (b *lineBuffer) all() []render.Line {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]render.Line, len(b.lines))
	copy(out, b.lines)
	return out
}
