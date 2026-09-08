package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// ownerOf reports the owner of a session, and whether the session is there. A
// live session reads its owner from memory; a stored session reads it from disk.
func (m *Manager) ownerOf(name string) (string, bool) {
	m.mu.Lock()
	item, live := m.entries[name]
	m.mu.Unlock()
	if live {
		return item.metaCopy().Owner, true
	}
	meta, err := m.Meta(name)
	if err != nil {
		return "", false
	}
	return meta.Owner, true
}

// apiSessions gives the owner-scoped view one API client reaches. The client id
// is the owner, and the client name marks the notice the interface shows.
func (m *Manager) apiSessions(clientID, clientName string) mcp.APISessions {
	return &ownedSessions{m: m, owner: clientID, name: clientName}
}

// ownedSessions is the owner-scoped view of the sessions one API client reaches.
// It hides every session the client does not own, so the client sees and touches
// only its own sessions. See docs/mcp/api.md.
type ownedSessions struct {
	m     *Manager
	owner string
	name  string
}

func (o *ownedSessions) guard(name string) error {
	owner, ok := o.m.ownerOf(name)
	if !ok || owner != o.owner {
		return fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return nil
}

func (o *ownedSessions) List() []mcp.Session {
	all := o.m.List()
	out := make([]mcp.Session, 0, len(all))
	for _, s := range all {
		if s.Owner == o.owner {
			out = append(out, s)
		}
	}
	return out
}

func (o *ownedSessions) Messages(name string, limit int) ([]mcp.Message, error) {
	if err := o.guard(name); err != nil {
		return nil, err
	}
	return o.m.Messages(name, limit)
}

func (o *ownedSessions) Jobs(name string) ([]mcp.Job, error) {
	if err := o.guard(name); err != nil {
		return nil, err
	}
	return o.m.Jobs(name)
}

func (o *ownedSessions) SetTitle(name, title string) error {
	if err := o.guard(name); err != nil {
		return err
	}
	if err := o.m.SetTitle(name, title); err != nil {
		return err
	}
	o.m.notify(name, o.name+" renamed "+name, true)
	return nil
}

func (o *ownedSessions) SendFrom(target, from, text string) (int, error) {
	if err := o.guard(target); err != nil {
		return 0, err
	}
	return o.m.SendFrom(target, from, text)
}

func (o *ownedSessions) Stop(ctx context.Context, name, by string) error {
	if err := o.guard(name); err != nil {
		return err
	}
	if err := o.m.Stop(ctx, name); err != nil {
		return err
	}
	o.m.notify(name, by+" stopped "+name, false)
	return nil
}

func (o *ownedSessions) Archive(name string, archived bool, by string) error {
	if err := o.guard(name); err != nil {
		return err
	}
	if err := o.m.Archive(name, archived); err != nil {
		return err
	}
	verb := " archived "
	if !archived {
		verb = " restored "
	}
	o.m.notify(name, by+verb+name, true)
	return nil
}

func (o *ownedSessions) StopJob(target, jobID, by string) (int, error) {
	if err := o.guard(target); err != nil {
		return 0, err
	}
	return o.m.StopJobFrom(target, by, jobID)
}

func (o *ownedSessions) Create(in mcp.CreateInput, by string) (string, error) {
	abs, err := filepath.Abs(in.Dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, in.Dir)
	}
	created, err := o.m.Spawn(context.Background(), Spec{
		Dir:            abs,
		Name:           in.Name,
		Model:          in.Model,
		PermissionMode: in.PermissionMode,
		Effort:         in.Effort,
		Owner:          o.owner,
	})
	if err != nil {
		return "", err
	}
	o.m.notify(created, by+" created "+created, true)
	return created, nil
}

// Stream serves the session's events to a peer, once the owner check passes. See
// docs/peers.md.
func (o *ownedSessions) Stream(ctx context.Context, name string) (<-chan wire.Event, error) {
	if err := o.guard(name); err != nil {
		return nil, err
	}
	return o.m.streamSession(ctx, name), nil
}
