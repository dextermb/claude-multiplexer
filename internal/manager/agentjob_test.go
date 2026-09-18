package manager

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// TestAgentTurnsGoToTheJobNotThePane drives the local-agent path against the
// fake binary. The agent's turns land in the job output file, and the pane keeps
// only the lifecycle line and the parent turn. See docs/sessions/jobs.md.
func TestAgentTurnsGoToTheJobNotThePane(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "agent", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "agent")

	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && len(snap.Jobs) == 1 && snap.Jobs[0].OutputPath != ""
	})

	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatal(err)
	}
	job := snap.Jobs[0]
	if job.TaskType != "local_agent" {
		t.Fatalf("TaskType = %q, want local_agent", job.TaskType)
	}

	body, err := session.ReadOutput(job)
	if err != nil {
		t.Fatalf("ReadOutput: %v", err)
	}
	for _, want := range []string{"Read render.go", "40 lines"} {
		if !strings.Contains(body, want) {
			t.Fatalf("the job output has no %q:\n%s", want, body)
		}
	}

	pane := strings.Join(render.Text(m.Lines(name)), "\n")
	if !strings.Contains(pane, "⚙ started") || !strings.Contains(pane, "echo: agent") {
		t.Fatalf("the pane lost the lifecycle line or the parent turn:\n%s", pane)
	}
	for _, hidden := range []string{"Read render.go", "40 lines"} {
		if strings.Contains(pane, hidden) {
			t.Fatalf("the pane shows an agent turn %q:\n%s", hidden, pane)
		}
	}

	retire(t, m, name)
	replay := strings.Join(render.Text(m.Replay(name)), "\n")
	if !strings.Contains(replay, "⚙ started") || !strings.Contains(replay, "echo: agent") {
		t.Fatalf("the replay lost the lifecycle line or the parent turn:\n%s", replay)
	}
	for _, hidden := range []string{"Read render.go", "40 lines"} {
		if strings.Contains(replay, hidden) {
			t.Fatalf("the replay shows an agent turn %q:\n%s", hidden, replay)
		}
	}
}
