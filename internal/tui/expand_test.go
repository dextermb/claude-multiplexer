package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func body(n int) string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("line %d", i+1)
	}
	return strings.Join(rows, "\n")
}

func outputModel(t *testing.T) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusOutput
	m.prompt.Blur()
	return m
}

func TestBlocksGroupAPromptItsCommandAndItsOutput(t *testing.T) {
	var lines []render.Line
	lines = append(lines, render.PromptLines("one\ntwo")...)
	lines = append(lines, render.BashLines("ls", "a\nb\nc", nil)...)
	lines = append(lines, render.Line{Class: render.ClassToolResult, Text: "← a\nb"})

	got := blocks(lines)
	want := []block{{0, 2}, {2, 3}, {3, 6}, {6, 7}}
	if len(got) != len(want) {
		t.Fatalf("blocks = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("block %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestABlockOfTwentyLinesDrawsNoMarker(t *testing.T) {
	m := outputModel(t)
	cap := m.capFor(render.ClassToolResult)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(cap)}})

	if len(m.capped) != 0 {
		t.Fatalf("a block of %d lines must not be capped", cap)
	}
	view := visible(m.outputText)
	if strings.Contains(view, "⋯") {
		t.Fatalf("the pane must draw no marker:\n%s", view)
	}
	if !strings.Contains(view, fmt.Sprintf("line %d", cap)) {
		t.Fatalf("the pane must draw every line:\n%s", view)
	}
}

func TestABlockOfMoreThanTwentyLinesShowsTwentyAndAMarker(t *testing.T) {
	m := outputModel(t)
	cap := m.capFor(render.ClassToolResult)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(cap + 1)}})

	view := visible(m.outputText)
	if !strings.Contains(view, "⋯ 1 more line") {
		t.Fatalf("the marker must count the one line it hides:\n%s", view)
	}
	if strings.Contains(view, fmt.Sprintf("line %d", cap+1)) {
		t.Fatalf("the pane must hide the last line:\n%s", view)
	}
	if rows := rowCount(m.outputText); rows != cap+1 {
		t.Fatalf("the block draws %d rows, want %d", rows, cap+1)
	}
}

func TestEnterOpensTheBlockUnderTheCursorAndClosesItAgain(t *testing.T) {
	m := outputModel(t)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(50)}})
	if m.blockCursor < 0 {
		t.Fatal("the cursor must sit on the capped block")
	}

	m, _ = step(t, m, key("enter"))
	view := visible(m.outputText)
	if !strings.Contains(view, "line 50") {
		t.Fatalf("an open block draws every line:\n%s", view)
	}
	if !strings.Contains(view, "⋯ show less") {
		t.Fatalf("an open block keeps a marker that closes it:\n%s", view)
	}

	m, _ = step(t, m, key("enter"))
	view = visible(m.outputText)
	if strings.Contains(view, "line 50") {
		t.Fatalf("enter must close the block again:\n%s", view)
	}
	if !strings.Contains(view, "⋯ 30 more lines") {
		t.Fatalf("the marker must return:\n%s", view)
	}
}

func TestTheBracketKeysWalkTheCappedBlocksAndStopAtTheEnds(t *testing.T) {
	m := outputModel(t)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(30)}})
	m.appendOutput([]render.Line{{Class: render.ClassText, Text: "a word"}})
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(40)}})

	if len(m.capped) != 2 {
		t.Fatalf("capped = %v, want the two large blocks", m.capped)
	}
	if m.blockCursor != m.capped[1] {
		t.Fatalf("the cursor starts on the newest capped block, got %d", m.blockCursor)
	}

	m, _ = step(t, m, key("["))
	if m.blockCursor != m.capped[0] {
		t.Fatalf("[ must move to the block before, got %d", m.blockCursor)
	}
	m, _ = step(t, m, key("["))
	if m.blockCursor != m.capped[0] {
		t.Fatalf("[ must stop at the first block, got %d", m.blockCursor)
	}
	m, _ = step(t, m, key("]"))
	if m.blockCursor != m.capped[1] {
		t.Fatalf("] must move to the next block, got %d", m.blockCursor)
	}
	m, _ = step(t, m, key("]"))
	if m.blockCursor != m.capped[1] {
		t.Fatalf("] must stop at the last block, got %d", m.blockCursor)
	}
}

func TestTheCursorMarksOnlyTheBlockItIsOn(t *testing.T) {
	m := outputModel(t)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(30)}})
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(40)}})

	m, _ = step(t, m, key("["))
	rows := strings.Split(visible(m.outputText), "\n")
	first, second := rows[m.markerAt[m.capped[0]]], rows[m.markerAt[m.capped[1]]]
	if !strings.HasPrefix(first, "▸ ") {
		t.Fatalf("the marker under the cursor carries the mark, got %q", first)
	}
	if strings.HasPrefix(second, "▸ ") {
		t.Fatalf("no other marker carries the mark, got %q", second)
	}
}

