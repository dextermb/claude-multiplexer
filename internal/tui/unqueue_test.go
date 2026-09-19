package tui

import "testing"

// queueTwoMore sends two more prompts on top of the in-flight one, so the
// backend queue holds "second" then "third" and the mirror holds three.
func queueTwoMore(t *testing.T, m Model) Model {
	t.Helper()
	m.focus = focusPrompt
	m.prompt.SetValue("second")
	m, _ = step(t, m, key("enter"))
	m.prompt.SetValue("third")
	m, _ = step(t, m, key("enter"))
	m.refresh()
	if got := len(m.queued["alpha"]); got != 3 {
		t.Fatalf("queued mirror = %d, want 3 (in-flight + two)", got)
	}
	return m
}

func TestBackspaceOnEmptyPromptRemovesTheNewestQueuedPrompt(t *testing.T) {
	m, _ := busyModel(t)
	m = queueTwoMore(t, m)

	m.prompt.Reset()
	next, cmd := step(t, m, key("backspace"))
	nm := next
	if cmd == nil {
		t.Fatal("backspace on an empty prompt with a queued prompt must return an unqueue command")
	}
	q := nm.queued["alpha"]
	if len(q) != 2 {
		t.Fatalf("queued mirror after backspace = %d, want 2", len(q))
	}
	if last := q[len(q)-1]; last != "second" {
		t.Fatalf("newest queued after backspace = %q, want \"second\"", last)
	}
	if nm.status != "removed a queued prompt" {
		t.Fatalf("status = %q, want the removed hint", nm.status)
	}
}

func TestBackspaceRemovesQueuedPromptsBottomToTop(t *testing.T) {
	m, _ := busyModel(t)
	m = queueTwoMore(t, m)

	m.prompt.Reset()
	m, _ = step(t, m, key("backspace"))
	m.refresh()
	m, _ = step(t, m, key("backspace"))
	q := m.queued["alpha"]
	if len(q) != 1 {
		t.Fatalf("queued mirror after two backspaces = %d, want 1 (only the in-flight)", len(q))
	}
	if q[0] != "a long task" {
		t.Fatalf("remaining prompt = %q, want the in-flight one", q[0])
	}
}

func TestBackspaceLeavesTheInFlightPrompt(t *testing.T) {
	m, _ := busyModel(t)
	// The only prompt is in-flight, so the backend queue is empty.
	m.prompt.Reset()
	next, cmd := step(t, m, key("backspace"))
	nm := next
	if cmd != nil {
		t.Fatal("backspace must not act when only the in-flight prompt remains")
	}
	if got := len(nm.queued["alpha"]); got != 1 {
		t.Fatalf("queued mirror = %d, want 1 (the in-flight prompt is kept)", got)
	}
	if nm.status == "removed a queued prompt" {
		t.Fatal("the removed hint must not show when nothing was removed")
	}
}

func TestBackspaceWithTextEditsThePromptNotTheQueue(t *testing.T) {
	m, _ := busyModel(t)
	m = queueTwoMore(t, m)

	m.prompt.SetValue("x")
	next, _ := step(t, m, key("backspace"))
	nm := next
	if got := len(nm.queued["alpha"]); got != 3 {
		t.Fatalf("queued mirror = %d, want 3 (backspace with text must not unqueue)", got)
	}
	if nm.status == "removed a queued prompt" {
		t.Fatal("backspace with text must not remove a queued prompt")
	}
}
