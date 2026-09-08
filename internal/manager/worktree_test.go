package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func toolUse(id, name, input string) session.Event {
	return session.Event{
		Kind: session.KindProtocol,
		Protocol: protocol.Event{
			Type: protocol.TypeAssistant,
			Message: &protocol.Message{
				Content: protocol.Content{{Type: "tool_use", ID: id, Name: name, Input: []byte(input)}},
			},
		},
	}
}

func toolResult(id, text string, isError bool) session.Event {
	return session.Event{
		Kind: session.KindProtocol,
		Protocol: protocol.Event{
			Type: protocol.TypeUser,
			Message: &protocol.Message{
				Content: protocol.Content{{
					Type:      "tool_result",
					ToolUseID: id,
					IsError:   isError,
					Content:   protocol.Content{{Type: "text", Text: text}},
				}},
			},
		},
	}
}

func TestTrackWorktreeSetsAndClearsTheWorkingDir(t *testing.T) {
	m := withMCP(t)
	dir := t.TempDir()
	work := filepath.Join(dir, ".claude", "worktrees", "feature")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	name, err := m.Spawn(context.Background(), Spec{Dir: dir, Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	enter := map[string]string{}
	exit := map[string]bool{}
	feed := func(ev session.Event) {
		ev.Session = name
		m.trackWorktree(ev, enter, exit)
	}

	feed(toolUse("t1", protocol.ToolEnterWorktree, "{}"))
	feed(toolResult("t1", "Created worktree at "+work+" on branch b. Now working there.", false))
	if got := m.WorkingDirs()[name]; got != work {
		t.Fatalf("after EnterWorktree, working dir = %q, want %q", got, work)
	}

	feed(toolUse("t2", protocol.ToolExitWorktree, `{"action":"remove"}`))
	feed(toolResult("t2", "Exited and removed worktree at "+work+". Session is now back in "+dir+".", false))
	if got := m.WorkingDirs()[name]; got != "" {
		t.Fatalf("after ExitWorktree, working dir = %q, want none", got)
	}
}

func TestTrackWorktreeFallsBackToTheInputPath(t *testing.T) {
	m := withMCP(t)
	dir := t.TempDir()
	work := filepath.Join(dir, ".claude", "worktrees", "existing")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	name, err := m.Spawn(context.Background(), Spec{Dir: dir, Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	enter := map[string]string{}
	exit := map[string]bool{}
	feed := func(ev session.Event) {
		ev.Session = name
		m.trackWorktree(ev, enter, exit)
	}

	feed(toolUse("t1", protocol.ToolEnterWorktree, `{"path":"`+work+`"}`))
	feed(toolResult("t1", "The session is now working in the worktree.", false))
	if got := m.WorkingDirs()[name]; got != work {
		t.Fatalf("working dir = %q, want the input path %q", got, work)
	}
}

func TestTrackWorktreeIgnoresAFailedEnter(t *testing.T) {
	m := withMCP(t)
	dir := t.TempDir()
	work := filepath.Join(dir, ".claude", "worktrees", "feature")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	name, err := m.Spawn(context.Background(), Spec{Dir: dir, Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	enter := map[string]string{}
	exit := map[string]bool{}
	feed := func(ev session.Event) {
		ev.Session = name
		m.trackWorktree(ev, enter, exit)
	}

	feed(toolUse("t1", protocol.ToolEnterWorktree, "{}"))
	feed(toolResult("t1", "Created worktree at "+work+" on branch b.", true))
	if got := m.WorkingDirs()[name]; got != "" {
		t.Fatalf("a failed EnterWorktree must not set a working dir, got %q", got)
	}
}
