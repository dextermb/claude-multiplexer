package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

// TestReplaceEventRebuildsInsteadOfAppending guards the streamed-session reset:
// the first event of a pump connection carries the whole buffer with Replace
// set, so the viewer rebuilds. It must not append the buffer a second time. See
// docs/peers.md.
func TestReplaceEventRebuildsInsteadOfAppending(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	const mark = "ZZMARKZZ"
	if err := mgr.AppendLines("alpha", []render.Line{{Class: render.ClassText, Text: mark}}); err != nil {
		t.Fatalf("AppendLines: %v", err)
	}

	m.lastSeq = 1
	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     2,
		Session: "alpha",
		Lines:   []render.Line{{Class: render.ClassText, Text: mark}},
	}))
	if got := strings.Count(visible(m.content), mark); got != 1 {
		t.Fatalf("after the delta, %q appears %d times, want 1", mark, got)
	}

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     3,
		Session: "alpha",
		Replace: true,
		Lines:   mgr.Lines("alpha"),
	}))
	if got := strings.Count(visible(m.content), mark); got != 1 {
		t.Fatalf("after the replace, %q appears %d times, want 1", mark, got)
	}
}
