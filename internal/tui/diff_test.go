package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/git"
)

func diffModel() Model {
	return Model{
		sel:       "a",
		focus:     focusDiff,
		diffs:     make(map[string]diffState),
		diffOpen:  make(map[string]map[string]bool),
		fileDiffs: make(map[string]map[string]string),
	}
}

func TestDiffPanelListsFilesAndCounts(t *testing.T) {
	m := diffModel()
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{
		{Status: "M", Path: "app.go", Insertions: 12, Deletions: 3},
		{Status: "A", Path: "git.go", Insertions: 40, Deletions: 0},
	}}

	text := visible(strings.Join(m.diffPanelLines(), "\n"))
	for _, want := range []string{"Changes · 2", "app.go", "+12", "−3", "git.go", "+40"} {
		if !strings.Contains(text, want) {
			t.Errorf("the panel does not show %q:\n%s", want, text)
		}
	}
}

func TestDiffPanelShowsAnExpandedFile(t *testing.T) {
	m := diffModel()
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{{Status: "M", Path: "app.go"}}}
	m.diffOpen["a"] = map[string]bool{"app.go": true}
	m.fileDiffs["a"] = map[string]string{"app.go": "@@ -1 +1 @@\n+added line\n"}

	text := visible(strings.Join(m.diffPanelLines(), "\n"))
	if !strings.Contains(text, "added line") {
		t.Errorf("an open file must show its diff:\n%s", text)
	}

	m.diffOpen["a"] = map[string]bool{}
	text = visible(strings.Join(m.diffPanelLines(), "\n"))
	if strings.Contains(text, "added line") {
		t.Errorf("a closed file must hide its diff:\n%s", text)
	}
}

func TestDiffPanelStates(t *testing.T) {
	m := diffModel()
	if got := visible(strings.Join(m.diffPanelLines(), "\n")); !strings.Contains(got, "reading") {
		t.Errorf("an unread session must say so: %q", got)
	}

	m.diffs["a"] = diffState{repo: false}
	if got := visible(strings.Join(m.diffPanelLines(), "\n")); !strings.Contains(got, "not a git repository") {
		t.Errorf("a non-repository must say so: %q", got)
	}

	m.diffs["a"] = diffState{repo: true}
	if got := visible(strings.Join(m.diffPanelLines(), "\n")); !strings.Contains(got, "no changes") {
		t.Errorf("a clean tree must say so: %q", got)
	}
}

func TestTogglingAFileFetchesThenHidesIt(t *testing.T) {
	m := diffModel()
	m.rows = []row{{name: "a", dir: "/tmp/a"}}
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{{Status: "M", Path: "app.go"}}}

	next, cmd := m.toggleDiffFile()
	m = next.(Model)
	if !m.diffOpen["a"]["app.go"] {
		t.Fatal("the first toggle must open the file")
	}
	if cmd == nil {
		t.Fatal("opening an unread file must fetch its diff")
	}

	next, _ = m.toggleDiffFile()
	m = next.(Model)
	if m.diffOpen["a"]["app.go"] {
		t.Fatal("the second toggle must close the file")
	}
}

func TestHandleDiffRefreshesTheOpenFiles(t *testing.T) {
	m := diffModel()
	m.rows = []row{{name: "a", dir: "/tmp/a"}}
	m.diffOpen["a"] = map[string]bool{"app.go": true}

	next, cmd := m.handleDiff(diffMsg{name: "a", repo: true, files: []git.FileChange{{Status: "M", Path: "app.go"}}})
	m = next.(Model)
	if !m.diffs["a"].repo {
		t.Fatal("handleDiff must store the state")
	}
	if cmd == nil {
		t.Fatal("handleDiff must re-fetch an open file")
	}
}

func TestBarShowsTheDiffCount(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.diffs["alpha"] = diffState{repo: true, stat: git.Stat{Insertions: 12, Deletions: 3, Files: 1}}
	bar := visible(m.barView())
	if !strings.Contains(bar, "+12") || !strings.Contains(bar, "−3") {
		t.Errorf("the bar must show the diff count:\n%s", bar)
	}

	m.diffs["alpha"] = diffState{repo: true, stat: git.Stat{}}
	if bar := visible(m.barView()); strings.Contains(bar, "+0") {
		t.Errorf("a clean tree must show no count:\n%s", bar)
	}
}

func TestOpeningThePanelResizesTheOutput(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	full := m.output.Width
	next, _ := m.openDiffPanel()
	m = next.(Model)
	if m.output.Width == full {
		t.Fatal("opening the panel must shrink the output width")
	}
	if m.output.Width != m.outputWidth() {
		t.Fatalf("output width = %d, want outputWidth %d", m.output.Width, m.outputWidth())
	}

	next, _ = m.closeDiffPanel()
	m = next.(Model)
	if m.output.Width != full {
		t.Fatalf("closing must restore the output width to %d, got %d", full, m.output.Width)
	}
}

