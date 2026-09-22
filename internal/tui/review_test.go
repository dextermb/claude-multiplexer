package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/git"
	"github.com/dextermb/claude-multiplexer/internal/manager"
)

const twoHunks = `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@ func A()
 one
+two
 three
@@ -20,2 +21,3 @@ func B()
 four
+five
`

const oneHunk = "@@ -0,0 +1,2 @@\n+alpha\n+beta\n"

func reviewModel() Model {
	m := diffModel()
	m.reviewMode = true
	m.prompt = textarea.New()
	m.width = 160
	m.height = 40
	m.diffs["a"] = oneGroup(
		git.FileChange{Status: "M", Path: "a.go", Insertions: 2, Deletions: 0},
		git.FileChange{Status: "A", Path: "b.go", Insertions: 2, Deletions: 0},
	)
	m.fileDiffs["a"] = map[fileKey]string{fk("a.go"): twoHunks, fk("b.go"): oneHunk}
	return m
}

func TestReviewNavigatesHunksThenRollsToNextFile(t *testing.T) {
	m := reviewModel()

	next, _ := m.reviewMoveHunk(1)
	m = next.(Model)
	if m.reviewFile != 0 || m.reviewHunk != 1 {
		t.Fatalf("j must step to the second hunk, got file %d hunk %d", m.reviewFile, m.reviewHunk)
	}

	next, _ = m.reviewMoveHunk(1)
	m = next.(Model)
	if m.reviewFile != 1 || m.reviewHunk != 0 {
		t.Fatalf("j past the last hunk must roll to the next file, got file %d hunk %d", m.reviewFile, m.reviewHunk)
	}

	next, _ = m.reviewMoveHunk(-1)
	m = next.(Model)
	if m.reviewFile != 0 {
		t.Fatalf("k from the first hunk must roll to the previous file, got file %d", m.reviewFile)
	}
}

func TestReviewHunkPromptNamesTheLineRange(t *testing.T) {
	prompt := hunkExplainPrompt("a.go", git.Hunks(twoHunks)[1])
	if !strings.Contains(prompt, "a.go:21-23") {
		t.Errorf("prompt must name the location:\n%s", prompt)
	}
	if strings.Contains(prompt, "+five") {
		t.Errorf("prompt must not carry the hunk text:\n%s", prompt)
	}
	if !strings.Contains(prompt, "origin/HEAD") {
		t.Errorf("prompt must name the base ref:\n%s", prompt)
	}
}

func TestReviewLineNumbersToggle(t *testing.T) {
	m := reviewModel()
	width := m.reviewDiffWidth()

	off := visible(strings.Join(firstOf(m.reviewDiffContent(width)), "\n"))

	next, _ := m.reviewToggleNumbers()
	m = next.(Model)
	if !m.reviewLineNumbers {
		t.Fatal("n must turn the line numbers on")
	}
	on := visible(strings.Join(firstOf(m.reviewDiffContent(width)), "\n"))

	// +five is the new-side line 22 of the second hunk, a number the diff text
	// carries nowhere else, so it can only come from the gutter.
	if strings.Contains(off, "22") {
		t.Errorf("line numbers must be off by default:\n%s", off)
	}
	if !strings.Contains(on, "22") {
		t.Errorf("n must show the new-side line numbers:\n%s", on)
	}

	next, _ = m.reviewToggleNumbers()
	m = next.(Model)
	if m.reviewLineNumbers {
		t.Fatal("n again must turn the line numbers off")
	}
}

func firstOf(lines []string, _ int) []string { return lines }

func TestReviewTabCyclesFocus(t *testing.T) {
	m := reviewModel()
	m.reviewFocus = reviewDiff
	for _, want := range []reviewSide{reviewExplain, reviewPrompt, reviewDiff} {
		next, _ := m.reviewCycleFocus()
		m = next.(Model)
		if m.reviewFocus != want {
			t.Fatalf("tab moved focus to %v, want %v", m.reviewFocus, want)
		}
	}
}

func TestReviewResumesANonRunningSession(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m.rows = []row{{name: "old", meta: manager.Meta{Name: "old", ClaudeSessionID: "abc"}}}
	m.sel = "old"

	next, cmd := m.reviewSelected()
	m = next.(Model)
	if m.reviewMode {
		t.Fatal("a non-running session must not open the review before it starts")
	}
	if !m.reviewPending {
		t.Fatal("s R on a non-running session must mark the review pending the resume")
	}
	if cmd == nil {
		t.Fatal("s R on a non-running session must return the resume command")
	}
}

func TestReviewEnterHidesSidebarAndLeaveRestores(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.sidebarHidden = false
	m, _ = step(t, m, key("esc"))

	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("R"))
	if !m.reviewMode {
		t.Fatal("s R must open the review screen")
	}
	if !m.sidebarHidden {
		t.Fatal("the review screen must hide the sidebar")
	}
	if m.focus != focusReview {
		t.Fatalf("the focus must move to the review screen, got %v", m.focus)
	}
	if m.output.Width != m.reviewExplainWidth() {
		t.Fatalf("the output pane must size to the explain width %d, got %d", m.reviewExplainWidth(), m.output.Width)
	}
	if view := visible(m.View()); !strings.Contains(view, "Explanation") {
		t.Fatalf("the review view must show the explanation pane:\n%s", view)
	}

	m, _ = step(t, m, key("esc"))
	if m.reviewMode {
		t.Fatal("esc must close the review screen")
	}
	if m.sidebarHidden {
		t.Fatal("leaving must restore the sidebar")
	}
}

func TestReviewMouseKeepsTheFocusOnTheScreen(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m, _ = step(t, m, key("esc"))
	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("R"))

	// A left click used to set m.focus by the region, which moved it off the
	// review screen and broke tab and esc.
	click := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 2, Y: 3}
	m, _ = step(t, m, click)
	if m.focus != focusReview {
		t.Fatalf("a click must keep the focus on the review screen, got %v", m.focus)
	}
	if m.reviewFocus != reviewDiff {
		t.Fatalf("a click on the left must focus the diff side, got %v", m.reviewFocus)
	}

	right := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: m.width - 3, Y: 3}
	m, _ = step(t, m, right)
	if m.reviewFocus != reviewExplain {
		t.Fatalf("a click on the right must focus the explanation, got %v", m.reviewFocus)
	}

	m, _ = step(t, m, key("esc"))
	if m.reviewMode {
		t.Fatal("esc must still close the review screen after a click")
	}
}

func TestReviewCapturesSequenceKeys(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m, _ = step(t, m, key("esc"))
	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("R"))
	if !m.reviewMode {
		t.Fatal("s R must open the review screen")
	}

	// s must not start a sequence in review, so it cannot open a panel or dialog
	// that moves the focus off the screen.
	m, _ = step(t, m, key("s"))
	if m.seq != nil {
		t.Fatal("the review screen must not start a key sequence")
	}
	m, _ = step(t, m, key("d"))
	if m.diffPanel {
		t.Fatal("s d must not open the diff panel from the review screen")
	}
	if m.focus != focusReview {
		t.Fatalf("the focus must stay on the review screen, got %v", m.focus)
	}
	m, _ = step(t, m, key("esc"))
	if m.reviewMode {
		t.Fatal("esc must still close the review screen")
	}
}
