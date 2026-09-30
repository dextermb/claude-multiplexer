package manager

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestFailureRecordLandsOnTheMeta(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "failonstop")
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = m.Stop(ctx, name)

	path := filepath.Join(m.Root(), "sessions", name, "meta.json")
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(path)
		return err == nil && meta.Error != ""
	})

	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if !strings.Contains(meta.Error, "exit status 3") {
		t.Errorf("error = %q, want the exit status of the child", meta.Error)
	}
	if meta.ExitCode == nil || *meta.ExitCode != 3 {
		t.Errorf("exit code = %v, want 3", meta.ExitCode)
	}
	if len(meta.Stderr) == 0 || !strings.Contains(strings.Join(meta.Stderr, "\n"), "refused to shut down") {
		t.Errorf("stderr = %v, want the last lines of the child", meta.Stderr)
	}
}

func TestACleanStopLeavesNoFailureRecord(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	path := filepath.Join(m.Root(), "sessions", name, "meta.json")
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && !snap.State.Live()
	})
	time.Sleep(200 * time.Millisecond)

	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if meta.Error != "" || meta.ExitCode != nil || len(meta.Stderr) != 0 {
		t.Errorf("a clean stop wrote a failure record: %q %v %v", meta.Error, meta.ExitCode, meta.Stderr)
	}
}

func TestResumeClearsTheFailureRecord(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "failonstop")
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "again", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "first")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = m.Stop(ctx, name)
	waitFor(t, 10*time.Second, func() bool { return m.Remove(name) == nil })

	path := filepath.Join(m.Root(), "sessions", name, "meta.json")
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(path)
		return err == nil && meta.Error != ""
	})

	t.Setenv("FAKECLAUDE_MODE", "")
	stored, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if _, err := m.Resume(context.Background(), stored); err != nil {
		t.Fatalf("Resume: %v", err)
	}

	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if meta.Error != "" || meta.ExitCode != nil || len(meta.Stderr) != 0 {
		t.Errorf("the resume kept the record of the run that ended: %q %v %v", meta.Error, meta.ExitCode, meta.Stderr)
	}
}

func TestFailureOfKeepsTheLastStderrLines(t *testing.T) {
	lines := make([]string, failureStderrLines+5)
	for i := range lines {
		lines[i] = "line"
	}
	lines[len(lines)-1] = "last"

	got := failureOf(session.Snapshot{Err: errors.New("broke")}, lines)
	if got.Error != "broke" {
		t.Errorf("error = %q, want broke", got.Error)
	}
	if len(got.Stderr) != failureStderrLines {
		t.Errorf("kept %d lines, want %d", len(got.Stderr), failureStderrLines)
	}
	if got.Stderr[len(got.Stderr)-1] != "last" {
		t.Errorf("kept the wrong end of the buffer")
	}
	if got.ExitCode != nil {
		t.Errorf("exit code = %v, want none for an error that is not an exit", got.ExitCode)
	}
}

func TestFailureOfIsEmptyForACleanEnd(t *testing.T) {
	got := failureOf(session.Snapshot{}, []string{"noise"})
	if got.Error != "" || got.ExitCode != nil || got.Stderr != nil {
		t.Errorf("failureOf = %+v, want an empty record", got)
	}
}

func TestFailureOfReadsTheExitCode(t *testing.T) {
	cmd := exec.Command(fakeClaude)
	cmd.Env = append(os.Environ(), "FAKECLAUDE_MODE=crash")
	err := cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the fake child gave %v, want an exit error", err)
	}
	got := failureOf(session.Snapshot{Err: err}, nil)
	if got.ExitCode == nil || *got.ExitCode != 1 {
		t.Errorf("exit code = %v, want 1", got.ExitCode)
	}
}

func TestTheFailureRecordSurvivesASelfArchive(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "failonstop")
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.SetIdleAction(name, false, true); err != nil {
		t.Fatalf("SetIdleAction: %v", err)
	}
	if err := m.Send(name, "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	path := filepath.Join(m.Root(), "sessions", name, "meta.json")
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(path)
		return err == nil && meta.Archived && meta.Error != ""
	})

	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if !meta.Archived || meta.Error == "" {
		t.Fatalf("the archive and the failure record must both survive, got archived=%v error=%q",
			meta.Archived, meta.Error)
	}
}
