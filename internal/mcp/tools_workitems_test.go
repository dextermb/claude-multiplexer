package mcp_test

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func TestWorkItemToolsSetStatusAndList(t *testing.T) {
	sessions := newFakeSessions()
	sessions.work = workItemState{
		enabled:   true,
		providers: []string{"linear"},
		item:      mcp.WorkItem{Provider: "linear", Status: "Backlog"},
		statuses:  []mcp.WorkItemStatus{{Name: "In Review"}, {Name: "Done"}},
	}
	client := workingDirClient(t, sessions)

	set := call(t, client, mcp.ToolSetWorkItem, map[string]any{"key": "LIN-1"})
	if set.IsError {
		t.Fatalf("set_workitem failed: %s", resultText(set))
	}
	if sessions.work.setKey != "LIN-1" {
		t.Fatalf("set key = %q, want LIN-1", sessions.work.setKey)
	}

	list := call(t, client, mcp.ToolWorkItemStatuses, map[string]any{})
	if !strings.Contains(resultText(list), "In Review") {
		t.Fatalf("list_workitem_statuses does not name the status:\n%s", resultText(list))
	}

	status := call(t, client, mcp.ToolSetWorkItemStatus, map[string]any{"status": "In Review"})
	if status.IsError {
		t.Fatalf("set_workitem_status failed: %s", resultText(status))
	}
	if sessions.work.setStatus != "In Review" {
		t.Fatalf("set status = %q, want In Review", sessions.work.setStatus)
	}
}

func TestSetWorkItemToolNeedsAKey(t *testing.T) {
	sessions := newFakeSessions()
	sessions.work = workItemState{enabled: true, providers: []string{"linear"}}
	client := workingDirClient(t, sessions)

	if result := call(t, client, mcp.ToolSetWorkItem, map[string]any{"key": "  "}); !result.IsError {
		t.Fatal("an empty key must be an error")
	}
	if sessions.work.setKey != "" {
		t.Fatalf("a bad key still set %q", sessions.work.setKey)
	}
}

func TestWorkItemToolsAbsentWhenDisabled(t *testing.T) {
	sessions := newFakeSessions() // work.enabled is false
	client := workingDirClient(t, sessions)

	names := toolNames(t, client)
	if names[mcp.ToolSetWorkItem] || names[mcp.ToolSetWorkItemStatus] {
		t.Fatal("the action tools must be absent when no provider is configured")
	}
	// The configure tools bootstrap the feature, so they are always present.
	if !names[mcp.ToolConfigureJira] || !names[mcp.ToolConfigureLinear] {
		t.Fatal("the configure tools must be present even when disabled")
	}
}

func TestConfigureToolsWriteTheProvider(t *testing.T) {
	sessions := newFakeSessions() // disabled, but configure is always available
	client := workingDirClient(t, sessions)

	ok := call(t, client, mcp.ToolConfigureLinear, map[string]any{"token": "lin_key"})
	if ok.IsError {
		t.Fatalf("configure_linear failed: %s", resultText(ok))
	}
	if sessions.work.configured != "linear" || sessions.work.configToken != "lin_key" {
		t.Fatalf("configure did not write: %+v", sessions.work)
	}

	bad := call(t, client, mcp.ToolConfigureJira, map[string]any{"token": "  "})
	if !bad.IsError {
		t.Fatal("an empty token must be an error")
	}
}

func toolNames(t *testing.T, client *sdk.ClientSession) map[string]bool {
	t.Helper()
	tools, err := client.ListTools(context.Background(), &sdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	return names
}
