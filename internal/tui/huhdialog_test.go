package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// settle runs the commands a form returns and feeds their messages back, as the
// program would, until the form stops asking. A command that waits, such as a
// cursor blink, is skipped.
func settle(t *testing.T, update func(tea.Msg) (formResult, tea.Cmd), result formResult, cmd tea.Cmd) formResult {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0 && steps < 50; steps++ {
		next := queue[0]
		queue = queue[1:]
		if next == nil {
			continue
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- next() }()
		var msg tea.Msg
		select {
		case msg = <-done:
		case <-time.After(50 * time.Millisecond):
			continue
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		var more tea.Cmd
		result, more = update(msg)
		queue = append(queue, more)
	}
	return result
}

func TestTheFormThemeInvertsTheOptionUnderTheCursor(t *testing.T) {
	d := newChoiceDialog(settingModel, "alpha", "sonnet")
	view := d.View(80)
	if !strings.Contains(view, "38;2;0;0;0;48;2;255;255;255m(●) sonnet") {
		t.Fatalf("the current option must start under the cursor, inverted:\n%q", view)
	}
}
