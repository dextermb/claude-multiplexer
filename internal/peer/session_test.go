package peer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

func peerServer(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
			return
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return New(config.PeerHost{Name: "b", URL: srv.URL, ClientID: "id", ClientSecret: "sec"})
}

func TestCreateSessionReturnsTheName(t *testing.T) {
	var gotBody map[string]string
	c := peerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]string{"name": "remote-1"})
	})
	name, err := c.CreateSession(context.Background(), CreateSpec{Dir: "/tmp", Model: "opus"})
	if err != nil {
		t.Fatal(err)
	}
	if name != "remote-1" {
		t.Errorf("name = %q, want remote-1", name)
	}
	if gotBody["dir"] != "/tmp" || gotBody["model"] != "opus" {
		t.Errorf("body = %+v, want dir /tmp and model opus", gotBody)
	}
}

func TestSendAndStopPostToThePeer(t *testing.T) {
	var paths []string
	c := peerServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	if err := c.Send(context.Background(), "remote-1", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(context.Background(), "remote-1"); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/api/sessions/remote-1/message" || paths[1] != "/api/sessions/remote-1/stop" {
		t.Errorf("paths = %v", paths)
	}
}

func TestStreamDecodesSSEEvents(t *testing.T) {
	c := peerServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 3; i++ {
			ev := wire.Event{Session: "remote-1", Snapshot: wire.Snapshot{Name: "remote-1", Turns: i}}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	})
	ch, err := c.Stream(context.Background(), "remote-1")
	if err != nil {
		t.Fatal(err)
	}
	var got int
	for ev := range ch {
		if ev.Session != "remote-1" {
			t.Errorf("session = %q", ev.Session)
		}
		got++
	}
	if got != 3 {
		t.Errorf("events = %d, want 3", got)
	}
}
