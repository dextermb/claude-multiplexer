package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func decodeSpectateLink(t *testing.T, link string) spectatePayload {
	t.Helper()
	blob := strings.TrimPrefix(link, "cmux://spectate/")
	if blob == link {
		t.Fatalf("link has no spectate scheme: %q", link)
	}
	raw, err := base64.RawURLEncoding.DecodeString(blob)
	if err != nil {
		t.Fatalf("decode blob: %v", err)
	}
	var p spectatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return p
}

func TestShareSessionMintsALink(t *testing.T) {
	m := newTestManager(t)
	if err := m.StartMCP(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	if err := m.mcp.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatalf("start peer: %v", err)
	}
	writeStoredMeta(t, m.opts.Root, "brave-otter", "")

	created, err := m.ShareSession("brave-otter", nil)
	if err != nil {
		t.Fatalf("share session: %v", err)
	}
	if created.Session != "brave-otter" || created.ID == "" {
		t.Fatalf("share created: %+v", created)
	}
	if created.Expires.IsZero() {
		t.Fatal("the default share has no expiry")
	}

	payload := decodeSpectateLink(t, created.Link)
	if payload.U != m.PeerEndpoint().URL {
		t.Fatalf("link url = %q, want %q", payload.U, m.PeerEndpoint().URL)
	}
	id, _, ok := strings.Cut(payload.T, ".")
	if !ok || id != created.ID {
		t.Fatalf("link token %q does not carry the id %q", payload.T, created.ID)
	}

	if got := m.ListShares(); len(got) != 1 || got[0].ID != created.ID {
		t.Fatalf("list shares = %+v, want the one share", got)
	}
	had, err := m.RevokeShare(created.ID)
	if err != nil || !had {
		t.Fatalf("revoke = (%v, %v), want (true, nil)", had, err)
	}
	if got := m.ListShares(); len(got) != 0 {
		t.Fatalf("list after revoke = %+v, want empty", got)
	}
}

func TestShareSessionNoExpiry(t *testing.T) {
	m := newTestManager(t)
	if err := m.StartMCP(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	if err := m.mcp.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatalf("start peer: %v", err)
	}
	writeStoredMeta(t, m.opts.Root, "brave-otter", "")

	zero := 0.0
	created, err := m.ShareSession("brave-otter", &zero)
	if err != nil {
		t.Fatalf("share session: %v", err)
	}
	if !created.Expires.IsZero() {
		t.Fatalf("expires_hours 0 set an expiry: %v", created.Expires)
	}
}

func TestShareSessionErrors(t *testing.T) {
	// No API store yet: ErrNoAPIStore.
	m := newTestManager(t)
	if _, err := m.ShareSession("x", nil); !errors.Is(err, ErrNoAPIStore) {
		t.Fatalf("no store: want ErrNoAPIStore, got %v", err)
	}

	if err := m.StartMCP(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	// Peering off: errPeeringOff, even for a session that exists.
	writeStoredMeta(t, m.opts.Root, "brave-otter", "")
	if _, err := m.ShareSession("brave-otter", nil); !errors.Is(err, errPeeringOff) {
		t.Fatalf("peering off: want errPeeringOff, got %v", err)
	}

	// Unknown session: ErrNotFound.
	if err := m.mcp.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatalf("start peer: %v", err)
	}
	if _, err := m.ShareSession("nope", nil); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("unknown session: want ErrNotFound, got %v", err)
	}

	// Revoke of an unknown share reports not-there, no error.
	if had, err := m.RevokeShare("shr_missing"); err != nil || had {
		t.Fatalf("revoke unknown = (%v, %v), want (false, nil)", had, err)
	}
}

func TestParseSpectateLinkRoundTrip(t *testing.T) {
	link := spectateLink("http://192.168.1.20:51900", "shr_abc.secret")
	payload, id, err := parseSpectateLink(link)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if payload.U != "http://192.168.1.20:51900" || payload.T != "shr_abc.secret" || id != "shr_abc" {
		t.Fatalf("payload = %+v, id = %q", payload, id)
	}
	for _, bad := range []string{"", "http://x", "cmux://spectate/", "cmux://spectate/!!notbase64"} {
		if _, _, err := parseSpectateLink(bad); !errors.Is(err, errBadShareLink) {
			t.Errorf("parse %q: want errBadShareLink, got %v", bad, err)
		}
	}
}

// watchOwnShare mints a share on a manager for a stored session, then watches it
// through the same manager's peer listener. It returns the local name.
func watchOwnShare(t *testing.T, m *Manager, session string) (string, mcp.ShareCreated) {
	t.Helper()
	if err := m.StartMCP(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	if err := m.mcp.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatalf("start peer: %v", err)
	}
	writeStoredMeta(t, m.opts.Root, session, "")
	created, err := m.ShareSession(session, nil)
	if err != nil {
		t.Fatalf("share session: %v", err)
	}
	name, err := m.WatchShare(created.Link)
	if err != nil {
		t.Fatalf("watch share: %v", err)
	}
	return name, created
}

