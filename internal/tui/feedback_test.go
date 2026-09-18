package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
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
		Seq:        m.lastSeq + 1,
		Session:    "alpha",
		Lines:      []render.Line{{Class: render.ClassPrompt, Text: "› explain the loader"}},
		PromptEcho: true,
	}))

	view := visible(m.output.View())
	if got := strings.Count(view, "explain the loader"); got != 1 {
		t.Fatalf("the prompt appears %d times after the echo, want once:\n%s", got, view)
	}
	if len(m.queued["alpha"]) != 0 {
		t.Fatalf("the held copy was not dropped: %v", m.queued["alpha"])
	}
}

// A slash command replays as a » /name meta line, not a violet prompt line, so
// the held copy must drop on the PromptEcho flag and not on a prompt line. See
// docs/tui/output.md.
func TestASlashCommandEchoDropsTheHeldCopy(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusPrompt
	m.prompt.SetValue("/xyzzy")
	m, _ = step(t, m, key("enter"))
	if len(m.queued["alpha"]) != 1 {
		t.Fatalf("the command was not held for the echo: %v", m.queued["alpha"])
	}

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:        m.lastSeq + 1,
		Session:    "alpha",
		Lines:      []render.Line{{Class: render.ClassMeta, Text: "» /xyzzy"}},
		PromptEcho: true,
	}))

	if len(m.queued["alpha"]) != 0 {
		t.Fatalf("the held copy of the command was not dropped: %v", m.queued["alpha"])
	}
}

// The spinner follows the busy state, so it clears when the session goes idle,
// even if a held copy lingers because no echo dropped it. See docs/tui/output.md.
func TestThinkingFollowsBusyNotQueue(t *testing.T) {
	m := Model{
		sel:      "alpha",
		partials: map[string]string{},
		queued:   map[string][]string{"alpha": {"/xyzzy"}},
		rows:     []row{{name: "alpha", live: true, state: session.StateBusy}},
	}
	if !m.thinkingSelected() {
		t.Fatal("the spinner must show while the session is busy")
	}
	m.rows[0].state = session.StateIdle
	if m.thinkingSelected() {
		t.Fatal("the spinner must clear when the session is idle, even with a held copy")
	}
}
