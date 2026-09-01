package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func promptModel(t *testing.T) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m.form = nil
	m = spawn(t, m, mgr, "api", t.TempDir())
	if m.focus != focusPrompt {
		t.Fatal("the prompt must have the focus")
	}
	return m
}

func newline(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlJ})
	return m
}

func TestThePromptBoxStartsAtTwoRows(t *testing.T) {
	m := promptModel(t)
	if got := m.promptHeight(); got != 3 {
		t.Fatalf("prompt block is %d rows, want 3", got)
	}
}

func TestThePromptBoxGrowsWithNewLinesAndStopsAtFour(t *testing.T) {
	m := promptModel(t)
	body := m.bodyHeight()

	m, _ = step(t, m, key("one"))
	m = newline(t, m)
	m, _ = step(t, m, key("two"))
	if got := m.promptHeight(); got != 3 {
		t.Fatalf("two lines gave %d rows, want 3", got)
	}

	m = newline(t, m)
	m, _ = step(t, m, key("three"))
	if got := m.promptHeight(); got != 4 {
		t.Fatalf("three lines gave %d rows, want 4", got)
	}
	if got := m.bodyHeight(); got != body-1 {
		t.Fatalf("body is %d rows, want %d", got, body-1)
	}

	m = newline(t, m)
	m, _ = step(t, m, key("four"))
	if got := m.promptHeight(); got != 5 {
		t.Fatalf("four lines gave %d rows, want 5", got)
	}

	m = newline(t, m)
	m, _ = step(t, m, key("five"))
	if got := m.promptHeight(); got != 5 {
		t.Fatalf("five lines gave %d rows, want 5, the cap", got)
	}
}

func TestTheOutputPaneGivesBackTheRowsThePromptTakes(t *testing.T) {
	m := promptModel(t)
	height := m.output.Height

	m = newline(t, m)
	m = newline(t, m)
	if got := m.output.Height; got != height-1 {
		t.Fatalf("the output pane is %d rows, want %d", got, height-1)
	}
	if got := lines(m.View()); got != 30 {
		t.Fatalf("the view is %d rows, want 30", got)
	}
}

func TestALongLineWrapsAndGrowsThePromptBox(t *testing.T) {
	m := promptModel(t)
	m, _ = step(t, m, key(strings.Repeat("word ", 80)))
	if got := m.promptHeight(); got != 5 {
		t.Fatalf("a wrapped line gave %d rows, want 5", got)
	}
}

func TestThePromptBoxShrinksAgain(t *testing.T) {
	m := promptModel(t)
	m = newline(t, m)
	m = newline(t, m)
	m = newline(t, m)
	if got := m.promptHeight(); got != 5 {
		t.Fatalf("four lines gave %d rows, want 5", got)
	}
	m.prompt.SetValue("")
	m, _ = step(t, m, key("a"))
	if got := m.promptHeight(); got != 3 {
		t.Fatalf("an emptied prompt gave %d rows, want 3", got)
	}
}

func TestWrappedRowsCountsTheDisplayRows(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		width int
		want  int
	}{
		{"empty", "", 20, 1},
		{"one short line", "hello", 20, 1},
		{"two lines", "hello\nworld", 20, 2},
		{"a word wraps whole", "hello world again", 12, 2},
		{"a word longer than the width breaks", strings.Repeat("x", 45), 20, 3},
		{"the count stops at the cap", strings.Repeat("a\n", 20), 20, promptRowsMax},
		{"no width", "hello", 0, promptRowsMin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wrappedRows(tc.text, tc.width); got != tc.want {
				t.Fatalf("wrappedRows(%q, %d) = %d, want %d", tc.text, tc.width, got, tc.want)
			}
		})
	}
}

func lines(view string) int {
	return len(strings.Split(view, "\n"))
}
