package protocol

import (
	"encoding/json"
	"strings"
)

const (
	ToolEnterWorktree = "EnterWorktree"
	ToolExitWorktree  = "ExitWorktree"
)

type enterWorktreeInput struct {
	Path string `json:"path"`
}

// EnterWorktree reads the optional path from an EnterWorktree tool_use block.
// The second result is false for another tool. The path is empty when the agent
// makes a new worktree without one. See docs/mcp/tools.md.
func (b Block) EnterWorktree() (string, bool) {
	if b.Type != "tool_use" || b.Name != ToolEnterWorktree {
		return "", false
	}
	var input enterWorktreeInput
	_ = json.Unmarshal(b.Input, &input)
	return input.Path, true
}

// ExitWorktree reports whether the block is an ExitWorktree tool_use block.
// See docs/mcp/tools.md.
func (b Block) ExitWorktree() bool {
	return b.Type == "tool_use" && b.Name == ToolExitWorktree
}

// WorktreePath reads the worktree directory out of the tool_result that
// EnterWorktree returns. The text names the directory as `worktree at <path> on
// branch <branch>`. It returns an empty string when the text carries none. See
// docs/mcp/tools.md.
func WorktreePath(text string) string {
	const marker = "worktree at "
	start := strings.Index(text, marker)
	if start < 0 {
		return ""
	}
	rest := text[start+len(marker):]
	end := strings.Index(rest, " on branch ")
	if end < 0 {
		return ""
	}
	return rest[:end]
}
