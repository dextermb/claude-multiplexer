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

// TestRealSessionRunsABackgroundJob answers which id links a BashOutput call to
// a job. It costs money, so it runs only on request. See docs/plans/job-output.md.
func TestRealSessionRunsABackgroundJob(t *testing.T) {
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

	ctx := context.Background()
	name, err := m.Spawn(ctx, Spec{
		Dir:          t.TempDir(),
		AllowedTools: []string{"Bash", "BashOutput"},
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	const prompt = `Run this exact command with the Bash tool and run_in_background set to true: ` +
		`sh -c 'echo first; sleep 3; echo second'. ` +
		`Then answer with one word: started. Do not read the output.`
	if err := m.Send(name, prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 180*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 1
	})
	// A task_notification may arrive after the turn ends, so give it room.
	time.Sleep(20 * time.Second)

	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Logf("state=%s turns=%d jobs=%d", snap.State, snap.Turns, len(snap.Jobs))
	if len(snap.Jobs) != 1 {
		dumpJobEvents(t, m, name)
		t.Fatalf("jobs = %d, want 1", len(snap.Jobs))
	}
	job := snap.Jobs[0]
	t.Logf("job id=%q status=%s command=%q", job.ID, job.Status, job.Command)
	t.Logf("summary=%q", job.Summary)
	t.Logf("path=%q", job.OutputPath)
	if job.Command == "" {
		t.Error("the job did not keep its command")
	}
	if job.OutputPath == "" {
		t.Fatal("the job did not learn its output path")
	}
	body, err := session.ReadOutput(job)
	if err != nil {
		t.Fatalf("ReadOutput: %v", err)
	}
	t.Logf("output:\n%s", body)
	if !strings.Contains(body, "first") || !strings.Contains(body, "second") {
		t.Errorf("output = %q, want both lines", body)
	}
	if job.Status != session.JobDone {
		t.Errorf("Status = %s, want done", job.Status)
	}
	if !strings.Contains(job.Summary, "exit code 0") {
		t.Errorf("Summary = %q", job.Summary)
	}
}

// dumpJobEvents prints every task event and every Bash or BashOutput block in
// the transcript, so the probe can be read by eye.
func dumpJobEvents(t *testing.T, m *Manager, name string) {
	t.Helper()
	file, err := os.Open(transcriptPath(m.Root(), name))
	if err != nil {
		t.Fatalf("transcript: %v", err)
	}
	defer file.Close()
	results := map[string]string{}
	var order []string
	reader := protocol.NewReader(file)
	for {
		ev, err := reader.Next()
		if err != nil {
			break
		}
		if ev.Task != nil {
			t.Logf("%s %s: %s", ev.Type, ev.Subtype, ev.Raw)
			continue
		}
		if ev.Message == nil {
			continue
		}
		for _, block := range ev.Message.Content {
			switch block.Type {
			case "tool_use":
				if block.Name != "Bash" && block.Name != "BashOutput" {
					continue
				}
				t.Logf("tool_use %s id=%q input=%s", block.Name, block.ID, block.Input)
				order = append(order, block.ID)
			case "tool_result":
				results[block.ToolUseID] = block.Content.Text()
			}
		}
	}
	for _, id := range order {
		t.Logf("tool_result for %q:\n%s", id, results[id])
	}
}
