package tui

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestCollectExpandablesKeepsOnlyLinesWithABody(t *testing.T) {
	lines := []render.Line{
		{Class: render.ClassText, Text: "hello"},
		{Class: render.ClassToolResult, Text: "← 3 lines", Full: "one\ntwo\nthree"},
		{Class: render.ClassToolResult, Text: "← ok"},
	}
	got := collectExpandables(lines)
	if len(got) != 1 {
		t.Fatalf("want 1 expandable, got %d", len(got))
	}
	if got[0].label != "← 3 lines" || got[0].body != "one\ntwo\nthree" {
		t.Fatalf("entry = %+v", got[0])
	}
}

func TestPagerOpensAResultThenReturnsThenCloses(t *testing.T) {
	entries := []pagerEntry{
		{label: "← 3 lines", body: "one\ntwo\nthree"},
		{label: "← 40 lines", body: "big body"},
	}
	p := newPager(entries, 80, 24)
	if p.cursor != 1 {
		t.Fatalf("cursor should start on the newest entry, got %d", p.cursor)
	}

	if open, _ := p.Update(key("enter")); !open || !p.showing {
		t.Fatalf("enter should open the chosen result")
	}
	if p.vp.View() == "" {
		t.Fatalf("the viewport should hold the body")
	}

	if open, _ := p.Update(key("esc")); !open || p.showing {
		t.Fatalf("esc should return to the list, not close")
	}

	if open, _ := p.Update(key("esc")); open {
		t.Fatalf("esc on the list should close the pager")
	}
}

func TestPagerMoveWraps(t *testing.T) {
	p := newPager([]pagerEntry{{label: "a"}, {label: "b"}}, 80, 24)
	p.cursor = 0
	p.move(-1)
	if p.cursor != 1 {
		t.Fatalf("move up from the top should wrap to the end, got %d", p.cursor)
	}
}
