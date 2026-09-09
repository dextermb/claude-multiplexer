package mcp_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestStreamRouteEmitsSSE(t *testing.T) {
	server, store := startAPIServer(t)
	_, _ = store.CreateAdmin()
	client, secret, _ := store.CreateClient("bruno")
	token := grant(t, server.BaseURL(), client.ClientID, secret)
	base := server.BaseURL()

	if code, _ := apiGet(t, base, "/api/sessions/mine/stream", ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", code)
	}
	code, body := apiGet(t, base, "/api/sessions/mine/stream", token)
	if code != http.StatusOK {
		t.Fatalf("stream: want 200, got %d", code)
	}
	if !strings.Contains(body, "data: ") || !strings.Contains(body, `"session":"mine"`) {
		t.Fatalf("stream body did not carry an SSE event: %q", body)
	}

	if code, _ := apiGet(t, base, "/api/sessions/other/stream", token); code != http.StatusNotFound {
		t.Fatalf("unowned stream: want 404, got %d", code)
	}
}
