package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectedText(t *testing.T) {
	lines := []string{"hello world", "second line", "third"}
	cases := []struct {
		name string
		a, b pos
		want string
	}{
		{"one line", pos{0, 0}, pos{0, 5}, "hello"},
		{"across lines", pos{0, 6}, pos{1, 6}, "world\nsecond"},
		{"reversed", pos{1, 6}, pos{0, 6}, "world\nsecond"},
		{"whole middle line", pos{0, 6}, pos{2, 0}, "world\nsecond line\n"},
		{"past the end", pos{0, 6}, pos{0, 99}, "world"},
		{"below the text", pos{2, 0}, pos{9, 0}, "third"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := selectedText(lines, c.a, c.b); got != c.want {
				t.Fatalf("selectedText = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDragInTheOutputCopies(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form = nil
	m.sel = "alpha"
	m.outputText = "hello world\nsecond line"
	m.setContent()

	var got string
	orig := copyToClipboard
	copyToClipboard = func(s string) error { got = s; return nil }
	t.Cleanup(func() { copyToClipboard = orig })

	drag := func(x, y int, action tea.MouseAction) tea.MouseMsg {
		return tea.MouseMsg{X: x, Y: y, Action: action, Button: tea.MouseButtonLeft}
	}
	m, _ = step(t, m, drag(sidebarWidth+gutterWidth, barHeight, tea.MouseActionPress))
	m, _ = step(t, m, drag(sidebarWidth+gutterWidth+6, barHeight+1, tea.MouseActionMotion))
	m, _ = step(t, m, drag(sidebarWidth+gutterWidth+6, barHeight+1, tea.MouseActionRelease))

	if want := "hello world\nsecond"; got != want {
		t.Fatalf("clipboard = %q, want %q", got, want)
	}
	if m.selection.dragging {
		t.Fatal("the drag must end on release")
	}
}

func TestAClickClearsTheSelection(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m.form = nil
	m.sel = "alpha"
	m.outputText = "hello world\nsecond line"
	m.setContent()
	m.selection = selRange{active: true, anchor: pos{0, 0}, cursor: pos{0, 5}}

	m, _ = step(t, m, tea.MouseMsg{
		X: sidebarWidth + gutterWidth, Y: barHeight, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m, _ = step(t, m, tea.MouseMsg{
		X: sidebarWidth + gutterWidth, Y: barHeight, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft,
	})
	if m.selection.active {
		t.Fatal("a click with no drag must leave no selection")
	}
}
