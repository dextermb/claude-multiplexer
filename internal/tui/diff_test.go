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

func TestTheDiffTickStopsWhenThePanelCloses(t *testing.T) {
	m := diffModel()
	m.rows = []row{{name: "a", dir: "/tmp/a"}}
	m.diffPanel = true
	m.diffTicking = true

	_, cmd := m.handleDiffTick()
	if cmd == nil {
		t.Fatal("an open panel must keep ticking")
	}

	m.diffPanel = false
	next, cmd := m.handleDiffTick()
	m = next.(Model)
	if m.diffTicking {
		t.Fatal("a closed panel must stop ticking")
	}
	if cmd != nil {
		t.Fatal("a closed panel must not schedule another tick")
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

func TestArrowsScrollAndJKPickFiles(t *testing.T) {
	m := diffModel()
	files := make([]git.FileChange, 8)
	for i := range files {
		files[i] = git.FileChange{Status: "M", Path: "f" + strconv.Itoa(i)}
	}
	m.diffs["a"] = diffState{repo: true, files: files}

	next, _ := m.diffKey(key("down"))
	m = next.(Model)
	if m.diffScroll != 1 {
		t.Fatalf("down must scroll one line, got %d", m.diffScroll)
	}
	if m.diffSel != 0 {
		t.Fatalf("down must not move the file selection, got %d", m.diffSel)
	}

	next, _ = m.diffKey(key("j"))
	m = next.(Model)
	if m.diffSel != 1 {
		t.Fatalf("j must move the file selection, got %d", m.diffSel)
	}
}

func longDiffModel(t *testing.T) Model {
	t.Helper()
	m := diffModel()
	m.height = 20
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{
		{Status: "M", Path: "f0"},
		{Status: "M", Path: "f1"},
	}}
	m.diffOpen["a"] = map[string]bool{"f0": true}
	body := "@@ -1,100 +1,100 @@\n"
	for i := range 100 {
		body += "+line " + strconv.Itoa(i) + "\n"
	}
	m.fileDiffs["a"] = map[string]string{"f0": body}
	if m.selectedBodyLen() <= m.bodyHeight() {
		t.Fatalf("the open diff must be taller than the viewport (%d vs %d)", m.selectedBodyLen(), m.bodyHeight())
	}
	return m
}

func TestJScrollsAnOpenDiffThenMovesToTheNextFile(t *testing.T) {
	m := longDiffModel(t)

	next, _ := m.diffKey(key("j"))
	m = next.(Model)
	if m.diffSel != 0 {
		t.Fatalf("j in a long open diff must not move the file yet, got %d", m.diffSel)
	}
	if m.diffScroll == 0 {
		t.Fatal("j must scroll through the open diff")
	}

	for range 500 {
		if m.diffSel == 1 {
			break
		}
		next, _ = m.diffKey(key("j"))
		m = next.(Model)
	}
	if m.diffSel != 1 {
		t.Fatal("j at the bottom of the diff must move to the next file")
	}
}

func TestKEntersThePreviousOpenDiffAtItsBottom(t *testing.T) {
	m := longDiffModel(t)
	m.diffSel = 1
	m.ensureDiffSelVisible()

	next, _ := m.diffKey(key("k"))
	m = next.(Model)
	if m.diffSel != 0 {
		t.Fatalf("k must move to the previous file, got %d", m.diffSel)
	}

	row := m.diffFileLine(0)
	last := row + m.selectedBodyLen()
	if last < m.diffScroll || last >= m.diffScroll+m.bodyHeight() {
		t.Fatalf("k must reveal the bottom of the previous diff, scroll %d last %d", m.diffScroll, last)
	}
	if row >= m.diffScroll {
		t.Fatal("k must enter the previous diff at its bottom, not its top")
	}

	before := m.diffScroll
	next, _ = m.diffKey(key("k"))
	m = next.(Model)
	if m.diffScroll != before-1 {
		t.Fatalf("k must then scroll up the diff, got %d want %d", m.diffScroll, before-1)
	}
}

func TestTheCurrentLineIsMarked(t *testing.T) {
	m := diffModel()
	m.height = 30
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{{Status: "M", Path: "f0"}}}
	m.diffOpen["a"] = map[string]bool{"f0": true}
	m.fileDiffs["a"] = map[string]string{"f0": "@@ -1,3 +1,3 @@\n+one\n+two\n+three\n"}

	const target = 4 // header, blank, row, hunk, then "+one"

	m.diffScroll = target
	marked := m.diffPanelLines()[target]
	m.diffScroll = 0
	plain := m.diffPanelLines()[target]
	if marked == plain {
		t.Fatalf("numbers off: the current line must render differently:\n marked %q\n plain  %q", marked, plain)
	}

	m.diffLineNumbers = true
	m.diffScroll = target
	markedNum := m.diffPanelLines()[target]
	m.diffScroll = 0
	plainNum := m.diffPanelLines()[target]
	if markedNum == plainNum {
		t.Fatalf("numbers on: the current line number must render differently:\n marked %q\n plain  %q", markedNum, plainNum)
	}
}

