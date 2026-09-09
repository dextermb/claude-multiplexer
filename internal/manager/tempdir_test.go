package manager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

// tempSession creates a hosted session in a temporary directory the host makes,
// the way the peer listener does, and returns its name and that directory.
func tempSession(t *testing.T, m *Manager) (string, string) {
	t.Helper()
	name, err := m.apiSessions("c1", "client").Create(mcp.CreateInput{TempDir: true, Hosted: true}, "client")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(snap.Dir), "cmux-session-") {
		t.Errorf("dir = %q, want a cmux-session temp directory", snap.Dir)
	}
	if info, err := os.Stat(snap.Dir); err != nil || !info.IsDir() {
		t.Fatalf("the temp directory was not made: %v", err)
	}
	return name, snap.Dir
}

func TestTempDirIsRemovedOnRemove(t *testing.T) {
	m := newTestManager(t)
	name, dir := tempSession(t, m)

	// One turn, so the session is kept and not discarded as empty.
	runOneTurn(t, m, name, "hello")
	retire(t, m, name) // Stop then Remove

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the temp directory survived Remove: %v", err)
	}
}

func TestTempDirIsRemovedOnZeroTurnDiscard(t *testing.T) {
	m := newTestManager(t)
	name, dir := tempSession(t, m)

	// No prompt, so the session ends with no turns and is discarded, taking its
	// temp directory with it.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		_, err := os.Stat(dir)
		return os.IsNotExist(err)
	})
}

func TestEvictKeepsTheTempDir(t *testing.T) {
	m := newTestManager(t)
	name, dir := tempSession(t, m)
	runOneTurn(t, m, name, "hello")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && !snap.State.Live()
	})

	// evict is the resume path: it drops the session but keeps the directory, so
	// the re-spawn still has it.
	if _, err := m.evict(name); err != nil {
		t.Fatalf("evict: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("evict removed the temp directory, which a resume needs: %v", err)
	}
	_ = os.RemoveAll(dir)
}
