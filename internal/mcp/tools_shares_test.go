package mcp_test

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func TestShareToolsViaControlSession(t *testing.T) {
	server := startServer(t, newFakeSessions())
	token, err := server.Register("boss", true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	session := connect(t, server, token)

	if out := resultText(call(t, session, mcp.ToolShareSession, map[string]any{"session": "mine"})); !strings.Contains(out, "cmux://spectate/") {
		t.Fatalf("share_session returned no link: %s", out)
	}
	if out := resultText(call(t, session, mcp.ToolListShares, map[string]any{})); !strings.Contains(out, "shr_test") {
		t.Fatalf("list_shares did not list the share: %s", out)
	}
	if call(t, session, mcp.ToolRevokeShare, map[string]any{"id": "shr_test"}).IsError {
		t.Error("revoke_share returned an error result")
	}
	if !call(t, session, mcp.ToolShareSession, map[string]any{}).IsError {
		t.Error("share_session with no session must return an error result")
	}
	if !call(t, session, mcp.ToolRevokeShare, map[string]any{}).IsError {
		t.Error("revoke_share with no id must return an error result")
	}
}
