package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestASentPromptAppearsAtOnce(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.SetValue("explain the loader")
	m, _ = step(t, m, key("enter"))

	if got := visible(m.output.View()); !strings.Contains(got, "explain the loader") {
		t.Fatalf("the prompt is not shown at once:\n%s", got)
	}
	if len(m.queued["alpha"]) != 1 {
		t.Fatalf("the prompt was not held for the echo: %v", m.queued["alpha"])
	}
}

func TestTheEchoDoesNotRepeatTheOptimisticPrompt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.SetValue("explain the loader")
	m, _ = step(t, m, key("enter"))

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     m.lastSeq + 1,
		Session: "alpha",
		Lines:   []render.Line{{Class: render.ClassPrompt, Text: "› explain the loader"}},
	}))

	view := visible(m.output.View())
	if got := strings.Count(view, "explain the loader"); got != 1 {
		t.Fatalf("the prompt appears %d times after the echo, want once:\n%s", got, view)
	}
	if len(m.queued["alpha"]) != 0 {
		t.Fatalf("the held copy was not dropped: %v", m.queued["alpha"])
	}
}
