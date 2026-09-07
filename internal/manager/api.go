package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
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

func (o *ownedSessions) Create(dir, name, by string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, dir)
	}
	created, err := o.m.Spawn(context.Background(), Spec{Dir: abs, Name: name, Owner: o.owner})
	if err != nil {
		return "", err
	}
	o.m.notify(created, by+" created "+created, true)
	return created, nil
}

// The credential-management methods delegate to the api store, and drop the
// access tokens of a client the moment the client changes. See docs/mcp/api.md.

func (m *Manager) CreateAPIAdmin() (string, error) {
	if m.apiStore == nil {
		return "", ErrNoAPIStore
	}
	return m.apiStore.CreateAdmin()
}

func (m *Manager) RotateAPIAdmin() (string, error) {
	if m.apiStore == nil {
		return "", ErrNoAPIStore
	}
	return m.apiStore.RotateAdmin()
}

func (m *Manager) RevokeAPIAdmin() error {
	if m.apiStore == nil {
		return ErrNoAPIStore
	}
	return m.apiStore.RevokeAdmin()
}

func (m *Manager) CreateAPIClient(name string) (mcp.APIClient, string, error) {
	if m.apiStore == nil {
		return mcp.APIClient{}, "", ErrNoAPIStore
	}
	client, secret, err := m.apiStore.CreateClient(name)
	if err != nil {
		return mcp.APIClient{}, "", err
	}
	return apiClientView(client), secret, nil
}

func (m *Manager) UpdateAPIClient(id string, name *string, disabled *bool) (mcp.APIClient, error) {
	if m.apiStore == nil {
		return mcp.APIClient{}, ErrNoAPIStore
	}
	client, err := m.apiStore.UpdateClient(id, name, disabled)
	if err != nil {
		return mcp.APIClient{}, err
	}
	if disabled != nil && *disabled {
		m.revokeClientTokens(id)
	}
	return apiClientView(client), nil
}

func (m *Manager) RotateAPIClient(id string) (string, error) {
	if m.apiStore == nil {
		return "", ErrNoAPIStore
	}
	secret, err := m.apiStore.RotateClient(id)
	if err != nil {
		return "", err
	}
	m.revokeClientTokens(id)
	return secret, nil
}

func (m *Manager) RevokeAPIClient(id string) error {
	if m.apiStore == nil {
		return ErrNoAPIStore
	}
	if err := m.apiStore.RevokeClient(id); err != nil {
		return err
	}
	m.revokeClientTokens(id)
	return nil
}

func (m *Manager) ListAPIClients() []mcp.APIClient {
	if m.apiStore == nil {
		return nil
	}
	clients := m.apiStore.ListClients()
	out := make([]mcp.APIClient, 0, len(clients))
	for _, client := range clients {
		out = append(out, apiClientView(client))
	}
	return out
}

func (m *Manager) APIEndpoint() mcp.APIEndpoint {
	start, end := m.apiPortRange()
	url := ""
	if m.mcp != nil {
		url = m.mcp.BaseURL()
	}
	return mcp.APIEndpoint{URL: url, PortStart: start, PortEnd: end}
}

func (m *Manager) revokeClientTokens(id string) {
	if m.mcp != nil {
		m.mcp.RevokeClientTokens(id)
	}
}

func apiClientView(client api.Client) mcp.APIClient {
	view := mcp.APIClient{
		ClientID: client.ClientID,
		Name:     client.Name,
		Disabled: client.Disabled,
	}
	if !client.CreatedAt.IsZero() {
		view.CreatedAt = client.CreatedAt.Format(time.RFC3339)
	}
	if !client.RotatedAt.IsZero() {
		view.RotatedAt = client.RotatedAt.Format(time.RFC3339)
	}
	return view
}