func TestBracketsJumpBetweenTheEmptyLines(t *testing.T) {
	m := diffModel()
	m.height = 12
	m.diffs["a"] = diffState{repo: true, files: []git.FileChange{{Status: "M", Path: "f0"}}}
	m.diffOpen["a"] = map[string]bool{"f0": true}

	var b strings.Builder
	b.WriteString("@@ -1,60 +1,60 @@\n")
	for i := range 60 {
		if i%10 == 0 {
			b.WriteString("+\n")
		} else {
			b.WriteString("+line " + strconv.Itoa(i) + "\n")
		}
	}
	m.fileDiffs["a"] = map[string]string{"f0": b.String()}

	rows := m.diffEmptyRows()
	if len(rows) < 3 {
		t.Fatalf("expected several empty rows, got %v", rows)
	}

	next, _ := m.diffKey(key("}"))
	m = next.(Model)
	if m.diffScroll != rows[0] {
		t.Fatalf("shift+] must jump to the first empty line %d, got %d", rows[0], m.diffScroll)
	}

	next, _ = m.diffKey(key("}"))
	m = next.(Model)
	if m.diffScroll != rows[1] {
		t.Fatalf("shift+] again must jump to the next empty line %d, got %d", rows[1], m.diffScroll)
	}

	next, _ = m.diffKey(key("{"))
	m = next.(Model)
	if m.diffScroll != rows[0] {
		t.Fatalf("shift+[ must jump back to the previous empty line %d, got %d", rows[0], m.diffScroll)
	}
}

func TestGAndCapitalGJumpTheFileListWhenClosed(t *testing.T) {
	m := diffModel()
	files := make([]git.FileChange, 6)
	for i := range files {
		files[i] = git.FileChange{Status: "M", Path: "f" + strconv.Itoa(i)}
	}
	m.diffs["a"] = diffState{repo: true, files: files}
	m.diffSel = 2

	next, _ := m.diffKey(key("G"))
	m = next.(Model)
	if m.diffSel != 5 {
		t.Fatalf("G with no file open must select the last file, got %d", m.diffSel)
	}

	next, _ = m.diffKey(key("g"))
	m = next.(Model)
	if m.diffSel != 0 {
		t.Fatalf("g with no file open must select the first file, got %d", m.diffSel)
	}
	if m.diffScroll != 0 {
		t.Fatalf("g must scroll the list to the top, got %d", m.diffScroll)
	}
}

func TestGAndCapitalGScrollTheDiffWhenOpen(t *testing.T) {
	m := diffModel()
	m.height = 30
	files := make([]git.FileChange, 3)
	for i := range files {
		files[i] = git.FileChange{Status: "M", Path: "f" + strconv.Itoa(i)}
	}
	m.diffs["a"] = diffState{repo: true, files: files}
	m.diffOpen["a"] = map[string]bool{"f0": true}
	body := "@@ -1 +1 @@\n"
	for i := range 200 {
		body += "+line " + strconv.Itoa(i) + "\n"
	}
	m.fileDiffs["a"] = map[string]string{"f0": body}
	m.diffSel = 0

	next, _ := m.diffKey(key("G"))
	m = next.(Model)
	if m.diffSel != 0 {
		t.Fatalf("G with a file open must not move the selection, got %d", m.diffSel)
	}
	if m.diffScroll == 0 {
		t.Fatal("G with a file open must scroll to the bottom of the diff")
	}
	bottom := m.diffScroll

	next, _ = m.diffKey(key("g"))
	m = next.(Model)
	if m.diffScroll != 0 {
		t.Fatalf("g with a file open must scroll to the top, got %d", m.diffScroll)
	}
	if bottom <= 0 {
		t.Fatal("the diff must be long enough to scroll")
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

func TestTabSkipsTheHiddenSidebar(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	next, _ := m.collapseSidebar()
	m = next.(Model)
	m.focus = focusOutput

	next, _ = m.toggleFocus()
	m = next.(Model)
	if m.focus != focusPrompt {
		t.Fatalf("Tab from the output must skip the hidden sidebar and reach the prompt, got %v", m.focus)
	}

	for i := 0; i < 5; i++ {
		next, _ = m.toggleFocus()
		m = next.(Model)
		if m.focus == focusSidebar {
			t.Fatalf("cycle step %d landed on the hidden sidebar", i)
		}
	}
}

func TestEscDoesNotRetreatToTheHiddenSidebar(t *testing.T) {
	m := Model{sidebarHidden: true}
	if m.retreatFocus() == focusSidebar {
		t.Fatal("Esc must not retreat to the hidden sidebar")
	}
	m.sidebarHidden = false
	if m.retreatFocus() != focusSidebar {
		t.Fatal("Esc retreats to the sidebar when it is shown")
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
