package manager

import (
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

type lineBuffer struct {
	mu    sync.Mutex
	lines []render.Line
	max   int
	// seq is the bus sequence of the last event whose lines this buffer holds.
	// A replay reads it as a watermark, so the stream drops a live event whose
	// lines are already in the replay. See docs/peers.md.
	seq uint64
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
	b.appendLocked(lines)
}

// appendAt appends the lines and records the bus sequence they arrived under, so
// a later replay knows the watermark up to which the buffer is current.
func (b *lineBuffer) appendAt(lines []render.Line, seq uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(lines) > 0 {
		b.appendLocked(lines)
	}
	b.seq = seq
}

func (b *lineBuffer) appendLocked(lines []render.Line) {
	b.lines = append(b.lines, lines...)
	if len(b.lines) > b.max {
		b.lines = append([]render.Line(nil), b.lines[len(b.lines)-b.max:]...)
	}
}

// resetAt replaces the whole buffer and records the sequence, so a streamed
// session's first event of a connection sets the buffer instead of appending a
// second copy of it, and keeps a correct watermark.
func (b *lineBuffer) resetAt(lines []render.Line, seq uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resetLocked(lines)
	b.seq = seq
}

func (b *lineBuffer) resetLocked(lines []render.Line) {
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

// snapshot returns a copy of the lines and the watermark together, so a caller
// that holds the bus lock reads a buffer that is current with the events it has
// already delivered.
func (b *lineBuffer) snapshot() ([]render.Line, uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]render.Line, len(b.lines))
	copy(out, b.lines)
	return out, b.seq
}
