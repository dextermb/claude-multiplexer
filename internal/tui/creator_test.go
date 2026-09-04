package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func controlRow(name, dir string) row {
	return row{name: name, dir: dir, live: true, state: session.StateIdle, control: true, label: "idle"}
}

func childRow(name, dir, parent string) row {
	return row{name: name, dir: dir, live: true, state: session.StateIdle, parent: parent, label: "idle"}
}

func TestACreatedSessionJoinsTheGroupOfItsCreator(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work/one"),
		childRow("api", "/work/two", "boss"),
		liveRow("docs", "/work/two", session.StateIdle),
	)

	if len(m.groups) != 2 {
		t.Fatalf("groups = %d, want the creator group and the directory group", len(m.groups))
	}
	if key := m.groups[0].key; key != byPrefix+"boss" {
		t.Fatalf("the first group is %q, want the group of boss", key)
	}
	if !m.groups[0].creator {
		t.Fatal("the group of a control session must be marked as one")
	}
	if m.groups[0].count != 2 {
		t.Fatalf("count = %d, want the control session and the one it created", m.groups[0].count)
	}
	if got := strings.Join(rowNames(m), ","); got != "boss,api,docs" {
		t.Fatalf("rows = %s, want the creator group first", got)
	}
	if key := m.groups[1].key; key != dirPrefix+"/work/two" {
		t.Fatalf("the second group is %q, want the directory of docs", key)
	}
}

func TestAControlSessionWithNoChildrenKeepsItsDirectory(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work/one"),
		liveRow("api", "/work/one", session.StateIdle),
	)

	if len(m.groups) != 1 {
		t.Fatalf("groups = %d, want the one directory group", len(m.groups))
	}
	if key := m.groups[0].key; key != dirPrefix+"/work/one" {
		t.Fatalf("group = %q, want the directory group", key)
	}
}

func TestTheControlSessionHeadsItsGroupWhateverItsState(t *testing.T) {
	m := groupModel(t, 30,
		childRow("api", "/work/two", "boss"),
		row{name: "boss", dir: "/work/one", label: "stored"},
	)

	if got := strings.Join(rowNames(m), ","); got != "boss,api" {
		t.Fatalf("rows = %s, want the stopped control session first", got)
	}
}

func TestTheHeaderOfACreatorGroupNamesTheControlSession(t *testing.T) {
	m := groupModel(t, 30,
		row{name: "boss", title: "Orchestrator", dir: "/work/one", live: true, state: session.StateIdle, control: true},
		childRow("api", "/work/two", "boss"),
	)

	header := visible(m.groupHeader(m.groups[0]))
	if !strings.Contains(header, controlMark) {
		t.Errorf("header = %q, want the control mark", header)
	}
	if !strings.Contains(header, "Orchestrator") {
		t.Errorf("header = %q, want the title of the control session", header)
	}
}

func TestACreatorGroupFoldsLikeAnyOther(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work/one"),
		childRow("api", "/work/two", "boss"),
		liveRow("docs", "/work/three", session.StateIdle),
	)

	m.setFold(byPrefix+"boss", true)

	view := visible(m.sidebarView())
	if strings.Contains(view, "api") {
		t.Errorf("the folded creator group still shows its rows:\n%s", view)
	}
	if !strings.Contains(view, "boss") {
		t.Errorf("the folded creator group must keep its header:\n%s", view)
	}
	if m.sel != "docs" {
		t.Fatalf("selected = %q, want the row of the next group", m.sel)
	}
}

func TestTheSidebarReadsTheCreatorOfALiveSession(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	m = spawn(t, m, mgr, "boss", t.TempDir())
	name, err := mgr.Spawn(context.Background(), manager.Spec{Name: "api", Dir: t.TempDir(), Parent: "boss"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	m, _ = step(t, m, spawnedMsg{name: name})

	if len(m.groups) != 1 {
		t.Fatalf("groups = %d, want the creator to hold both sessions", len(m.groups))
	}
	if key := m.groups[0].key; key != byPrefix+"boss" {
		t.Fatalf("group = %q, want the group of boss", key)
	}
	if got := strings.Join(rowNames(m), ","); got != "boss,api" {
		t.Fatalf("rows = %s, want boss first", got)
	}
}

func TestTheHeadRowDropsTheControlMark(t *testing.T) {
	m := groupModel(t, 30,
		controlRow("boss", "/work/one"),
		childRow("api", "/work/two", "boss"),
	)

	if head := visible(m.sessionRow(m.rows[0])); strings.Contains(head, controlMark) {
		t.Errorf("row = %q, want no control mark, because the header carries it", head)
	}

	alone := groupModel(t, 30, controlRow("solo", "/work/one"))
	if row := visible(alone.sessionRow(alone.rows[0])); !strings.Contains(row, controlMark) {
		t.Errorf("row = %q, want the control mark on a row with no group of its own", row)
	}
}
