package manager

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// TestRealSessionCallsTheTools proves the MCP server against the installed
// Claude Code. It costs money, so it runs only on request. See docs/mcp.md.
func TestRealSessionCallsTheTools(t *testing.T) {
	if os.Getenv("MULTIPLEXIER_REAL") == "" {
		t.Skip("set MULTIPLEXIER_REAL=1 to run one real session, which costs money")
	}
	m, err := New(Options{
		Root:         t.TempDir(),
		ClaudePath:   "claude",
		DefaultModel: "claude-haiku-4-5-20251001",
		Renderer:     render.Renderer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.Shutdown(ctx)
	}()
	if err := m.StartMCP(); err != nil {
		t.Fatalf("StartMCP: %v", err)
	}

	ctx := context.Background()
	name, err := m.Spawn(ctx, Spec{Dir: t.TempDir(), Name: "probe", Control: true})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := m.Send(name, `Call the tool mcp__mux__rename_session with the title "Billing rewrite". Answer with one word when it is done.`); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 120*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Title == "Billing rewrite"
	})

	waitFor(t, 120*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 1
	})

	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Logf("state=%s turns=%d cost=%.4f", snap.State, snap.Turns, snap.Cost)
	if snap.State != session.StateIdle {
		t.Fatalf("the child did not continue after the tool: %s", snap.State)
	}
	for _, line := range m.Lines(name) {
		if strings.HasPrefix(line.Text, "●") || strings.Contains(line.Text, "mux") {
			t.Logf("line: %s", line.Text)
		}
	}
	if status := mcpStatus(t, m, name); status != "connected" {
		t.Fatalf("the child reported the mux server as %q", status)
	}
}

// mcpStatus reads what the init event said about the mux server.
func mcpStatus(t *testing.T, m *Manager, name string) string {
	t.Helper()
	file, err := os.Open(transcriptPath(m.Root(), name))
	if err != nil {
		t.Fatalf("transcript: %v", err)
	}
	defer file.Close()
	reader := protocol.NewReader(file)
	for {
		ev, err := reader.Next()
		if err != nil {
			return "no init event"
		}
		if !ev.IsInit() || ev.Init == nil {
			continue
		}
		for _, server := range ev.Init.MCPServers {
			if server.Name == "mux" {
				return server.Status
			}
		}
		return "not listed"
	}
}
