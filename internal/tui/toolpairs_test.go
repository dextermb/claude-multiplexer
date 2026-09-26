package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func pairModel(t *testing.T) Model {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 120, 40)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	for len(m.sub.C) > 0 {
		m, _ = step(t, m, eventMsg(<-m.sub.C))
	}
	return m
}

func paneRows(m Model) []string {
	var rows []string
	for _, row := range strings.Split(ansi.Strip(m.outputText), "\n") {
		rows = append(rows, strings.TrimRight(row, " "))
	}
	return rows
}

func rowWith(rows []string, text string) int {
	for i, row := range rows {
		if strings.Contains(row, text) {
			return i
		}
	}
	return -1
}

func call(id, text string) render.Line {
	return render.Line{Class: render.ClassToolUse, Text: text, Tool: id}
}

func result(id, text string) render.Line {
	return render.Line{Class: render.ClassToolResult, Text: text, Tool: id}
}

func TestAFoldedResultBecomesANoteOnItsCallRow(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{result("t1", "← one\ntwo\nthree")})

	rows := paneRows(m)
	at := rowWith(rows, "READ")
	if at < 0 || !strings.HasSuffix(rows[at], "3 lines") {
		t.Fatalf("the call row does not carry the note:\n%s", strings.Join(rows, "\n"))
	}
	if rowWith(rows, "← one") >= 0 {
		t.Fatalf("a folded result must not draw its body:\n%s", strings.Join(rows, "\n"))
	}
}

func TestEnterOpensAFoldedResultUnderItsCall(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{call("t2", "→ Read b.go")})
	m.appendOutput([]render.Line{result("t1", "← alpha\nbeta")})
	m.appendOutput([]render.Line{result("t2", "← gamma")})

	var first int
	for index, id := range m.tools.folded {
		if id == "t1" {
			first = index
		}
	}
	m.toggleBlock(first)

	rows := paneRows(m)
	a, body, b := rowWith(rows, "a.go"), rowWith(rows, "← alpha"), rowWith(rows, "b.go")
	if !(a >= 0 && body == a+1 && b > body) {
		t.Fatalf("the body must open under its own call:\n%s", strings.Join(rows, "\n"))
	}
	if rowWith(rows, "show less") < 0 {
		t.Fatalf("an open result needs its close marker:\n%s", strings.Join(rows, "\n"))
	}
}

func TestTheCursorLandsOnTheCallRowOfAFoldedResult(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{result("t1", "← one\ntwo")})
	m.resetBlockCursor()

	styled := strings.Split(m.outputText, "\n")[rowWith(paneRows(m), "READ")]
	if styled != invertStyle.Render(ansi.Strip(styled)) {
		t.Fatalf("the call row under the cursor must invert: %q", styled)
	}
}

func TestABashResultKeepsItsBody(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Bash go test ./...")})
	m.appendOutput([]render.Line{result("t1", "← ok  the/package  0.4s")})

	rows := paneRows(m)
	if rowWith(rows, "BASH") < 0 || rowWith(rows, "← ok") < 0 {
		t.Fatalf("a bash call keeps its result body:\n%s", strings.Join(rows, "\n"))
	}
}

func TestAnErrorResultMarksTheCallAndKeepsItsBody(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{result("t1", "←! no such file")})

	rows := paneRows(m)
	if at := rowWith(rows, "READ"); at < 0 || !strings.HasSuffix(rows[at], "× error") {
		t.Fatalf("the call row must say error:\n%s", strings.Join(rows, "\n"))
	}
	if rowWith(rows, "←! no such file") < 0 {
		t.Fatalf("an error keeps its body:\n%s", strings.Join(rows, "\n"))
	}
}

func TestAnEditCallShowsItsNote(t *testing.T) {
	m := pairModel(t)
	edit := call("t1", "→ Edit a.go")
	edit.Note = "+3 −12"
	m.appendOutput([]render.Line{edit})
	m.appendOutput([]render.Line{result("t1", "← The file a.go has been updated.")})

	rows := paneRows(m)
	if at := rowWith(rows, "EDIT"); at < 0 || !strings.HasSuffix(rows[at], "+3 −12") {
		t.Fatalf("the edit row must carry its note:\n%s", strings.Join(rows, "\n"))
	}
}

func TestARebuildDrawsThePairsAsTheAppendsDid(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{call("t2", "→ Bash ls")})
	m.appendOutput([]render.Line{result("t1", "← one\ntwo")})
	m.appendOutput([]render.Line{result("t2", "← a\nb")})
	appended := m.outputText

	m.redrawBlocks()
	if m.outputText != appended {
		t.Fatalf("a redraw changed the pane:\n%s\n---\n%s", ansi.Strip(appended), ansi.Strip(m.outputText))
	}
}

func TestTheRawViewDoesNotPairTools(t *testing.T) {
	m := pairModel(t)
	m.showRaw = true
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{result("t1", "← one\ntwo")})

	rows := paneRows(m)
	if rowWith(rows, "→ Read a.go") < 0 || rowWith(rows, "← one") < 0 {
		t.Fatalf("the raw view shows the lines as written:\n%s", strings.Join(rows, "\n"))
	}
}

func TestANewCallDropsTheOldResultOfItsID(t *testing.T) {
	m := pairModel(t)
	m.appendOutput([]render.Line{call("t1", "→ Read a.go")})
	m.appendOutput([]render.Line{result("t1", "← one\ntwo")})
	m.appendOutput([]render.Line{call("t1", "→ Read b.go")})

	rows := paneRows(m)
	if at := rowWith(rows, "b.go"); at < 0 || strings.HasSuffix(rows[at], "2 lines") {
		t.Fatalf("the new call must not take the old result:\n%s", strings.Join(rows, "\n"))
	}
}
