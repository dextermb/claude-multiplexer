package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestOrderJobsRunningFirst(t *testing.T) {
	jobs := []session.Job{
		{ID: "a", Status: session.JobDone},
		{ID: "b", Status: session.JobRunning},
		{ID: "c", Status: session.JobKilled},
		{ID: "d", Status: session.JobRunning},
	}
	got := orderJobs(jobs)
	order := []string{got[0].ID, got[1].ID, got[2].ID, got[3].ID}
	want := []string{"b", "d", "a", "c"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestJobsModalViewListsEveryJob(t *testing.T) {
	jobs := []session.Job{
		{ID: "j1", Description: "build the binary", Status: session.JobRunning},
		{ID: "j2", Description: "run the tests", Status: session.JobDone},
	}
	view := newJobsModal("api", jobs, 80, 24).View(80, 24)
	for _, want := range []string{"Background jobs", "api", "build the binary", "run the tests", "running", "done"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not show %q:\n%s", want, view)
		}
	}
}

func TestJobsModalEmptyState(t *testing.T) {
	view := newJobsModal("api", nil, 80, 24).View(80, 24)
	if !strings.Contains(view, "No background jobs") {
		t.Fatalf("empty modal must say so:\n%s", view)
	}
}

func TestSessionRowShowsJobBadge(t *testing.T) {
	m := Model{}
	got := m.sessionRow(row{name: "api", live: true, state: session.StateBusy, jobs: 2})
	if !strings.Contains(got, "⚙2") {
		t.Fatalf("row must show the job badge:\n%q", got)
	}
}
