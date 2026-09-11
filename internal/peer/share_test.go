package peer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/wire"
)

func TestShareClientStreamsThroughTheSharePath(t *testing.T) {
	var streamPath string
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamPath = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		ev := wire.Event{Session: "shr_abc", Snapshot: wire.Snapshot{Name: "shr_abc"}}
		data, _ := json.Marshal(ev)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
	defer srv.Close()

	c := NewShare(srv.URL, "shr_abc.secret")
	ch, err := c.Stream(context.Background(), "shr_abc")
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for range ch {
		got++
	}
	if got != 1 {
		t.Fatalf("events = %d, want 1", got)
	}
	if streamPath != "/api/shares/shr_abc/stream" {
		t.Errorf("share stream path = %q", streamPath)
	}
	if authHeader != "Bearer shr_abc.secret" {
		t.Errorf("auth header = %q, want the share token as a bearer", authHeader)
	}
}

func TestShareClientStreamReturnsUnauthorizedOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewShare(srv.URL, "shr_abc.secret")
	if _, err := c.Stream(context.Background(), "shr_abc"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("stream on 401: want ErrUnauthorized, got %v", err)
	}
}

func TestShareClientRefusesWrites(t *testing.T) {
	c := NewShare("http://host:51900", "shr_abc.secret")
	if err := c.Send(context.Background(), "shr_abc", "hi"); !errors.Is(err, errReadOnlyShare) {
		t.Errorf("Send on a share: want errReadOnlyShare, got %v", err)
	}
	if err := c.Stop(context.Background(), "shr_abc"); !errors.Is(err, errReadOnlyShare) {
		t.Errorf("Stop on a share: want errReadOnlyShare, got %v", err)
	}
	if err := c.Interrupt(context.Background(), "shr_abc"); !errors.Is(err, errReadOnlyShare) {
		t.Errorf("Interrupt on a share: want errReadOnlyShare, got %v", err)
	}
}

func TestShareClientReadsMessagesThroughTheSharePath(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"messages": []map[string]string{{"role": "user", "text": "hi"}}})
	}))
	defer srv.Close()

	c := NewShare(srv.URL, "shr_abc.secret")
	msgs, err := c.Messages(context.Background(), "shr_abc", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || path != "/api/shares/shr_abc/messages" {
		t.Errorf("messages = %+v, path = %q", msgs, path)
	}
}