func TestWatchShareAttachesReadOnly(t *testing.T) {
	m := newTestManager(t)
	name, _ := watchOwnShare(t, m, "brave-otter")

	var row *mcp.Session
	for _, s := range m.List() {
		if s.Name == name {
			row = &s
		}
	}
	if row == nil || !row.ReadOnly {
		t.Fatalf("the spectator session is not read-only in the list: %+v", row)
	}

	if err := m.Send(name, "hi"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Send: want ErrReadOnly, got %v", err)
	}
	if err := m.Stop(context.Background(), name); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Stop: want ErrReadOnly, got %v", err)
	}
	if err := m.Interrupt(name, false); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Interrupt: want ErrReadOnly, got %v", err)
	}

	data, err := os.ReadFile(remotesPath(m.opts.Root))
	if err != nil {
		t.Fatalf("read remotes: %v", err)
	}
	if !strings.Contains(string(data), "share_link") || !strings.Contains(string(data), "read_only") {
		t.Fatalf("remotes.json did not record the share link: %s", data)
	}
}

func TestWatchShareDetachesOnRevokedShare(t *testing.T) {
	m := newTestManager(t)
	if err := m.StartMCP(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	if err := m.mcp.StartPeer("127.0.0.1:0"); err != nil {
		t.Fatalf("start peer: %v", err)
	}
	writeStoredMeta(t, m.opts.Root, "brave-otter", "")
	created, err := m.ShareSession("brave-otter", nil)
	if err != nil {
		t.Fatalf("share session: %v", err)
	}
	// Revoke first, so the very first stream attempt gets a terminal 401.
	if _, err := m.RevokeShare(created.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	name, err := m.WatchShare(created.Link)
	if err != nil {
		t.Fatalf("watch share: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		found := false
		for _, s := range m.List() {
			if s.Name == name {
				found = true
			}
		}
		if !found {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the spectator session did not detach after the share was revoked")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWatchedReflectsALiveSpectator(t *testing.T) {
	m := newTestManager(t)
	name, _ := watchOwnShare(t, m, "brave-otter")

	// The stream opens on its own pump, so wait for the host to see the watcher.
	waitFor(t, 3*time.Second, func() bool { return m.Watched()["brave-otter"] })

	// Stop watching, and the host no longer reports the session as watched.
	if !m.StopWatching(name) {
		t.Fatalf("StopWatching(%q) = false", name)
	}
	waitFor(t, 3*time.Second, func() bool { return !m.Watched()["brave-otter"] })
}

func TestStopWatchingDetachesLocally(t *testing.T) {
	m := newTestManager(t)
	name, _ := watchOwnShare(t, m, "brave-otter")

	if m.StopWatching("no-such-session") {
		t.Error("StopWatching reported a detach for an unknown session")
	}
	if !m.StopWatching(name) {
		t.Fatalf("StopWatching(%q) = false, want true", name)
	}
	for _, s := range m.List() {
		if s.Name == name {
			t.Fatalf("the spectator session %q is still listed after StopWatching", name)
		}
	}
}

func TestSameHost(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"192.168.1.20", "192.168.1.20", true},
		{"::1", "0:0:0:0:0:0:0:1", true},
		{"192.168.1.20", "192.168.1.21", false},
		{"MacBook.local", "macbook.local", true},
		{"host-a", "host-b", false},
		{"192.168.1.20", "", false},
		{"", "192.168.1.20", false},
	} {
		if got := sameHost(tc.a, tc.b); got != tc.want {
			t.Errorf("sameHost(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestSpectatorHostGroupsUnderConfiguredPeer(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(m.opts.Root, config.FileName)}
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "studio", URL: "http://192.168.1.20:51900"}); err != nil {
		t.Fatalf("add peer: %v", err)
	}

	// The share link reaches the same host on any port, so the spectator groups
	// under the configured peer name, not the raw address.
	if got := m.spectatorHost("http://192.168.1.20:60123"); got != "studio" {
		t.Errorf("spectatorHost matched peer = %q, want \"studio\"", got)
	}

	// A link to an unconfigured host keeps the URL host as the group.
	if got := m.spectatorHost("http://10.0.0.5:51900"); got != "10.0.0.5" {
		t.Errorf("spectatorHost no match = %q, want \"10.0.0.5\"", got)
	}
}

func TestWatchShareRejectsABadLink(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.WatchShare("not-a-link"); !errors.Is(err, errBadShareLink) {
		t.Fatalf("bad link: want errBadShareLink, got %v", err)
	}
}
