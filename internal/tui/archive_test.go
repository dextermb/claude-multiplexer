package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func stoppedChild(name, dir, parent string) row {
	return row{name: name, dir: dir, parent: parent, label: "stored"}
}

func TestControlNameResolvesFromParentOrControlFlag(t *testing.T) {
	if got := controlName(controlRow("boss", "/work")); got != "boss" {
		t.Errorf("a control row resolves to %q, want boss", got)
	}
	if got := controlName(childRow("api", "/work", "boss")); got != "boss" {
		t.Errorf("a child row resolves to %q, want its parent boss", got)
	}
	if got := controlName(liveRow("lone", "/work", session.StateIdle)); got != "" {
		t.Errorf("a plain row resolves to %q, want the empty string", got)
	}
}

func TestAttachedToArchiveSkipsRunningAndArchived(t *testing.T) {
	rows := []row{
		controlRow("boss", "/work"),
		stoppedChild("api", "/work", "boss"),
		childRow("docs", "/work", "boss"),
		{name: "old", parent: "boss", archived: true, label: "archived"},
		stoppedChild("other", "/work", "someone-else"),
	}
	names, running := attachedToArchive(rows, "boss")
	if got := strings.Join(names, ","); got != "api" {
		t.Errorf("names = %q, want only the stopped child api", got)
	}
	if running != 1 {
		t.Errorf("running = %d, want the one live child docs", running)
	}
}

func TestArchiveAttachedNeedsAControlSession(t *testing.T) {
	m := groupModel(t, 30, liveRow("lone", "/work", session.StateIdle))
	m.sel = "lone"

	next, cmd := m.archiveAttached()
	m = next.(Model)
	if cmd != nil {
		t.Fatal("a row with no control session must archive nothing")
	}
	if !strings.Contains(m.errText, "control session") {
		t.Fatalf("errText = %q, want the control-session notice", m.errText)
	}
}

func TestArchiveAttachedDispatchesForTheGroup(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work"),
		stoppedChild("api", "/work", "boss"),
	)
	m.sel = "boss"

	next, cmd := m.archiveAttached()
	m = next.(Model)
	if cmd == nil {
		t.Fatal("s A on a control session with a stopped child must archive it")
	}
	if m.errText != "" {
		t.Fatalf("errText = %q, want none", m.errText)
	}
}

func TestArchiveAttachedResolvesTheControlFromAChild(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work"),
		stoppedChild("api", "/work", "boss"),
		stoppedChild("docs", "/work", "boss"),
	)
	m.sel = "api"

	_, cmd := m.archiveAttached()
	if cmd == nil {
		t.Fatal("s A on a child must archive the group of its control session")
	}
}

func TestArchiveAttachedReportsWhenEveryChildRuns(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work"),
		childRow("api", "/work", "boss"),
	)
	m.sel = "boss"

	next, cmd := m.archiveAttached()
	m = next.(Model)
	if cmd != nil {
		t.Fatal("a running child cannot be archived")
	}
	if !strings.Contains(m.errText, "still running") {
		t.Fatalf("errText = %q, want the running notice", m.errText)
	}
}

func TestArchivedManyMsgReportsTheCounts(t *testing.T) {
	m, _ := newTestModel(t, "")
	m, _ = step(t, m, archivedManyMsg{count: 2, skipped: 1})
	if !strings.Contains(m.status, "archived 2 attached sessions") {
		t.Errorf("status = %q, want the archived count", m.status)
	}
	if !strings.Contains(m.status, "skipped 1 running session") {
		t.Errorf("status = %q, want the skipped count", m.status)
	}
}
