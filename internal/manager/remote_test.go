package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/peer"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// fakePeer is a peer host the manager reaches: it grants a token, creates a
// session, and streams a scripted set of events. It records the input routes it
// receives, so a test can assert Send/Stop/Interrupt reached it.
type fakePeer struct {
	events []wire.Event
	done   chan struct{}

	mu    sync.Mutex
	paths []string
}

func (f *fakePeer) start(t *testing.T) config.PeerHost {
	t.Helper()
	f.done = make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
		case r.URL.Path == "/api/sessions" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]string{"name": "remote-1"})
		case r.URL.Path == "/api/sessions/remote-1/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)
			for _, ev := range f.events {
				data, _ := json.Marshal(ev)
				fmt.Fprintf(w, "data: %s\n\n", data)
				if flusher != nil {
					flusher.Flush()
				}
			}
			select {
			case <-r.Context().Done():
			case <-f.done:
			}
		default:
			f.mu.Lock()
			f.paths = append(f.paths, r.URL.Path)
			f.mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}
	}))
	// This cleanup runs before the manager shutdown (registered earlier, so it
	// runs later), so the stream handler returns before the server closes.
	t.Cleanup(func() {
		close(f.done)
		srv.Close()
	})
	return config.PeerHost{Name: "workstation", URL: srv.URL, ClientID: "id", ClientSecret: "sec"}
}

func (f *fakePeer) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

func TestAttachRemoteStreamsIntoTheBus(t *testing.T) {
	m := newTestManager(t)
	fp := &fakePeer{events: []wire.Event{
		{Session: "remote-1", Lines: []render.Line{{Text: "hello from the peer"}}, Snapshot: wire.Snapshot{Name: "remote-1", State: "idle", Turns: 1}},
		{Session: "remote-1", Lines: []render.Line{{Text: "second line"}}, Snapshot: wire.Snapshot{Name: "remote-1", State: "busy", Turns: 2}},
	}}
	host := fp.start(t)

	sub := m.Subscribe(64)
	defer sub.Close()

	local, err := m.AttachRemote(host, peer.CreateSpec{Dir: "/work"})
	if err != nil {
		t.Fatal(err)
	}

	var got []Event
	waitFor(t, 3*time.Second, func() bool {
		for {
			select {
			case ev := <-sub.C:
				if ev.Session == local {
					got = append(got, ev)
				}
				continue
			default:
			}
			break
		}
		return len(got) >= 2
	})

	// The pump republishes each event under the local name, so the pane treats it
	// as local.
	if got[0].Session != local {
		t.Errorf("event session = %q, want %q", got[0].Session, local)
	}

	// Lines and Snapshot route to the remote, and the first event replaces the
	// buffer while the second appends, so the buffer holds both lines once.
	waitFor(t, 3*time.Second, func() bool {
		return len(m.Lines(local)) == 2
	})
	snap, err := m.Snapshot(local)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Turns != 2 || snap.State.String() != "busy" {
		t.Errorf("snapshot = turns %d state %s, want 2 busy", snap.Turns, snap.State.String())
	}

	// List carries the streamed session with its peer as Host.
	found := false
	for _, s := range m.List() {
		if s.Name == local {
			found = true
			if s.Host != "workstation" {
				t.Errorf("Host = %q, want workstation", s.Host)
			}
			if !s.Live {
				t.Errorf("a streamed session reads as not live")
			}
		}
	}
	if !found {
		t.Fatalf("the streamed session %q is not in the list", local)
	}
}

func TestRemoteInputRoutesToThePeer(t *testing.T) {
	m := newTestManager(t)
	fp := &fakePeer{}
	host := fp.start(t)

	local, err := m.AttachRemote(host, peer.CreateSpec{Dir: "/work"})
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Send(local, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := m.Interrupt(local, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop(context.Background(), local); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 3*time.Second, func() bool {
		return len(fp.seen()) >= 3
	})
	want := map[string]bool{
		"/api/sessions/remote-1/message":   false,
		"/api/sessions/remote-1/interrupt": false,
		"/api/sessions/remote-1/stop":      false,
	}
	for _, p := range fp.seen() {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for path, hit := range want {
		if !hit {
			t.Errorf("the peer never saw %s", path)
		}
	}
}

func TestClosedEventDetachesTheRemote(t *testing.T) {
	m := newTestManager(t)
	fp := &fakePeer{events: []wire.Event{
		{Session: "remote-1", Snapshot: wire.Snapshot{Name: "remote-1", State: "exited"}, Closed: true},
	}}
	host := fp.start(t)

	local, err := m.AttachRemote(host, peer.CreateSpec{Dir: "/work"})
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, 3*time.Second, func() bool {
		return m.remote(local) == nil
	})
	for _, s := range m.List() {
		if s.Name == local {
			t.Errorf("the closed streamed session is still in the list")
		}
	}
}

func TestReattachRemotesRestoresAStreamedSession(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(m.opts.Root, config.FileName)}
	fp := &fakePeer{}
	host := fp.start(t)
	// The config names the peer under the same name the link records, so the
	// re-attach finds the credentials to reach it.
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: host.Name, URL: host.URL, ClientID: host.ClientID, ClientSecret: host.ClientSecret}); err != nil {
		t.Fatal(err)
	}

	local, err := m.AttachRemote(host, peer.CreateSpec{Dir: "/work"})
	if err != nil {
		t.Fatal(err)
	}
	m.detachPumpsOnly()

	// A fresh manager on the same root re-attaches the streamed session from the
	// saved link, under the same local name, without creating a new session.
	m2, err := New(Options{Root: m.opts.Root, ClaudePath: fakeClaude, Renderer: render.Renderer{}, ConfigPaths: m.opts.ConfigPaths})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m2.Shutdown(ctx)
	})
	m2.reattachRemotes()

	if re := m2.remote(local); re == nil || re.remoteName != "remote-1" || re.peer != "workstation" {
		t.Fatalf("the streamed session was not re-attached: %+v", re)
	}
}

// detachPumpsOnly cancels the remote pumps without clearing the saved links, so
// a test can simulate a restart.
func (m *Manager) detachPumpsOnly() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, re := range m.remotes {
		re.cancel()
	}
}
