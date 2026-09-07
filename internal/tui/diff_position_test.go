package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

func posPtr(s string) *string { return &s }
func sizePtr(n int) *int      { return &n }

func gridModel(width, height int, files []git.FileChange) Model {
	m := diffModel()
	m.width = width
	m.height = height
	m.layout = config.ResolvedLayout{
		DiffPosition: config.DiffBottom,
		DiffSize:     12,
		PromptMin:    1,
		PromptMax:    1,
	}
	m.diffPanel = true
	m.diffs["a"] = oneGroup(files...)
	return m
}

func sampleFiles(n int) []git.FileChange {
	out := make([]git.FileChange, n)
	for i := range out {
		out[i] = git.FileChange{Status: "M", Path: "pkg/file" + string(rune('a'+i)) + ".go", Insertions: i, Deletions: 1}
	}
	return out
}

func TestDiffBottomUsesFullWidthAndShortensOutput(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 40)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.layouts = map[string]config.Layout{"stack": {DiffPosition: posPtr(config.DiffBottom), DiffSize: sizePtr(10)}}
	m.activeLayout = "stack"
	m.applyLayout()

	base := m.baseOutputWidth()
	next, _ := m.openDiffPanel()
	m = next.(Model)

	if !m.diffHorizontal() {
		t.Fatal("the bottom position must be horizontal")
	}
	if m.outputWidth() != base {
		t.Fatalf("output width = %d, want the full %d on a horizontal panel", m.outputWidth(), base)
	}
	if m.diffPanelWidth() != base {
		t.Fatalf("panel width = %d, want the full %d", m.diffPanelWidth(), base)
	}
	h := m.diffPanelHeight()
	if h <= 0 {
		t.Fatalf("panel height = %d, want a positive height", h)
	}
	if want := m.bodyHeight() - barHeight - h; m.outputHeight() != want {
		t.Fatalf("output height = %d, want %d (shrunk by the panel)", m.outputHeight(), want)
	}
}

func TestDiffGridIsMultiColumnAndOpensBelow(t *testing.T) {
	files := sampleFiles(6)
	m := gridModel(80, 40, files)

	cols := m.diffGridCols(len(files))
	if cols < 2 {
		t.Fatalf("a horizontal panel must draw more than one column, got %d", cols)
	}
	gridRows := (len(files) + cols - 1) / cols

	closed := len(m.diffPanelLines())
	if want := 1 + gridRows; closed != want {
		t.Fatalf("closed panel has %d lines, want %d (header plus grid rows)", closed, want)
	}

	m.diffOpen["a"] = map[fileKey]bool{fk(files[0].Path): true}
	m.fileDiffs["a"] = map[fileKey]string{fk(files[0].Path): "@@ -1 +1 @@\n+added line\n"}
	open := m.diffPanelLines()
	if len(open) <= closed {
		t.Fatal("opening a file must add its diff below the grid")
	}
	found := false
	for _, line := range open[closed:] {
		if strings.Contains(ansi.Strip(line), "added line") {
			found = true
		}
	}
	if !found {
		t.Fatal("the open diff must draw below the whole grid")
	}
}

func TestDiffGridNavigatesInTwoDimensions(t *testing.T) {
	files := sampleFiles(6)
	m := gridModel(80, 40, files)
	cols := m.diffGridCols(len(files))
	if cols < 2 || cols >= len(files) {
		t.Fatalf("this test needs a small grid, got %d columns for %d files", cols, len(files))
	}

	next, _ := m.diffKey(key("l"))
	m = next.(Model)
	if m.diffSel != 1 {
		t.Fatalf("l must move one file right, got %d", m.diffSel)
	}
	next, _ = m.diffKey(key("j"))
	m = next.(Model)
	if m.diffSel != 1+cols {
		t.Fatalf("j must move one grid row down (by %d), got %d", cols, m.diffSel)
	}
	next, _ = m.diffKey(key("k"))
	m = next.(Model)
	if m.diffSel != 1 {
		t.Fatalf("k must move one grid row up, got %d", m.diffSel)
	}
	next, _ = m.diffKey(key("h"))
	m = next.(Model)
	if m.diffSel != 0 {
		t.Fatalf("h must move one file left, got %d", m.diffSel)
	}
}

func TestDiffResizeGrowsHeightWhenHorizontal(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 40)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.layouts = map[string]config.Layout{"stack": {DiffPosition: posPtr(config.DiffBottom), DiffSize: sizePtr(10)}}
	m.activeLayout = "stack"
	m.applyLayout()

	next, _ := m.openDiffPanel()
	m = next.(Model)
	start := m.diffPanelHeight()

	next, _ = m.widenDiff()
	m = next.(Model)
	if m.diffPanelHeight() != start+diffSizeStep {
		t.Fatalf("d + gave height %d, want %d (grows the height on a horizontal side)", m.diffPanelHeight(), start+diffSizeStep)
	}
	if m.diffPanelWidth() != m.baseOutputWidth() {
		t.Fatal("a horizontal panel keeps the full width when it resizes")
	}
}
