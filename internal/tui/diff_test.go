package tui

import (
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
		fileDiffs: make(map[string]map[string][]string),
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
	m.fileDiffs["a"] = map[string][]string{"app.go": styleDiffLine("+added line")}

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