func TestTabReachesTheOpenPanel(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	next, _ := m.openDiffPanel()
	m = next.(Model)
	m.focus = focusOutput

	next, _ = m.toggleFocus()
	m = next.(Model)
	if m.focus != focusDiff {
		t.Fatalf("Tab from the output must reach the diff panel, got %v", m.focus)
	}

	next, _ = m.toggleFocus()
	m = next.(Model)
	if m.focus != focusSidebar {
		t.Fatalf("Tab from the panel must move on to the sidebar, got %v", m.focus)
	}

	next, _ = m.closeDiffPanel()
	m = next.(Model)
	m.focus = focusOutput
	next, _ = m.toggleFocus()
	m = next.(Model)
	if m.focus != focusSidebar {
		t.Fatalf("with the panel closed, Tab from the output skips it, got %v", m.focus)
	}
}

func TestResizingTheDiffPanelPersists(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	next, _ := m.openDiffPanel()
	m = next.(Model)
	start := m.diffPanelWidth()
	wideOutput := m.outputWidth()

	next, _ = m.widenDiff()
	m = next.(Model)
	if m.diffPanelWidth() != start+diffWidthStep {
		t.Fatalf("d + gave width %d, want %d", m.diffPanelWidth(), start+diffWidthStep)
	}
	if m.outputWidth() >= wideOutput {
		t.Fatal("a wider panel must shrink the output")
	}

	widened := m.diffPanelWidth()
	next, _ = m.closeDiffPanel()
	m = next.(Model)
	next, _ = m.openDiffPanel()
	m = next.(Model)
	if m.diffPanelWidth() != widened {
		t.Fatalf("the panel forgot its width: %d, want %d", m.diffPanelWidth(), widened)
	}

	next, _ = m.narrowDiff()
	m = next.(Model)
	if m.diffPanelWidth() != widened-diffWidthStep {
		t.Fatalf("d - gave width %d, want %d", m.diffPanelWidth(), widened-diffWidthStep)
	}
}

func TestHalfWidthTogglesAndReturns(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	next, _ := m.openDiffPanel()
	m = next.(Model)
	set := m.diffPanelWidth()

	next, _ = m.toggleHalfDiff()
	m = next.(Model)
	if m.diffPanelWidth() != m.clampDiffWidth(160/2) {
		t.Fatalf("d / gave width %d, want about half the screen", m.diffPanelWidth())
	}

	next, _ = m.toggleHalfDiff()
	m = next.(Model)
	if m.diffPanelWidth() != set {
		t.Fatalf("d / again must return to the set width %d, got %d", set, m.diffPanelWidth())
	}
}

func TestTheDTargetStartsOnlyWithThePanelOpen(t *testing.T) {
	if _, ok := sequenceTarget("d", false, false); ok {
		t.Error("with the panel closed, d must not start a sequence")
	}
	if target, ok := sequenceTarget("d", false, true); !ok || target != "d" {
		t.Errorf("with the panel open, d must start the d target, got %q %v", target, ok)
	}
}

func TestLineNumbersToggle(t *testing.T) {
	m := diffModel()
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{{Status: "M", Path: "a.txt"}}}
	m.diffOpen["a"] = map[string]bool{"a.txt": true}
	m.fileDiffs["a"] = map[string]string{"a.txt": "@@ -1,2 +1,2 @@\n one\n+two\n"}

	off := visible(strings.Join(m.diffPanelLines(), "\n"))

	next, _ := m.toggleDiffNumbers()
	m = next.(Model)
	on := visible(strings.Join(m.diffPanelLines(), "\n"))

	if strings.Contains(off, " 2 ") {
		t.Errorf("line numbers must be off by default:\n%s", off)
	}
	if !strings.Contains(on, "2") {
		t.Errorf("d n must show the new-file line numbers:\n%s", on)
	}
}

func TestTheSelectedFileHasThePurpleBackground(t *testing.T) {
	m := diffModel()
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{
		{Status: "M", Path: "a.txt"},
		{Status: "M", Path: "b.txt"},
	}}
	m.diffSel = 1

	lines := m.diffPanelLines()
	selected, other := lines[3], lines[2]
	if !strings.Contains(selected, "48;5;62") {
		t.Errorf("the selected row must carry the purple background:\n%q", selected)
	}
	if strings.Contains(other, "48;5;62") {
		t.Errorf("an unselected row must not carry the background:\n%q", other)
	}
}

func TestSelectingAFileScrollsItIntoView(t *testing.T) {
	m := diffModel()
	files := make([]git.FileChange, 6)
	for i := range files {
		files[i] = git.FileChange{Status: "M", Path: "f" + strconv.Itoa(i)}
	}
	m.diffs["a"] = diffState{repo: true, files: files}

	m.diffSel = 5
	m.ensureDiffSelVisible()
	if m.diffScroll == 0 {
		t.Fatal("selecting a file below the fold must scroll the panel")
	}
}

func TestCollapsingTheSidebarReclaimsWidth(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 120, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusSidebar

	wide := m.baseOutputWidth()
	next, _ := m.collapseSidebar()
	m = next.(Model)

	if !m.sidebarHidden {
		t.Fatal("collapseSidebar must hide the sidebar")
	}
	if m.focus == focusSidebar {
		t.Fatal("a collapse must move the focus off the sidebar")
	}
	if got := m.baseOutputWidth() - wide; got != sidebarWidth {
		t.Fatalf("the panes gained %d columns, want %d", got, sidebarWidth)
	}

	next, _ = m.expandSidebar()
	m = next.(Model)
	if m.sidebarHidden {
		t.Fatal("expandSidebar must show the sidebar again")
	}
}
