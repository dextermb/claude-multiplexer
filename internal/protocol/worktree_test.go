package protocol

import "testing"

func TestEnterWorktreeReadsThePath(t *testing.T) {
	block := Block{Type: "tool_use", Name: "EnterWorktree", Input: []byte(`{"path":".claude/worktrees/x"}`)}
	got, ok := block.EnterWorktree()
	if !ok || got != ".claude/worktrees/x" {
		t.Fatalf("EnterWorktree = %q, %v", got, ok)
	}
}

func TestEnterWorktreeAllowsNoPath(t *testing.T) {
	block := Block{Type: "tool_use", Name: "EnterWorktree", Input: []byte(`{}`)}
	got, ok := block.EnterWorktree()
	if !ok || got != "" {
		t.Fatalf("EnterWorktree = %q, %v; want empty path and ok", got, ok)
	}
}

func TestEnterWorktreeRejectsAnotherTool(t *testing.T) {
	block := Block{Type: "tool_use", Name: "Read", Input: []byte(`{"path":"x"}`)}
	if _, ok := block.EnterWorktree(); ok {
		t.Fatal("only EnterWorktree names a worktree")
	}
}

func TestExitWorktreeMatchesTheTool(t *testing.T) {
	if !(Block{Type: "tool_use", Name: "ExitWorktree"}).ExitWorktree() {
		t.Fatal("ExitWorktree must match its tool")
	}
	if (Block{Type: "tool_use", Name: "EnterWorktree"}).ExitWorktree() {
		t.Fatal("EnterWorktree is not ExitWorktree")
	}
}

// The text is the one the EnterWorktree tool returns; see docs/mcp/tools.md.
func TestWorktreePathReadsTheRealResult(t *testing.T) {
	const text = "Created worktree at /Users/x/repo/.claude/worktrees/keen-squishing-marshmallow " +
		"on branch worktree-keen-squishing-marshmallow. The session is now working in the worktree."
	want := "/Users/x/repo/.claude/worktrees/keen-squishing-marshmallow"
	if got := WorktreePath(text); got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

// The text is the one EnterWorktree returns for a worktree it enters by path;
// see docs/mcp/tools.md.
func TestWorktreePathReadsAnEnteredWorktree(t *testing.T) {
	const text = "Entered worktree at /Users/x/repo/.claude/worktrees/probe on branch worktree-probe. " +
		"The session is now working in the worktree."
	want := "/Users/x/repo/.claude/worktrees/probe"
	if got := WorktreePath(text); got != want {
		t.Fatalf("WorktreePath = %q, want %q", got, want)
	}
}

func TestWorktreePathIsEmptyWithoutTheMarker(t *testing.T) {
	for _, text := range []string{"", "Exited the worktree.", "Created worktree at /tmp/x"} {
		if got := WorktreePath(text); got != "" {
			t.Fatalf("WorktreePath(%q) = %q, want empty", text, got)
		}
	}
}
