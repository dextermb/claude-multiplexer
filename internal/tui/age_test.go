package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestAgeStringFormatsRelativeAge(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{10 * time.Second, "now"},
		{59 * time.Second, "now"},
		{time.Minute, "1m"},
		{2 * time.Minute, "2m"},
		{59 * time.Minute, "59m"},
		{time.Hour, "1h"},
		{23 * time.Hour, "23h"},
		{24 * time.Hour, "1d"},
		{72 * time.Hour, "3d"},
	}
	for _, c := range cases {
		if got := ageString(now, now.Add(-c.ago)); got != c.want {
			t.Errorf("ageString(%s ago) = %q, want %q", c.ago, got, c.want)
		}
	}
}

func TestOutputPaneShowsBlockAgeAtRightEdge(t *testing.T) {
	m := outputModel(t)
	m.showAge = true
	at := time.Now().Add(-2 * time.Minute)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: "one short line", At: at}})

	row := firstRow(m.outputText)
	if !strings.HasSuffix(strings.TrimRight(ansi.Strip(row), " "), "2m") {
		t.Errorf("row does not end with the age: %q", ansi.Strip(row))
	}
	if w := lipgloss.Width(row); w != m.outputWidth() {
		t.Errorf("row width = %d, want the full pane width %d", w, m.outputWidth())
	}
}

func TestMultiLineBlockShowsAgeOnFirstRowOnly(t *testing.T) {
	m := outputModel(t)
	m.showAge = true
	at := time.Now().Add(-time.Hour)
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: body(4), At: at}})

	rows := strings.Split(m.outputText, "\n")
	if len(rows) < 2 {
		t.Fatalf("want a multi-line block, got %d rows", len(rows))
	}
	if !strings.HasSuffix(strings.TrimRight(ansi.Strip(rows[0]), " "), "1h") {
		t.Errorf("first row does not carry the age: %q", ansi.Strip(rows[0]))
	}
	for i, row := range rows[1:] {
		if strings.Contains(ansi.Strip(row), "1h") {
			t.Errorf("row %d carries the age but must not: %q", i+1, ansi.Strip(row))
		}
	}
}

func TestAgeColumnOffLeavesThePaneUnchanged(t *testing.T) {
	at := time.Now().Add(-3 * time.Minute)
	line := render.Line{Class: render.ClassToolResult, Text: "one short line", At: at}

	off := outputModel(t)
	off.appendOutput([]render.Line{line})

	on := outputModel(t)
	on.showAge = true
	on.appendOutput([]render.Line{line})

	if strings.Contains(off.outputText, "3m") {
		t.Errorf("age must not show with the column off: %q", off.outputText)
	}
	if !strings.Contains(on.outputText, "3m") {
		t.Errorf("age must show with the column on: %q", on.outputText)
	}
}

func TestAgeTickRecomputesTheAge(t *testing.T) {
	m := outputModel(t)
	m.showAge = true
	m.appendOutput([]render.Line{{Class: render.ClassToolResult, Text: "one short line", At: time.Now().Add(-5 * time.Minute)}})
	if !strings.Contains(m.outputText, "5m") {
		t.Fatalf("want the age baked in on first draw: %q", ansi.Strip(m.outputText))
	}

	m.shownLines[0].At = time.Now()
	tm, _ := m.handleAgeTick()
	m = tm.(Model)

	if strings.Contains(ansi.Strip(m.outputText), "5m") {
		t.Errorf("the tick did not recompute the age: %q", ansi.Strip(m.outputText))
	}
	if !strings.Contains(ansi.Strip(m.outputText), "now") {
		t.Errorf("want the recomputed age: %q", ansi.Strip(m.outputText))
	}
}

func firstRow(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	return text
}
