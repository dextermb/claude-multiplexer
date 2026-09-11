package manager

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
