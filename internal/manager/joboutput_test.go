package manager

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

// TestAJobCarriesItsCommandAndOutput drives the whole background job path
// against the fake binary. See docs/sessions.md.
func TestAJobCarriesItsCommandAndOutput(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "jobs")
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && len(snap.Jobs) == 1 && snap.Jobs[0].OutputPath != ""
	})
	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatal(err)
	}
	job := snap.Jobs[0]
	if !strings.Contains(job.Command, "echo first") {
		t.Fatalf("Command = %q", job.Command)
	}
	body, err := session.ReadOutput(job)
	if err != nil {
		t.Fatalf("ReadOutput: %v", err)
	}
	if strings.TrimSpace(body) != "first" {
		t.Fatalf("output = %q, want the interim line", body)
	}

	if err := m.Send(name, "finish job"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && len(snap.Jobs) == 1 && !snap.Jobs[0].Status.Running()
	})
	snap, _ = m.Snapshot(name)
	job = snap.Jobs[0]
	if job.Status != session.JobDone {
		t.Fatalf("Status = %s", job.Status)
	}
	if !strings.Contains(job.Summary, "exit code 0") {
		t.Fatalf("Summary = %q", job.Summary)
	}
	body, err = session.ReadOutput(job)
	if err != nil {
		t.Fatalf("ReadOutput: %v", err)
	}
	if !strings.Contains(body, "second") {
		t.Fatalf("output = %q, want the finished lines", body)
	}
}
