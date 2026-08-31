package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestThePaneMarksALineThatOpens(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	collapsed := m.wrap([]render.Line{{Class: render.ClassToolResult, Text: "← 4213 lines", Full: "a\nb"}})
	plain := m.wrap([]render.Line{{Class: render.ClassToolResult, Text: "← ok"}})
	if !strings.Contains(collapsed, "⏎") {
		t.Fatalf("a line with a body must carry the mark:\n%s", visible(collapsed))
	}
	if strings.Contains(plain, "⏎") {
		t.Fatalf("a plain line must not carry the mark:\n%s", visible(plain))
	}
}

func TestEnterOpensThePagerAndItListsEveryBody(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassText, Text: "hello"},
		{Class: render.ClassToolResult, Text: "← 3 lines", Full: "one\ntwo\nthree"},
		{Class: render.ClassToolResult, Text: "← 40 lines", Full: "big body here"},
	})
	m.focus = focusOutput
	m.prompt.Blur()

	m, _ = step(t, m, key("enter"))
	if m.pager == nil {
		t.Fatal("enter must open the pager when a result carries a body")
	}
	if len(m.pager.entries) != 2 {
		t.Fatalf("the pager lists %d results, want 2", len(m.pager.entries))
	}
	view := visible(m.pager.View(m.width, m.bodyHeight()))
	for _, want := range []string{"← 3 lines", "← 40 lines"} {
		if !strings.Contains(view, want) {
			t.Errorf("the list is missing %q:\n%s", want, view)
		}
	}
}

func TestInsideThePagerEnterPagesEscReturnsEscCloses(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassToolResult, Text: "← 3 lines", Full: "one\ntwo\nthree"},
	})
	m.focus = focusOutput
	m.prompt.Blur()

	m, _ = step(t, m, key("enter"))
	if m.pager == nil {
		t.Fatal("enter must open the pager")
	}
	m, _ = step(t, m, key("enter"))
	if !m.pager.showing {
		t.Fatal("enter inside the list must page the chosen body")
	}
	if !strings.Contains(visible(m.pager.View(m.width, m.bodyHeight())), "two") {
		t.Fatalf("the document view must show the body:\n%s", visible(m.pager.View(m.width, m.bodyHeight())))
	}
	m, _ = step(t, m, key("esc"))
	if m.pager == nil || m.pager.showing {
		t.Fatal("esc must step back to the list, not close")
	}
	m, _ = step(t, m, key("esc"))
	if m.pager != nil {
		t.Fatal("a second esc on the list must close the pager")
	}
}

func TestEnterWithNoBodyMovesToThePrompt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassText, Text: "hello"},
		{Class: render.ClassToolResult, Text: "← ok"},
	})
	m.focus = focusOutput
	m.prompt.Blur()

	m, _ = step(t, m, key("enter"))
	if m.pager != nil {
		t.Fatal("enter must not open a pager when no result carries a body")
	}
	if m.focus != focusPrompt {
		t.Fatalf("focus = %v, want the prompt", m.focus)
	}
}

func TestTheOutputKeyIAlwaysMovesToThePrompt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassToolResult, Text: "← 3 lines", Full: "one\ntwo\nthree"},
	})
	m.focus = focusOutput
	m.prompt.Blur()

	m, _ = step(t, m, key("i"))
	if m.pager != nil {
		t.Fatal("i must not open the pager")
	}
	if m.focus != focusPrompt {
		t.Fatalf("focus = %v, want the prompt", m.focus)
	}
}

func TestWhileThePagerIsOpenTheMouseDoesNothingAndCtrlCCloses(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassToolResult, Text: "← 3 lines", Full: "one\ntwo\nthree"},
	})
	m.focus = focusOutput
	m.prompt.Blur()
	m, _ = step(t, m, key("enter"))
	if m.pager == nil {
		t.Fatal("the pager must be open")
	}

	before := m.focus
	m, _ = step(t, m, tea.MouseMsg{
		X: sidebarWidth + 5, Y: 3, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if m.pager == nil {
		t.Fatal("a mouse event must not close the pager")
	}
	if m.focus != before {
		t.Fatalf("a mouse event must not change the focus while the pager is open, focus = %v", m.focus)
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.pager != nil {
		t.Fatal("ctrl+c must close the pager")
	}
}

func TestTheViewFillsTheWindowWithThePagerOpen(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 96, 22)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.appendOutput([]render.Line{
		{Class: render.ClassToolResult, Text: "← 4213 lines", Full: strings.Repeat("body line\n", 50)},
	})
	if !strings.Contains(visible(m.outputText), "⏎") {
		t.Fatalf("the pane must show the mark before the pager opens:\n%s", visible(m.outputText))
	}

	m.focus = focusOutput
	m.prompt.Blur()
	m, _ = step(t, m, key("enter"))
	if m.pager == nil {
		t.Fatal("the pager must be open")
	}

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Fatalf("the view has %d lines, want %d", len(lines), m.height)
	}
	for i, line := range lines {
		if width := lipgloss.Width(line); width > m.width {
			t.Fatalf("line %d is %d wide, want at most %d", i, width, m.width)
		}
	}
}
