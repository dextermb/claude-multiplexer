package mcp_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// recordingAPI records the last CreateInput, so a test asserts whether a create
// through the peer listener is marked hosted.
type recordingAPI struct {
	mu   sync.Mutex
	last mcp.CreateInput
}

func (r *recordingAPI) Create(in mcp.CreateInput, _ string) (string, error) {
	r.mu.Lock()
	r.last = in
	r.mu.Unlock()
	return "new", nil
}

func (r *recordingAPI) createdHosted() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last.Hosted
}

func (r *recordingAPI) createdTempDir() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last.TempDir
}

func (r *recordingAPI) List() []mcp.Session                             { return nil }
func (r *recordingAPI) Messages(string, int) ([]mcp.Message, error)     { return nil, nil }
func (r *recordingAPI) Jobs(string) ([]mcp.Job, error)                  { return nil, nil }
func (r *recordingAPI) SetTitle(string, string) error                   { return nil }
func (r *recordingAPI) SendFrom(string, string, string) (int, error)    { return 0, nil }
func (r *recordingAPI) Stop(context.Context, string, string) error      { return nil }
func (r *recordingAPI) Interrupt(context.Context, string, string) error { return nil }
func (r *recordingAPI) Archive(string, bool, string) error              { return nil }
func (r *recordingAPI) StopJob(string, string, string) (int, error)     { return 0, nil }
func (r *recordingAPI) Stream(context.Context, string) (<-chan wire.Event, error) {
	return nil, nil
}

// TestPeerListenerCreatesHosted checks that a create through the peer listener
// is hosted, and the same create through the loopback API is not.
func TestPeerListenerCreatesHosted(t *testing.T) {
	store, err := api.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAdmin(); err != nil {
		t.Fatal(err)
	}
	client, secret, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatal(err)
	}

	rec := &recordingAPI{}
	server := mcp.NewServer(newFakeSessions())
	server.EnableAPI(store, func(_, _ string) mcp.APISessions { return rec })
	if err := server.Start(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := server.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})

	// A create through the peer listener is hosted.
	token := grant(t, server.PeerBaseURL(), client.ClientID, secret)
	if code, _ := apiDo(t, http.MethodPost, server.PeerBaseURL()+"/api/sessions", token, `{"dir":"/tmp"}`); code != http.StatusOK {
		t.Fatalf("peer create status = %d", code)
	}
	if !rec.createdHosted() {
		t.Errorf("a create through the peer listener is not hosted")
	}

	// A create through the loopback API is not hosted.
	loopToken := grant(t, server.BaseURL(), client.ClientID, secret)
	if code, _ := apiDo(t, http.MethodPost, server.BaseURL()+"/api/sessions", loopToken, `{"dir":"/tmp"}`); code != http.StatusOK {
		t.Fatalf("loopback create status = %d", code)
	}
	if rec.createdHosted() {
		t.Errorf("a create through the loopback API is hosted")
	}
}

// TestPeerCreateAcceptsTempDir checks that a create with temp_dir and no dir is
// accepted and carries the flag to the host.
func TestPeerCreateAcceptsTempDir(t *testing.T) {
	store, err := api.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAdmin(); err != nil {
		t.Fatal(err)
	}
	client, secret, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatal(err)
	}

	rec := &recordingAPI{}
	server := mcp.NewServer(newFakeSessions())
	server.EnableAPI(store, func(_, _ string) mcp.APISessions { return rec })
	if err := server.Start(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := server.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})

	token := grant(t, server.PeerBaseURL(), client.ClientID, secret)
	if code, _ := apiDo(t, http.MethodPost, server.PeerBaseURL()+"/api/sessions", token, `{"temp_dir":true}`); code != http.StatusOK {
		t.Fatalf("temp_dir create with no dir = %d, want 200", code)
	}
	if !rec.createdTempDir() {
		t.Errorf("the host did not receive temp_dir")
	}
}

// TestReserveGateRefusesHostedCreate checks that a create through the peer
// listener is refused with 403 while the gate is tripped, and a create through
// the loopback API is not.
func TestReserveGateRefusesHostedCreate(t *testing.T) {
	store, err := api.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAdmin(); err != nil {
		t.Fatal(err)
	}
	client, secret, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatal(err)
	}

	sessions := newFakeSessions()
	sessions.hostingPaused = true
	server := mcp.NewServer(sessions)
	server.EnableAPI(store, func(_, _ string) mcp.APISessions { return fakeAPI{} })
	if err := server.Start(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := server.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})

	// The peer listener refuses a new session while the gate is tripped.
	token := grant(t, server.PeerBaseURL(), client.ClientID, secret)
	if code, _ := apiDo(t, http.MethodPost, server.PeerBaseURL()+"/api/sessions", token, `{"dir":"/tmp"}`); code != http.StatusForbidden {
		t.Fatalf("peer create while tripped = %d, want 403", code)
	}

	// The loopback API is not gated by the reserve.
	loopToken := grant(t, server.BaseURL(), client.ClientID, secret)
	if code, _ := apiDo(t, http.MethodPost, server.BaseURL()+"/api/sessions", loopToken, `{"dir":"/tmp"}`); code != http.StatusOK {
		t.Fatalf("loopback create while tripped = %d, want 200", code)
	}
}
