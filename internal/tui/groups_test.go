package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

// groupModel builds a model whose rows are given, and whose group key is the
// directory of the row, so a test does not need a repository on disk.
func groupModel(t *testing.T, height int, rows ...row) Model {
	t.Helper()
	m, _ := newTestModel(t, "")
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 100, Height: height})
	m.form = nil
	children := make(map[string]bool, len(rows))
	for _, item := range rows {
		if item.parent != "" {
			children[item.parent] = true
		}
	}
	for i := range rows {
		rows[i].group = m.rowGroup(rows[i], children)
	}
	m.rows, m.groups = groupRows(rows, m.folded)
	m.buildLines()
	m.selectFirst()
	m.clampOffset()
	return m
}

func liveRow(name, dir string, state session.State) row {
	return row{name: name, dir: dir, live: true, state: state, label: state.String()}
}

func storedRow(name, dir string) row {
	return row{name: name, dir: dir, label: "stored"}
}

func groupLabelsOf(m Model) []string {
	out := make([]string, 0, len(m.groups))
	for _, item := range m.groups {
		out = append(out, item.label)
	}
	return out
}

func rowNames(m Model) []string {
	out := make([]string, 0, len(m.rows))
	for _, item := range m.rows {
		out = append(out, item.name)
	}
	return out
}

func TestAGroupWithALiveSessionComesFirst(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		storedRow("notes", "/work/two"),
		liveRow("docs", "/work/two", session.StateIdle),
		storedRow("old", "/work/three"),
	)

	if got := groupLabelsOf(m); strings.Join(got, ",") != "one,two,three" {
		t.Fatalf("groups = %v, want the live groups first", got)
	}
	if got := rowNames(m); strings.Join(got, ",") != "api,notes,docs,old" {
		t.Fatalf("rows = %v, want each group to keep its own order", got)
	}
}

func TestEveryGroupHasAHeaderLine(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("docs", "/work/one", session.StateIdle),
		liveRow("web", "/work/two", session.StateIdle),
	)

	if len(m.lines) != 5 {
		t.Fatalf("lines = %d, want 3 rows and 2 headers", len(m.lines))
	}
	if !m.lines[0].header() || !m.lines[3].header() {
		t.Fatalf("lines = %+v, want a header above each group", m.lines)
	}

	view := visible(m.sidebarView())
	for _, want := range []string{"one", "two", "api", "docs", "web"} {
		if !strings.Contains(view, want) {
			t.Errorf("the sidebar has no %q:\n%s", want, view)
		}
	}
}

func TestAFoldedGroupHidesItsRows(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("docs", "/work/one", session.StateIdle),
		liveRow("web", "/work/two", session.StateIdle),
	)

	m.setFold(dirPrefix+"/work/one", true)

	if len(m.lines) != 3 {
		t.Fatalf("lines = %d, want the two headers and the one row left", len(m.lines))
	}
	view := visible(m.sidebarView())
	if strings.Contains(view, "api") || strings.Contains(view, "docs") {
		t.Errorf("a folded group still shows its rows:\n%s", view)
	}
	if !strings.Contains(view, "one") || !strings.Contains(view, "web") {
		t.Errorf("a folded group must keep its header, and its neighbours:\n%s", view)
	}

	m.setFold(dirPrefix+"/work/one", false)
	if len(m.lines) != 5 {
		t.Fatalf("lines = %d after unfolding, want every row back", len(m.lines))
	}
}

func TestFoldingTheSelectedGroupTakesTheSelectionOut(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("web", "/work/two", session.StateIdle),
	)
	if m.sel != "api" {
		t.Fatalf("selected = %q, want the first row", m.sel)
	}

	m.setFold(dirPrefix+"/work/one", true)
	if m.sel != "web" {
		t.Fatalf("selected = %q, want the row of the next group", m.sel)
	}

	m.setFold(dirPrefix+"/work/two", true)
	if m.sel != "web" {
		t.Fatalf("selected = %q, want the selection to stay when every group is folded", m.sel)
	}
}

func TestFoldingTheLastGroupTakesTheSelectionBack(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("web", "/work/two", session.StateIdle),
	)
	next, _ := m.move(1)
	m = next.(Model)
	if m.sel != "web" {
		t.Fatalf("selected = %q, want web", m.sel)
	}

	m.setFold(dirPrefix+"/work/two", true)
	if m.sel != "api" {
		t.Fatalf("selected = %q, want the row before the folded group", m.sel)
	}
}

func TestMovingStepsOverAHeaderAndOverAFoldedGroup(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("docs", "/work/two", session.StateIdle),
		liveRow("web", "/work/three", session.StateIdle),
	)

	next, _ := m.move(1)
	m = next.(Model)
	if m.sel != "docs" {
		t.Fatalf("selected = %q, want the row of the next group", m.sel)
	}

	m.setFold(dirPrefix+"/work/three", true)
	next, _ = m.move(1)
	m = next.(Model)
	if m.sel != "docs" {
		t.Fatalf("selected = %q, want no move into a folded group", m.sel)
	}

	next, _ = m.move(-1)
	m = next.(Model)
	if m.sel != "api" {
		t.Fatalf("selected = %q, want api", m.sel)
	}
}

