package tui

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func first(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func TestRealSessionThroughTheInterface(t *testing.T) {
	if os.Getenv("MULTIPLEXIER_REAL") == "" {
		t.Skip("set MULTIPLEXIER_REAL=1 to run one real session, which costs money")
	}
	mgr, err := manager.New(manager.Options{
		Root:         t.TempDir(),
		ClaudePath:   "claude",
		DefaultModel: "claude-haiku-4-5-20251001",
		Renderer:     render.Renderer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		mgr.Shutdown(ctx)
	}()

	sub := mgr.Subscribe(4096)
	defer sub.Close()

	m := New(Options{Manager: mgr, DefaultDir: "/tmp", DefaultModel: "claude-haiku-4-5-20251001"})
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 104, Height: 20})
	m, _ = step(t, m, key("esc"))

	name, err := mgr.Spawn(context.Background(), manager.Spec{Name: "real", Dir: "/tmp"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	m, _ = step(t, m, spawnedMsg{name: name})

	for _, r := range "Write two short sentences about mutexes." {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = step(t, m, key("enter"))

	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		if snap, err := mgr.Snapshot(name); err == nil && snap.Turns >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	m.refresh()
	m.rebuildOutput()

	snap, _ := mgr.Snapshot(name)
	if snap.Turns < 1 {
		t.Fatalf("no turn finished, state %v", snap.State)
	}
	var partials []string
	for len(sub.C) > 0 {
		if ev := <-sub.C; ev.Partial != "" {
			partials = append(partials, ev.Partial)
		}
	}
	if len(partials) == 0 {
		t.Error("no streaming text arrived")
	} else {
		t.Logf("partials: %d, first %q, last %q",
			len(partials), first(partials), partials[len(partials)-1])
	}

	t.Log("bar:\n" + m.barView())
	t.Log("output:\n" + strings.Join(render.Text(mgr.Lines(name)), "\n"))
	if snap.OutputTokens == 0 || snap.LastDuration == 0 || snap.Cost == 0 {
		t.Errorf("the bar numbers are incomplete: %+v", snap)
	}
}
