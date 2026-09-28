package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func findModel(t *testing.T, rows ...string) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusOutput
	m.prompt.Blur()
	m.outputText = strings.Join(rows, "\n")
	m.setContent()
	m.output.GotoBottom()
	return m
}

func filler(n int) []string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("filler %d", i+1)
	}
	return rows
}

func typeFind(t *testing.T, m Model, text string) Model {
	t.Helper()
	m, _ = step(t, m, key("/"))
	if !m.find.on {
		t.Fatal("/ must open the find box")
	}
	for _, r := range text {
		m, _ = step(t, m, key(string(r)))
	}
	return m
}

func TestParseFind(t *testing.T) {
	cases := []struct {
		raw     string
		needle  string
		prompts bool
	}{
		{"fix", "fix", false},
		{"FIX", "fix", false},
		{">", "", true},
		{">fix", "fix", true},
		{"> fix", "fix", true},
	}
	for _, c := range cases {
		needle, prompts := parseFind(c.raw)
		if needle != c.needle || prompts != c.prompts {
			t.Errorf("parseFind(%q) = %q, %v, want %q, %v", c.raw, needle, prompts, c.needle, c.prompts)
		}
	}
}

func TestFindHitsReadsEveryRowAndThePromptRowsAlone(t *testing.T) {
	content := strings.Join([]string{
		"› fix the parser",
		"  and the lexer",
		"I fixed the parser",
		"› read the file",
		"done",
	}, "\n")

	if got := findHits(content, "fix", false); len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("findHits(fix) = %v, want [0 2]", got)
	}
	if got := findHits(content, "fix", true); len(got) != 1 || got[0] != 0 {
		t.Fatalf("findHits(>fix) = %v, want [0]", got)
	}
	if got := findHits(content, "", true); len(got) != 2 || got[0] != 0 || got[1] != 3 {
		t.Fatalf("findHits(>) = %v, want [0 3]", got)
	}
	if got := findHits(content, "", false); got != nil {
		t.Fatalf("an empty needle finds nothing, got %v", got)
	}
}

func TestSeekRowWraps(t *testing.T) {
	hits := []int{2, 9, 20}
	cases := []struct {
		name    string
		from    int
		back    bool
		row     int
		wrapped bool
	}{
		{"backwards", 15, true, 9, false},
		{"backwards wraps at the top", 2, true, 20, true},
		{"forwards", 9, false, 20, false},
		{"forwards wraps at the bottom", 20, false, 2, true},
	}
	for _, c := range cases {
		row, wrapped := seekRow(hits, c.from, c.back)
		if row != c.row || wrapped != c.wrapped {
			t.Errorf("%s: seekRow = %d, %v, want %d, %v", c.name, row, wrapped, c.row, c.wrapped)
		}
	}
}

func TestSlashSearchesBackwards(t *testing.T) {
	rows := append([]string{"› fix the parser"}, filler(40)...)
	rows = append(rows, "the newest line")
	m := findModel(t, rows...)
	bottom := m.output.YOffset()

	m = typeFind(t, m, "fix")
	if m.find.row != 0 {
		t.Fatalf("the hit is row %d, want row 0", m.find.row)
	}
	if m.output.YOffset() >= bottom {
		t.Fatalf("the pane did not scroll up: offset %d, bottom %d", m.output.YOffset(), bottom)
	}

	m, _ = step(t, m, key("enter"))
	if m.find.on {
		t.Fatal("enter must close the box")
	}
	if !m.find.live() {
		t.Fatal("enter must keep the needle")
	}
}