func TestTheCursorReturnsToTheNewestBlockAtTheEndOfATurn(t *testing.T) {
	m := outputModel(t)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(30)}})
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(40)}})
	m, _ = step(t, m, key("["))
	if m.blockCursor != m.capped[0] {
		t.Fatalf("the cursor must be on the older block, got %d", m.blockCursor)
	}

	row := m.rows[m.selIndex()]
	row.state = session.StateBusy
	m.rows[m.selIndex()] = row
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 1, Session: m.sel}))

	if m.blockCursor != m.capped[len(m.capped)-1] {
		t.Fatalf("the end of a turn returns the cursor to the newest block, got %d", m.blockCursor)
	}
}

func TestAClickOnAMarkerOpensThatBlock(t *testing.T) {
	m := outputModel(t)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(40)}})
	index := m.capped[0]
	m.output.SetYOffset(0)

	m, _ = step(t, m, tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      sidebarWidth + gutterWidth + 2,
		Y:      barHeight + m.markerAt[index] - m.output.YOffset,
	})
	if !m.expanded[index] {
		t.Fatal("a click on the marker row must open the block")
	}
}

func TestTheViewFillsTheWindowWithABlockOpenAndClosed(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 96, 22)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusOutput
	m.prompt.Blur()
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(50)}})

	for _, open := range []bool{false, true} {
		if open {
			m, _ = step(t, m, key("enter"))
		}
		lines := strings.Split(m.View(), "\n")
		if len(lines) != m.height {
			t.Fatalf("open=%v: the view has %d lines, want %d", open, len(lines), m.height)
		}
		for i, line := range lines {
			if width := lipgloss.Width(line); width > m.width {
				t.Fatalf("open=%v: line %d is %d wide, want at most %d", open, i, width, m.width)
			}
		}
	}
}

func TestTheSettingsChangeTheCap(t *testing.T) {
	m := outputModel(t)
	if err := m.mgr.AppendLines(m.sel, []render.Line{{Class: render.ClassToolResult, Text: body(30)}}); err != nil {
		t.Fatalf("AppendLines: %v", err)
	}
	m.rebuildOutput()
	if len(m.capped) != 1 {
		t.Fatalf("capped = %v, want the one large block", m.capped)
	}

	m, _ = step(t, m, settingsMsg{caps: capsAll(5)})
	view := visible(m.outputText)
	if !strings.Contains(view, "⋯ 25 more lines") {
		t.Fatalf("a cap of 5 hides 25 rows:\n%s", view)
	}
	if strings.Contains(view, "line 6") {
		t.Fatalf("a cap of 5 draws five rows:\n%s", view)
	}

	m, _ = step(t, m, settingsMsg{caps: capsAll(0)})
	view = visible(m.outputText)
	if len(m.capped) != 0 || strings.Contains(view, "⋯") {
		t.Fatalf("a cap of 0 caps nothing:\n%s", view)
	}
	if !strings.Contains(view, "line 30") {
		t.Fatalf("a cap of 0 draws every row:\n%s", view)
	}
}

func TestTheModelStartsAtTheDefaultCap(t *testing.T) {
	m, _ := newTestModel(t, "")
	if cap := m.capFor(render.ClassToolResult); cap != config.DefaultBlockCap {
		t.Fatalf("cap = %d, want %d", cap, config.DefaultBlockCap)
	}
}

// capsAll resolves one row count for every bucket, as a settings change that
// sets the global default does.
func capsAll(n int) map[string]int {
	return config.ResolveBlockCaps(config.Config{BlockCap: &n})
}

func TestAPerTypeCapDiffersByBucket(t *testing.T) {
	m := outputModel(t)
	if err := m.mgr.AppendLines(m.sel, []render.Line{{Class: render.ClassText, Text: body(30)}}); err != nil {
		t.Fatalf("AppendLines: %v", err)
	}
	if err := m.mgr.AppendLines(m.sel, []render.Line{{Class: render.ClassToolResult, Text: body(30)}}); err != nil {
		t.Fatalf("AppendLines: %v", err)
	}
	three := 3
	caps := config.ResolveBlockCaps(config.Config{BlockCaps: map[string]*int{
		config.BucketMessage: nil,
		config.BucketTool:    &three,
	}})

	m, _ = step(t, m, settingsMsg{caps: caps})
	view := visible(m.outputText)
	if !strings.Contains(view, "line 30") {
		t.Fatalf("a message never caps, so it draws every row:\n%s", view)
	}
	if !strings.Contains(view, "⋯ 27 more lines") {
		t.Fatalf("a tool result caps at three, hiding 27:\n%s", view)
	}
}
