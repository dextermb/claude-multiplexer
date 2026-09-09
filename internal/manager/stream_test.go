package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// TestApplyRemoteResetMarksReplace pins the reset contract: the first event of a
// connection replaces the buffer and marks the event Replace, so a viewer
// rebuilds instead of appending the whole buffer a second time. A later event
// appends and does not mark Replace. See docs/peers.md.
func TestApplyRemoteResetMarksReplace(t *testing.T) {
	m := newTestManager(t)
	sub := m.Subscribe(16)
	defer sub.Close()

	re := &remoteEntry{localName: "r", lines: newLineBuffer(1000), cancel: func() {}}
	m.mu.Lock()
	m.remotes["r"] = re
	m.remoteOrder = append(m.remoteOrder, "r")
	m.mu.Unlock()

	full := []render.Line{{Text: "a"}, {Text: "b"}}
	m.applyRemote(re, wire.Event{Session: "remote", Lines: full}, true)

	if got := re.lines.all(); len(got) != 2 {
		t.Fatalf("buffer has %d lines after the reset, want 2", len(got))
	}
	ev := <-sub.C
	if !ev.Replace {
		t.Fatal("the reset event must set Replace")
	}
	if len(ev.Lines) != 2 {
		t.Fatalf("the reset event carries %d lines, want the whole buffer", len(ev.Lines))
	}

	m.applyRemote(re, wire.Event{Session: "remote", Lines: []render.Line{{Text: "c"}}}, false)

	if got := re.lines.all(); len(got) != 3 {
		t.Fatalf("buffer has %d lines after the append, want 3", len(got))
	}
	ev = <-sub.C
	if ev.Replace {
		t.Fatal("an append event must not set Replace")
	}
}

// TestSubscribeAtSplitsReplayFromLive pins the boundary contract: an event
// already folded into the buffer is in the replay and never in the live
// channel, and a later event is live only, with a sequence past the watermark.
func TestSubscribeAtSplitsReplayFromLive(t *testing.T) {
	bus := NewBus()
	buf := newLineBuffer(100)

	a := render.Line{Text: "a"}
	bus.publishLines(buf, []render.Line{a}, Event{Session: "s", Lines: []render.Line{a}})

	sub, lines, watermark := bus.subscribeAt(buf, 8)
	defer sub.Close()

	if len(lines) != 1 || lines[0].Text != "a" {
		t.Fatalf("replay lines = %v, want [a]", lines)
	}
	if watermark == 0 {
		t.Fatal("watermark not set from the buffer")
	}
	if len(sub.C) != 0 {
		t.Fatal("the folded event leaked into the live channel")
	}

	b := render.Line{Text: "b"}
	bus.publishLines(buf, []render.Line{b}, Event{Session: "s", Lines: []render.Line{b}})

	ev := <-sub.C
	if ev.Seq <= watermark {
		t.Fatalf("live event seq %d is not past the watermark %d", ev.Seq, watermark)
	}
	if len(ev.Lines) != 1 || ev.Lines[0].Text != "b" {
		t.Fatalf("live lines = %v, want [b]", ev.Lines)
	}
}

// TestStreamDoesNotDoubleLinesAtTheReplayBoundary streams a session while a
// publisher fills its buffer, so the replay and the live tail race. Every line
// must arrive exactly once, so the boundary never doubles the output. See
// docs/peers.md.
func TestStreamDoesNotDoubleLinesAtTheReplayBoundary(t *testing.T) {
	m := newTestManager(t)

	const iterations = 100
	const linesPerRun = 20

	for iter := 0; iter < iterations; iter++ {
		name := fmt.Sprintf("r%d", iter)
		re := &remoteEntry{localName: name, lines: newLineBuffer(4096), cancel: func() {}}
		m.mu.Lock()
		m.remotes[name] = re
		m.remoteOrder = append(m.remoteOrder, name)
		m.mu.Unlock()

		start := make(chan struct{})
		go func() {
			<-start
			for i := 0; i < linesPerRun; i++ {
				line := render.Line{Text: fmt.Sprintf("%s-%d", name, i)}
				m.bus.publishLines(re.lines, []render.Line{line},
					Event{Session: name, Lines: []render.Line{line}})
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		close(start)
		ch := m.streamSession(ctx, name)

		seen := map[string]int{}
		count := 0
		timeout := time.After(2 * time.Second)
	collect:
		for count < linesPerRun {
			select {
			case ev, ok := <-ch:
				if !ok {
					break collect
				}
				for _, line := range ev.Lines {
					seen[line.Text]++
					count++
				}
			case <-timeout:
				break collect
			}
		}
		cancel()

		for i := 0; i < linesPerRun; i++ {
			key := fmt.Sprintf("%s-%d", name, i)
			if seen[key] != 1 {
				t.Fatalf("iteration %d: line %s seen %d times, want exactly 1",
					iter, key, seen[key])
			}
		}
	}
}
