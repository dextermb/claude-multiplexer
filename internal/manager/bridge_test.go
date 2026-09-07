package manager

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func newBridgeManager(t *testing.T) *Manager {
	t.Helper()
	root := t.TempDir()
	m, err := New(Options{
		Root:        root,
		ClaudePath:  fakeClaude,
		Renderer:    render.Renderer{},
		ConfigPaths: []string{filepath.Join(root, "config.json")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m.Shutdown(ctx)
	})
	return m
}

func TestBridgeSettings(t *testing.T) {
	b := &bridge{m: newBridgeManager(t)}

	if _, err := b.SetConfig("editor", json.RawMessage(`"nvim"`), "boss"); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if _, _, err := b.UnsetConfig("editor", "boss"); err != nil {
		t.Fatalf("UnsetConfig: %v", err)
	}
	if _, err := b.SetEditor("code", nil, "boss"); err != nil {
		t.Fatalf("SetEditor: %v", err)
	}
	if _, _, err := b.UnsetEditor("both", "boss"); err != nil {
		t.Fatalf("UnsetEditor: %v", err)
	}
	rows := 10
	if _, err := b.SetBlockCap("tool", &rows, "boss"); err != nil {
		t.Fatalf("SetBlockCap: %v", err)
	}
	if _, _, err := b.UnsetBlockCap("tool", "boss"); err != nil {
		t.Fatalf("UnsetBlockCap: %v", err)
	}
	if b.ConfigPath().Target == "" {
		t.Fatal("ConfigPath has no target")
	}
}

func TestBridgeLayout(t *testing.T) {
	b := &bridge{m: newBridgeManager(t)}

	if _, err := b.SaveLayout("wide", mcp.LayoutDims{}, "boss"); err != nil {
		t.Fatalf("SaveLayout: %v", err)
	}
	if _, err := b.SetLayout("wide", mcp.ScopeAll, "boss"); err != nil {
		t.Fatalf("SetLayout: %v", err)
	}
	if _, err := b.Layouts(""); err != nil {
		t.Fatalf("Layouts: %v", err)
	}
	if _, _, err := b.UnsetLayout(mcp.ScopeAll, "boss"); err != nil {
		t.Fatalf("UnsetLayout: %v", err)
	}
	if _, _, err := b.DeleteLayout("wide", "boss"); err != nil {
		t.Fatalf("DeleteLayout: %v", err)
	}
	if _, err := b.SetLayout("wide", "bogus", "boss"); err != mcp.ErrBadScope {
		t.Fatalf("SetLayout bad scope: want ErrBadScope, got %v", err)
	}
}

func TestBridgeSchedule(t *testing.T) {
	m := newBridgeManager(t)
	b := &bridge{m: m}

	sched, err := b.CreateSchedule(mcp.ScheduleInput{Cron: "* * * * *", Dir: m.opts.Root, Prompt: "hi", Name: "job"}, "boss")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	newCron := "*/2 * * * *"
	if _, err := b.UpdateSchedule(sched.Name, mcp.ScheduleEdit{Cron: &newCron}, "boss"); err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}
	if len(b.ListSchedules()) != 1 {
		t.Fatalf("ListSchedules: want 1")
	}
	if _, err := b.SetScheduleEnabled(sched.Name, false, "boss"); err != nil {
		t.Fatalf("SetScheduleEnabled: %v", err)
	}
	if b.SchedulePath().Dir == "" {
		t.Fatal("SchedulePath is empty")
	}
	if changed, err := b.DeleteSchedule(sched.Name, "boss"); err != nil || !changed {
		t.Fatalf("DeleteSchedule: changed=%v err=%v", changed, err)
	}
}

func TestBridgeAPI(t *testing.T) {
	m := newBridgeManager(t)
	if err := m.StartMCP(); err != nil {
		t.Fatalf("StartMCP: %v", err)
	}
	b := &bridge{m: m}

	if _, err := b.CreateAPIAdmin(); err != nil {
		t.Fatalf("CreateAPIAdmin: %v", err)
	}
	client, secret, err := b.CreateAPIClient("bruno")
	if err != nil || secret == "" {
		t.Fatalf("CreateAPIClient: %v", err)
	}
	if len(b.ListAPIClients()) != 1 {
		t.Fatal("ListAPIClients: want 1")
	}
	if b.APIEndpoint().URL == "" {
		t.Fatal("APIEndpoint has no URL")
	}
	disabled := true
	if _, err := b.UpdateAPIClient(client.ClientID, nil, &disabled); err != nil {
		t.Fatalf("UpdateAPIClient: %v", err)
	}
	if _, err := b.RotateAPIClient(client.ClientID); err != nil {
		t.Fatalf("RotateAPIClient: %v", err)
	}
	if err := b.RevokeAPIClient(client.ClientID); err != nil {
		t.Fatalf("RevokeAPIClient: %v", err)
	}
	if _, err := b.RotateAPIAdmin(); err != nil {
		t.Fatalf("RotateAPIAdmin: %v", err)
	}
	if err := b.RevokeAPIAdmin(); err != nil {
		t.Fatalf("RevokeAPIAdmin: %v", err)
	}
}

func TestBridgeSession(t *testing.T) {
	m := newBridgeManager(t)
	b := &bridge{m: m}

	name, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: "s"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := b.SetTitle(name, "Title"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	if len(b.List()) == 0 {
		t.Fatal("List is empty")
	}
	if _, err := b.Jobs(name); err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	if _, err := b.TemplatePath(name); err != nil {
		t.Fatalf("TemplatePath: %v", err)
	}
	if _, err := b.StopJob(name, "no-such-job", "boss"); err == nil {
		t.Fatal("StopJob on an unknown job did not error")
	}
	if _, err := b.Create("/does/not/exist", "x", "boss"); err == nil {
		t.Fatal("Create on a missing directory did not error")
	}
}
