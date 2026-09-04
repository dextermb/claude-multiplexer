package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func outputFile(t *testing.T, id, body string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+".output")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func modalWith(t *testing.T, jobs []session.Job) *jobsModal {
	t.Helper()
	m := newJobsModal("api", jobs, 80, 24)
	m.View(80, 24)
	return m
}

func TestJobsListMarksTheCursor(t *testing.T) {
	m := modalWith(t, []session.Job{
		{ID: "j1", Description: "build the binary", Status: session.JobRunning},
		{ID: "j2", Description: "run the tests", Status: session.JobDone},
	})
	view := visible(m.View(80, 24))
	if !strings.Contains(view, "▸") {
		t.Fatalf("the list must mark the cursor:\n%s", view)
	}
	if !strings.Contains(view, "enter open") {
		t.Fatalf("the hint must name enter:\n%s", view)
	}
}

func TestEnterOpensTheJobUnderTheCursor(t *testing.T) {
	path := outputFile(t, "j2", "first\nsecond\n")
	m := modalWith(t, []session.Job{
		{ID: "j1", Description: "build the binary", Status: session.JobRunning},
		{ID: "j2", Description: "run the tests", Command: "go test ./...",
			OutputPath: path, Status: session.JobDone},
	})
	m.Update(key("down"))
	if _, cmd := m.Update(key("enter")); cmd == nil {
		t.Fatal("opening a job must start the refresh tick")
	}
	view := visible(m.View(80, 24))
	for _, want := range []string{"run the tests", "go test ./...", "first", "second", "esc back"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the detail must show %q:\n%s", want, view)
		}
	}
}

func TestEscStepsBackToTheList(t *testing.T) {
	m := modalWith(t, []session.Job{{ID: "j1", Description: "build", Status: session.JobRunning}})
	m.Update(key("enter"))
	open, _ := m.Update(key("esc"))
	if !open {
		t.Fatal("esc in the detail must step back, not close")
	}
	if m.showing {
		t.Fatal("the detail must be closed")
	}
	if open, _ := m.Update(key("esc")); open {
		t.Fatal("esc in the list must close the dialog")
	}
}

func TestAJobWithNoOutputSaysSo(t *testing.T) {
	m := modalWith(t, []session.Job{{ID: "j1", Description: "build", Status: session.JobRunning}})
	m.Update(key("enter"))
	if view := visible(m.View(80, 24)); !strings.Contains(view, "Waiting for the output file") {
		t.Fatalf("a job with no path must say so:\n%s", view)
	}

	empty := outputFile(t, "j2", "")
	m2 := modalWith(t, []session.Job{{ID: "j2", Description: "build", OutputPath: empty, Status: session.JobRunning}})
	m2.Update(key("enter"))
	if view := visible(m2.View(80, 24)); !strings.Contains(view, "No output yet") {
		t.Fatalf("an empty file must say so:\n%s", view)
	}
}

func TestABadOutputPathReportsTheReason(t *testing.T) {
	m := modalWith(t, []session.Job{{ID: "j1", Description: "build", OutputPath: "/etc/passwd", Status: session.JobDone}})
	m.Update(key("enter"))
	if view := visible(m.View(80, 24)); !strings.Contains(view, "could not be read") {
		t.Fatalf("a path outside the job must be refused:\n%s", view)
	}
}

func TestNewOutputReachesAnOpenJob(t *testing.T) {
	path := outputFile(t, "j1", "first\n")
	job := session.Job{ID: "j1", Description: "build", OutputPath: path, Status: session.JobRunning}
	m := modalWith(t, []session.Job{job})
	m.Update(key("enter"))

	if err := os.WriteFile(path, []byte("first\nsecond\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, cmd := m.Update(jobTickMsg{}); cmd == nil {
		t.Fatal("the tick must schedule the next one")
	}
	if view := visible(m.View(80, 24)); !strings.Contains(view, "second") {
		t.Fatalf("new output must reach the open job:\n%s", view)
	}
}

func TestTheTickStopsWhenTheDetailCloses(t *testing.T) {
	m := modalWith(t, []session.Job{{ID: "j1", Description: "build", Status: session.JobRunning}})
	if _, cmd := m.Update(jobTickMsg{}); cmd != nil {
		t.Fatal("a closed detail must not keep the tick alive")
	}
}

// A job that stops moves down the list, so the cursor must follow its job.
func TestTheCursorHoldsItsJobWhenTheOrderChanges(t *testing.T) {
	m := modalWith(t, []session.Job{
		{ID: "j1", Description: "one", Status: session.JobRunning},
		{ID: "j2", Description: "two", Status: session.JobRunning},
	})
	m.Update(key("down"))
	if got := m.jobs[m.cursor].ID; got != "j2" {
		t.Fatalf("cursor on %q, want j2", got)
	}
	m.setJobs([]session.Job{
		{ID: "j1", Description: "one", Status: session.JobRunning},
		{ID: "j2", Description: "two", Status: session.JobDone},
	})
	if got := m.jobs[m.cursor].ID; got != "j2" {
		t.Fatalf("cursor moved to %q, want j2", got)
	}
	if m.cursor != 1 {
		t.Fatalf("the finished job must sit last, cursor = %d", m.cursor)
	}
}

func TestTheDetailFitsANarrowPane(t *testing.T) {
	path := outputFile(t, "j1", "hello\n")
	m := newJobsModal("api", []session.Job{
		{ID: "j1", Description: "build", Command: "go build ./...", OutputPath: path, Status: session.JobDone},
	}, 44, 20)
	m.View(44, 20)
	m.Update(key("enter"))
	for _, line := range strings.Split(visible(m.View(44, 20)), "\n") {
		if width := len([]rune(line)); width > 44 {
			t.Fatalf("line of %d runes in a pane of 44: %q", width, line)
		}
	}
}
