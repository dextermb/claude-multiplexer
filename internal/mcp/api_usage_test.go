package mcp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/usage"
)

func TestUsageRouteNeedsAToken(t *testing.T) {
	server, store := startAPIServer(t)
	_, _ = store.CreateAdmin()
	client, secret, _ := store.CreateClient("bruno")
	base := server.BaseURL()

	if code, _ := apiGet(t, base, "/api/usage", ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", code)
	}

	token := grant(t, base, client.ClientID, secret)
	code, body := apiGet(t, base, "/api/usage", token)
	if code != http.StatusOK {
		t.Fatalf("usage: want 200, got %d", code)
	}
	var got usage.Usage
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("usage did not decode: %v (%s)", err, body)
	}
}
