package manager

import (
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

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