func TestTheHeaderOfAFoldedGroupShowsTheMostUrgentRow(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("docs", "/work/one", session.StateWaiting),
		liveRow("web", "/work/two", session.StateIdle),
	)
	m.setFold(dirPrefix+"/work/one", true)

	header := visible(m.groupHeader(m.groups[0]))
	if !strings.Contains(header, foldShutMark) {
		t.Errorf("header = %q, want the folded mark", header)
	}
	if !strings.Contains(header, "?") {
		t.Errorf("header = %q, want the glyph of the waiting session", header)
	}
	if !strings.Contains(header, "2") {
		t.Errorf("header = %q, want the count of the rows it hides", header)
	}

	open := visible(m.groupHeader(m.groups[1]))
	if !strings.Contains(open, foldOpenMark) {
		t.Errorf("header = %q, want the unfolded mark", open)
	}
}

func TestTheScrollOffsetCountsTheHeaderLines(t *testing.T) {
	m := groupModel(t, 10,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("api2", "/work/one", session.StateIdle),
		liveRow("docs", "/work/two", session.StateIdle),
		liveRow("docs2", "/work/two", session.StateIdle),
		liveRow("web", "/work/three", session.StateIdle),
		liveRow("web2", "/work/three", session.StateIdle),
	)
	if len(m.lines) != 9 {
		t.Fatalf("lines = %d, want 6 rows and 3 headers", len(m.lines))
	}
	if m.visibleLines() >= len(m.lines) {
		t.Fatalf("the sidebar shows %d lines, so the list does not scroll", m.visibleLines())
	}

	for range 5 {
		next, _ := m.move(1)
		m = next.(Model)
	}
	if m.sel != "web2" {
		t.Fatalf("selected = %q, want the last row", m.sel)
	}
	if want := len(m.lines) - m.visibleLines(); m.listOffset != want {
		t.Fatalf("listOffset = %d, want %d", m.listOffset, want)
	}
	if lines := strings.Split(visible(m.sidebarView()), "\n"); !strings.Contains(strings.Join(lines, "\n"), "web2") {
		t.Fatal("the selected row must be on the screen")
	}
}

func TestFoldEveryGroupKeepsTheOneYouAreIn(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("docs", "/work/two", session.StateIdle),
		liveRow("web", "/work/three", session.StateIdle),
	)
	next, _ := m.move(1)
	m = next.(Model)

	next, _ = m.toggleAllFolds()
	m = next.(Model)
	if m.sel != "docs" {
		t.Fatalf("selected = %q, want the group of the selection to stay open", m.sel)
	}
	if len(m.lines) != 4 {
		t.Fatalf("lines = %d, want 3 headers and the one row left", len(m.lines))
	}

	next, _ = m.toggleAllFolds()
	m = next.(Model)
	if len(m.lines) != 6 {
		t.Fatalf("lines = %d, want every group unfolded again", len(m.lines))
	}
}

func TestZFoldsTheGroupOfTheSelectedSession(t *testing.T) {
	m := groupModel(t, 30,
		liveRow("api", "/work/one", session.StateIdle),
		liveRow("web", "/work/two", session.StateIdle),
	)

	next, _ := m.sidebarKey(key("z"))
	m = next.(Model)
	if !m.folded[dirPrefix+"/work/one"] {
		t.Fatal("z must fold the group of the selected session")
	}
	if m.sel != "web" {
		t.Fatalf("selected = %q, want the row of the next group", m.sel)
	}

	next, _ = m.sidebarKey(key("z"))
	m = next.(Model)
	if !m.folded[dirPrefix+"/work/two"] {
		t.Fatal("z must fold the group the selection moved to")
	}
}

func TestAClickOnAHeaderFoldsTheGroup(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	one, two := t.TempDir(), t.TempDir()
	m = spawn(t, m, mgr, "alpha", one)
	m = spawn(t, m, mgr, "beta", two)

	if len(m.groups) != 2 {
		t.Fatalf("groups = %d, want one for each directory", len(m.groups))
	}

	m, _ = step(t, m, tea.MouseMsg{
		X: 3, Y: titleHeight, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if !m.groups[0].folded {
		t.Fatal("a click on a header must fold that group")
	}
	if strings.Contains(visible(m.sidebarView()), "alpha") {
		t.Fatal("the folded group still shows its row")
	}

	m, _ = step(t, m, tea.MouseMsg{
		X: 3, Y: titleHeight, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if m.groups[0].folded {
		t.Fatal("a second click must unfold the group")
	}
}

func TestANewSessionUnfoldsItsGroup(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m.setFold(m.rows[0].group, true)
	if !m.groups[0].folded {
		t.Fatal("the group must be folded before the next session starts")
	}

	m = spawn(t, m, mgr, "beta", dir)
	if m.groups[0].folded {
		t.Fatal("a session that starts in a folded group must unfold it")
	}
	if m.sel != "beta" {
		t.Fatalf("selected = %q, want the new session", m.sel)
	}
}

func TestSessionsInOneRepositoryShareAGroup(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	repo := t.TempDir()
	mkdir(t, repo, ".git")
	inner := mkdir(t, repo, "internal")
	tree := mkdir(t, repo, ".worktrees", "api")
	writeGitFile(t, tree, "gitdir: "+repo+"/.git/worktrees/api")

	m = spawn(t, m, mgr, "alpha", inner)
	m = spawn(t, m, mgr, "beta", tree)

	if len(m.groups) != 1 {
		t.Fatalf("groups = %d, want the repository to hold both sessions", len(m.groups))
	}
	if m.groups[0].count != 2 {
		t.Fatalf("count = %d, want 2", m.groups[0].count)
	}
}
