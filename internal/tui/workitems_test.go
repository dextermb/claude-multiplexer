package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func workItemRow(name, dir, status string) row {
	item := liveRow(name, dir, session.StateIdle)
	key := name + "-1"
	item.title = key
	item.workItem = manager.WorkItemBadge{Provider: "linear", Key: key, Status: status}
	return item
}

func TestWorkItemSessionsGroupByStatus(t *testing.T) {
	m := groupModel(t, 40,
		workItemRow("alpha", "/work/repo", "In Review"),
		liveRow("plain", "/work/other", session.StateIdle),
		workItemRow("beta", "/work/repo", "In Progress"),
		workItemRow("gamma", "/work/repo", "In Review"),
	)

	labels := groupLabelsOf(m)
	want := []string{"In Progress", "In Review", "other"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("groups = %v, want the status groups first, in order, then the directory group", labels)
	}

	if got := rowNames(m); strings.Join(got, ",") != "beta,alpha,gamma,plain" {
		t.Fatalf("rows = %v, want each status group to hold its own sessions", got)
	}

	view := visible(m.sidebarView())
	if !strings.Contains(view, "alpha-1") {
		t.Fatalf("a renamed work-item session must show its key as its name:\n%s", view)
	}
	if !strings.Contains(view, "In Review") {
		t.Fatalf("a status group must show the status as its header:\n%s", view)
	}
}

func TestStatusBarShowsWorkItemNextToEffort(t *testing.T) {
	item := row{mode: "default", effort: "high"}
	item.workItem = manager.WorkItemBadge{Key: "GIM-1", Status: "In Review"}

	full := barDetails(item)[0]
	effortAt := slices.Index(full, "high effort")
	statusAt := slices.Index(full, "In Review")
	if effortAt < 0 || statusAt < 0 {
		t.Fatalf("bar = %v, want the effort and the work-item status", full)
	}
	if statusAt < effortAt {
		t.Fatalf("bar = %v, want the work-item status after the effort", full)
	}
}

func TestBarWorkItemFallsBackToKey(t *testing.T) {
	linked := row{}
	linked.workItem = manager.WorkItemBadge{Key: "GIM-9"}
	if got := barWorkItem(linked); got != "GIM-9" {
		t.Fatalf("barWorkItem = %q, want the key when no status is known", got)
	}
	if got := barWorkItem(row{}); got != "" {
		t.Fatalf("barWorkItem = %q, want empty for a session with no link", got)
	}
}
