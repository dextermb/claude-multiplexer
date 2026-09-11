package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/peer"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// defaultShareTTL is how long a share lives when the caller names no expiry, so
// a forgotten link does not stay open. See docs/peers.md.
const defaultShareTTL = 24 * time.Hour

// spectatorBase is the base name of a spectator session, because the viewer
// hides the host-local name. The row shows the shared title from the stream.
const spectatorBase = "shared"

// errPeeringOff is the failure when a session shares a session but the peer
// listener is off, because the link would point at nothing.
var errPeeringOff = errors.New("manager: peering is off; run enable_peering and restart before you share a session")

// errBadShareLink is the failure when a watch link is not a spectate link this
// host understands.
var errBadShareLink = errors.New("manager: not a cmux spectate link")

// shareSessions returns the read-only view a spectator reaches through a share.
// A share opens one session, so the view pins that name and serves only its read
// routes. See docs/peers.md.
func (m *Manager) shareSessions(session string) mcp.APISessions {
	return &sharedSession{m: m, session: session}
}

// sharedSession is the read-only view of the one session a share opens. It
// serves List, Messages, and Stream for that session only, and it refuses every
// write. See docs/peers.md.
type sharedSession struct {
	m       *Manager
	session string
}

func (v *sharedSession) guard(name string) error {
	if name != v.session {
		return fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	if _, ok := v.m.ownerOf(name); !ok {
		return fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return nil
}

func (v *sharedSession) List() []mcp.Session {
	for _, s := range v.m.List() {
		if s.Name == v.session {
			return []mcp.Session{s}
		}
	}
	return nil
}

func (v *sharedSession) Messages(name string, limit int) ([]mcp.Message, error) {
	if err := v.guard(name); err != nil {
		return nil, err
	}
	return v.m.Messages(name, limit)
}

func (v *sharedSession) Stream(ctx context.Context, name string) (<-chan wire.Event, error) {
	if err := v.guard(name); err != nil {
		return nil, err
	}
	return v.m.streamSession(ctx, name), nil
}

func (v *sharedSession) Jobs(string) ([]mcp.Job, error)               { return nil, mcp.ErrReadOnly }
func (v *sharedSession) SetTitle(string, string) error                { return mcp.ErrReadOnly }
func (v *sharedSession) SendFrom(string, string, string) (int, error) { return 0, mcp.ErrReadOnly }
func (v *sharedSession) Stop(context.Context, string, string) error   { return mcp.ErrReadOnly }
func (v *sharedSession) Interrupt(context.Context, string, string) error {
	return mcp.ErrReadOnly
}
func (v *sharedSession) Archive(string, bool, string) error          { return mcp.ErrReadOnly }
func (v *sharedSession) StopJob(string, string, string) (int, error) { return 0, mcp.ErrReadOnly }
func (v *sharedSession) Create(mcp.CreateInput, string) (string, error) {
	return "", mcp.ErrReadOnly
}

// spectatePayload is the JSON a spectate link carries: the peer-listener URL and
// the share token. It is base64url-encoded into the link. See docs/peers.md.
type spectatePayload struct {
	U string `json:"u"`
	T string `json:"t"`
}

// spectateLink bundles the peer URL and the token into one pasteable link.
func spectateLink(peerURL, token string) string {
	payload, _ := json.Marshal(spectatePayload{U: peerURL, T: token})
	return "cmux://spectate/" + base64.RawURLEncoding.EncodeToString(payload)
}

// ShareSession mints a read-only share for a session, and returns the share with
// its spectate link. It fails when the session is not there, or when peering is
// off. See docs/peers.md.
func (m *Manager) ShareSession(session string, expiresHours *float64) (mcp.ShareCreated, error) {
	if m.apiStore == nil {
		return mcp.ShareCreated{}, ErrNoAPIStore
	}
	session = strings.TrimSpace(session)
	if session == "" {
		return mcp.ShareCreated{}, mcp.ErrNoTarget
	}
	if _, ok := m.ownerOf(session); !ok {
		return mcp.ShareCreated{}, fmt.Errorf("%w: %s", mcp.ErrNotFound, session)
	}
	end := m.PeerEndpoint()
	if !end.Enabled || end.URL == "" {
		return mcp.ShareCreated{}, errPeeringOff
	}
	ttl := defaultShareTTL
	if expiresHours != nil {
		if *expiresHours <= 0 {
			ttl = 0
		} else {
			ttl = time.Duration(*expiresHours * float64(time.Hour))
		}
	}
	share, token, err := m.apiStore.CreateShare(session, ttl)
	if err != nil {
		return mcp.ShareCreated{}, err
	}
	return mcp.ShareCreated{
		ID:      share.ID,
		Session: share.Session,
		Expires: share.ExpiresAt,
		Link:    spectateLink(end.URL, token),
	}, nil
}

// ListShares lists the active shares, without a secret. See docs/peers.md.
func (m *Manager) ListShares() []mcp.ShareView {
	if m.apiStore == nil {
		return nil
	}
	shares := m.apiStore.ListShares()
	out := make([]mcp.ShareView, 0, len(shares))
	for _, s := range shares {
		out = append(out, mcp.ShareView{
			ID:      s.ID,
			Session: s.Session,
			Scope:   s.Scope,
			Created: s.CreatedAt,
			Expires: s.ExpiresAt,
		})
	}
	return out
}

// RevokeShare ends a share by id, and reports whether the share was there. The
// live stream ends on its next re-check. See docs/peers.md.
func (m *Manager) RevokeShare(id string) (bool, error) {
	if m.apiStore == nil {
		return false, ErrNoAPIStore
	}
	if err := m.apiStore.RevokeShare(strings.TrimSpace(id)); err != nil {
		if errors.Is(err, api.ErrUnknownShare) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// parseSpectateLink decodes a cmux://spectate/<blob> link into its payload and
// the share id inside the token. It fails when the link is not a spectate link.
func parseSpectateLink(link string) (spectatePayload, string, error) {
	blob, ok := strings.CutPrefix(strings.TrimSpace(link), "cmux://spectate/")
	if !ok || blob == "" {
		return spectatePayload{}, "", errBadShareLink
	}
	raw, err := base64.RawURLEncoding.DecodeString(blob)
	if err != nil {
		return spectatePayload{}, "", errBadShareLink
	}
	var payload spectatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return spectatePayload{}, "", errBadShareLink
	}
	id, _, ok := strings.Cut(payload.T, ".")
	if payload.U == "" || payload.T == "" || !ok || id == "" {
		return spectatePayload{}, "", errBadShareLink
	}
	return payload, id, nil
}

// shareHost is the host of a peer URL, shown as the Host of a spectator session.
func shareHost(peerURL string) string {
	if u, err := url.Parse(peerURL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "shared"
}

// WatchShare attaches a read-only spectator session from a spectate link. It
// returns the local name. The viewer needs no peer listener and no client. See
// docs/peers.md.
func (m *Manager) WatchShare(link string) (string, error) {
	payload, id, err := parseSpectateLink(link)
	if err != nil {
		return "", err
	}
	client := peer.NewShare(payload.U, payload.T)
	local := m.attach(attachSpec{
		peer:       shareHost(payload.U),
		client:     client,
		remoteName: id,
		base:       spectatorBase,
		readOnly:   true,
		shareLink:  strings.TrimSpace(link),
	})
	m.saveRemotes()
	return local, nil
}
