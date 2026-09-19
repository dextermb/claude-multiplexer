package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func TestGetPRReturnsAPullRequestPerCodeBase(t *testing.T) {
	sessions := newFakeSessions()
	sessions.pr.enabled = true
	sessions.pr.items = []mcp.PullRequest{
		{Provider: "github", Number: 1045, State: "open", Unresolved: 3, Dir: "/a", Found: true},
		{Provider: "gitlab", Number: 88, State: "open", Dir: "/b", Found: true},
	}
	client := workingDirClient(t, sessions)

	result := call(t, client, mcp.ToolGetPR, nil)
	if result.IsError {
		t.Fatalf("get_pr failed: %s", resultText(result))
	}
	var out struct {
		OK    bool `json:"ok"`
		Count int  `json:"count"`
		PRs   []struct {
			Number int    `json:"number"`
			Dir    string `json:"dir"`
		} `json:"prs"`
	}
	if err := json.Unmarshal([]byte(resultText(result)), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Count != 2 || len(out.PRs) != 2 {
		t.Fatalf("out = %+v, want 2 PRs", out)
	}
	if out.PRs[0].Number != 1045 || out.PRs[1].Number != 88 {
		t.Fatalf("PRs = %+v", out.PRs)
	}
}

func TestGetPRAbsentWhenDisabled(t *testing.T) {
	sessions := newFakeSessions() // pr.enabled is false
	client := workingDirClient(t, sessions)

	if toolNames(t, client)[mcp.ToolGetPR] {
		t.Fatal("get_pr must be absent when no provider is configured")
	}
}