func TestTheStepKeysWalkTheHitsAndWrap(t *testing.T) {
	rows := []string{"fix one"}
	rows = append(rows, filler(10)...)
	rows = append(rows, "fix two")
	rows = append(rows, filler(10)...)
	rows = append(rows, "fix three")
	rows = append(rows, filler(10)...)
	m := findModel(t, rows...)

	m = typeFind(t, m, "fix")
	m, _ = step(t, m, key("enter"))
	if m.find.row != 11 {
		t.Fatalf("the first hit is row %d, want row 11, the nearest above the top of the pane", m.find.row)
	}
	if count := m.findCount(); !strings.HasPrefix(count, "2/3") {
		t.Fatalf("the status shows %q, want 2/3", count)
	}

	m, _ = step(t, m, key("n"))
	if m.find.row != 0 {
		t.Fatalf("n steps to row %d, want row 0", m.find.row)
	}
	m, _ = step(t, m, key("N"))
	if m.find.row != 11 {
		t.Fatalf("N steps back to row %d, want row 11", m.find.row)
	}
	m, _ = step(t, m, key("N"))
	if m.find.row != 22 || m.find.note != "" {
		t.Fatalf("N steps to row %d with note %q, want row 22 and no note", m.find.row, m.find.note)
	}
	m, _ = step(t, m, key("N"))
	if m.find.row != 0 || m.find.note != "wrapped" {
		t.Fatalf("N wraps to row %d with note %q, want row 0 and wrapped", m.find.row, m.find.note)
	}
	m, _ = step(t, m, key("n"))
	if m.find.row != 22 || m.find.note != "wrapped" {
		t.Fatalf("n wraps to row %d with note %q, want row 22 and wrapped", m.find.row, m.find.note)
	}
}

func TestEscWhileTypingRestoresTheOffset(t *testing.T) {
	rows := append([]string{"fix the parser"}, filler(40)...)
	m := findModel(t, rows...)
	bottom := m.output.YOffset()

	m = typeFind(t, m, "fix")
	if m.output.YOffset() == bottom {
		t.Fatal("the search must move the pane")
	}
	m, _ = step(t, m, key("esc"))
	if m.find.live() || m.find.on {
		t.Fatal("esc must drop the needle")
	}
	if m.output.YOffset() != bottom {
		t.Fatalf("esc leaves the offset at %d, want %d", m.output.YOffset(), bottom)
	}
}

func TestTheNeedleReleasesTheNewSessionKey(t *testing.T) {
	rows := append([]string{"fix the parser"}, filler(40)...)
	m := findModel(t, rows...)

	m = typeFind(t, m, "fix")
	m, _ = step(t, m, key("enter"))
	m, _ = step(t, m, key("n"))
	if m.form != nil {
		t.Fatal("n must step the hits while a needle is live")
	}

	m, _ = step(t, m, key("esc"))
	if m.find.live() {
		t.Fatal("esc on the pane must clear the needle")
	}
	m, _ = step(t, m, key("n"))
	if m.form == nil {
		t.Fatal("n must start a new session once the needle is cleared")
	}
}

func TestNewOutputHoldsTheHitOnItsRow(t *testing.T) {
	rows := append([]string{"fix the parser"}, filler(40)...)
	m := findModel(t, rows...)

	m = typeFind(t, m, "fix")
	m, _ = step(t, m, key("enter"))
	row := m.find.row

	m.outputText += "\n" + strings.Join(filler(5), "\n")
	m.setContent()
	if m.find.row != row {
		t.Fatalf("new output moved the hit to row %d, want row %d", m.find.row, row)
	}
}

func TestThePaneMarksTheCurrentHit(t *testing.T) {
	rows := append([]string{"fix the parser"}, filler(40)...)
	m := findModel(t, rows...)

	m = typeFind(t, m, "fix")
	painted := strings.Split(m.findPaint(m.content), "\n")
	plain := strings.Split(m.content, "\n")
	if painted[0] == plain[0] {
		t.Fatal("the current hit must be marked")
	}
	if !strings.Contains(visible(painted[0]), "fix the parser") {
		t.Fatalf("the marked row lost its text: %q", visible(painted[0]))
	}
	if painted[1] != plain[1] {
		t.Fatal("only the current hit is marked")
	}
}

func TestAClosedBlockHidesItsRowsFromTheSearch(t *testing.T) {
	m := outputModel(t)
	m, _ = step(t, m, settingsMsg{caps: capsAll(3)})
	lines := []render.Line{{Class: render.ClassToolResult, Text: body(20) + "\nthe needle"}}
	if err := m.mgr.AppendLines(m.sel, lines); err != nil {
		t.Fatalf("AppendLines: %v", err)
	}
	m.rebuildOutput()
	if len(m.capped) != 1 {
		t.Fatalf("capped = %v, want the one large block", m.capped)
	}

	m = typeFind(t, m, "the needle")
	if len(m.find.hits) != 0 {
		t.Fatalf("a closed block hides its rows, got hits %v", m.find.hits)
	}

	m.toggleBlock(m.capped[0])
	if len(m.find.hits) != 1 {
		t.Fatalf("an open block shows its rows, got hits %v", m.find.hits)
	}
}
