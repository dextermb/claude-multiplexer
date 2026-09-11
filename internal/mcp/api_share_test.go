package mcp_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

// startShareServer starts a server with the API and the share surface, on the
// peer listener, and returns the server and the store.
func startShareServer(t *testing.T) (*mcp.Server, *api.Store) {
	t.Helper()
	store, err := api.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	server := mcp.NewServer(newFakeSessions())
	server.EnableAPI(store, func(_, _ string) mcp.APISessions { return fakeAPI{} })
	server.EnableShares(func(string) mcp.APISessions { return fakeAPI{} })
	if err := server.Start(0, 0); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := server.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatalf("start peer: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server, store
}

func TestShareSurfaceStreamsReadOnly(t *testing.T) {
	server, store := startShareServer(t)
	base := server.PeerBaseURL()

	share, token, err := store.CreateShare("mine", time.Hour)
	if err != nil {
		t.Fatalf("create share: %v", err)
	}

	// No token, and a bad token, both get 401.
	if code, _ := apiGet(t, base, "/api/shares/"+share.ID+"/stream", ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", code)
	}
	if code, _ := apiGet(t, base, "/api/shares/"+share.ID+"/stream", share.ID+".wrong"); code != http.StatusUnauthorized {
		t.Fatalf("bad token: want 401, got %d", code)
	}

	// The stream carries SSE events under the share id, never the host-local name.
	code, body := apiGet(t, base, "/api/shares/"+share.ID+"/stream", token)
	if code != http.StatusOK {
		t.Fatalf("stream: want 200, got %d", code)
	}
	if !strings.Contains(body, "data: ") || !strings.Contains(body, `"session":"`+share.ID+`"`) {
		t.Fatalf("stream body did not carry the share id: %q", body)
	}
	if strings.Contains(body, "mine") {
		t.Fatalf("the stream leaked the host-local session name: %q", body)
	}

	// The info route reports the label, not the session name.
	code, body = apiGet(t, base, "/api/shares/"+share.ID, token)
	if code != http.StatusOK || !strings.Contains(body, share.ID) {
		t.Fatalf("info: want 200 with the share id, got %d (%s)", code, body)
	}
	if strings.Contains(body, "mine") {
		t.Fatalf("the info route leaked the host-local session name: %q", body)
	}

	// Messages read-only, under the share id.
	if code, body := apiGet(t, base, "/api/shares/"+share.ID+"/messages", token); code != http.StatusOK || !strings.Contains(body, share.ID) {
		t.Fatalf("messages: want 200 with the share id, got %d (%s)", code, body)
	}
}

func TestShareSurfaceHasNoWriteRoute(t *testing.T) {
	server, store := startShareServer(t)
	base := server.PeerBaseURL()
	share, token, _ := store.CreateShare("mine", time.Hour)

	for _, path := range []string{"/message", "/stop", "/interrupt", "/archive"} {
		if code, _ := apiDo(t, http.MethodPost, base+"/api/shares/"+share.ID+path, token, `{"text":"hi"}`); code != http.StatusNotFound {
			t.Errorf("POST %s: want 404 (no write route), got %d", path, code)
		}
	}
}

func TestShareSurfaceRejectsRevoked(t *testing.T) {
	server, store := startShareServer(t)
	base := server.PeerBaseURL()
	share, token, _ := store.CreateShare("mine", time.Hour)

	if code, _ := apiGet(t, base, "/api/shares/"+share.ID, token); code != http.StatusOK {
		t.Fatalf("before revoke: want 200, got %d", code)
	}
	if err := store.RevokeShare(share.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if code, _ := apiGet(t, base, "/api/shares/"+share.ID, token); code != http.StatusUnauthorized {
		t.Fatalf("after revoke: want 401, got %d", code)
	}
}
