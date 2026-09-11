package manager

import (
	"context"
	"fmt"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

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
