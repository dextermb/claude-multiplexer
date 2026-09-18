package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// TestDeriveSidebarFromSnapshots derives the whole sidebar view from fixed
// snapshots and metas, with no manager. The seam makes this possible: the fold
// takes plain data, so a test reads the rows, the groups, and the lines it gives.
func TestDeriveSidebarFromSnapshots(t *testing.T) {
	in := sidebarInputs{
		snapshots: []session.Snapshot{
			{Name: "api", Dir: "/work/one", State: session.StateBusy},
			{Name: "web", Dir: "/work/two", State: session.StateIdle},
		},
		stored: []manager.Meta{
			{Name: "notes", Dir: "/work/one"},
		},
		grants: map[string]bool{"api": true},
		roots: map[string]string{
			"/work/one": "/work/one",
			"/work/two": "/work/two",
		},
		folded: map[string]bool{},
	}

	view := deriveSidebar(in)

	names := make([]string, 0, len(view.rows))
	for _, item := range view.rows {
		names = append(names, item.name)
	}
	if strings.Join(names, ",") != "api,notes,web" {
		t.Fatalf("rows = %v, want the busy group first, then its own order", names)
	}

	if len(view.groups) != 2 {
		t.Fatalf("groups = %d, want one for each directory", len(view.groups))
	}
	if view.groups[0].key != dirPrefix+"/work/one" {
		t.Fatalf("first group = %q, want the group with the busy session", view.groups[0].key)
	}
	if view.groups[0].count != 2 {
		t.Fatalf("count = %d, want the live and the stored row", view.groups[0].count)
	}

	for _, item := range view.rows {
		if item.name == "api" && !item.control {
			t.Fatal("the grant did not reach the row: api must be a control session")
		}
	}

	headers := 0
	for _, line := range view.lines {
		if line.header() {
			headers++
		}
	}
	if headers != 2 {
		t.Fatalf("headers = %d, want one per group", headers)
	}
}

// TestDeriveSidebarFiltersBySearch keeps only the rows the needle matches, and
// drops the archived metas unless showArchived is set.
func TestDeriveSidebarFiltersBySearch(t *testing.T) {
	in := sidebarInputs{
		snapshots: []session.Snapshot{
			{Name: "api", Dir: "/work/one", State: session.StateIdle},
			{Name: "web", Dir: "/work/two", State: session.StateIdle},
		},
		stored: []manager.Meta{
			{Name: "old", Dir: "/work/one", Archived: true},
		},
		roots: map[string]string{
			"/work/one": "/work/one",
			"/work/two": "/work/two",
		},
		needle: "api",
		folded: map[string]bool{},
	}

	view := deriveSidebar(in)
	if len(view.rows) != 1 || view.rows[0].name != "api" {
		t.Fatalf("rows = %v, want only the row that matches the needle", view.rows)
	}

	in.needle = ""
	if got := deriveSidebar(in); len(got.rows) != 2 {
		t.Fatalf("rows = %d, want the archived meta hidden without showArchived", len(got.rows))
	}

	in.showArchived = true
	if got := deriveSidebar(in); len(got.rows) != 3 {
		t.Fatalf("rows = %d, want the archived meta shown with showArchived", len(got.rows))
	}
}
