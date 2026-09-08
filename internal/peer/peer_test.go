package peer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func TestUsageRunsTheGrantThenReadsUsage(t *testing.T) {
	var grants, reads int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			atomic.AddInt32(&grants, 1)
			json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
		case "/api/usage":
			atomic.AddInt32(&reads, 1)
			if r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"five_hour": map[string]any{"remaining": 42},
				"ok":        true,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New(config.PeerHost{Name: "b", URL: srv.URL, ClientID: "id", ClientSecret: "sec"})
	got, err := c.Usage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 42 {
		t.Errorf("remaining = %v, want 42", got.FiveHour.Remaining)
	}

	if _, err := c.Usage(context.Background()); err != nil {
		t.Fatal(err)
	}
	if grants != 1 {
		t.Errorf("grants = %d, want 1 (the token is cached)", grants)
	}
	if reads != 2 {
		t.Errorf("reads = %d, want 2", reads)
	}
}

func TestUsageRefreshesTheTokenOnA401(t *testing.T) {
	var grants int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			n := atomic.AddInt32(&grants, 1)
			json.NewEncoder(w).Encode(map[string]any{"access_token": tokenFor(n), "expires_in": 3600})
		case "/api/usage":
			if r.Header.Get("Authorization") != "Bearer tok-2" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	c := New(config.PeerHost{Name: "b", URL: srv.URL, ClientID: "id", ClientSecret: "sec"})
	if _, err := c.Usage(context.Background()); err != nil {
		t.Fatal(err)
	}
	if grants != 2 {
		t.Errorf("grants = %d, want 2 (the first token was stale)", grants)
	}
}

func tokenFor(n int32) string {
	if n == 1 {
		return "tok-1"
	}
	return "tok-2"
}

func TestUsageFailsWhenTheGrantIsRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := New(config.PeerHost{Name: "b", URL: srv.URL, ClientID: "id", ClientSecret: "bad"})
	if _, err := c.Usage(context.Background()); err == nil {
		t.Fatal("Usage did not fail on a rejected grant")
	}
}
