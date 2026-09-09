package manager

import (
	"errors"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

// errBadCredentialType is returned when a lent-credential type is not token or
// key. See docs/peers/hoisted.md.
var errBadCredentialType = errors.New("manager: the credential type must be token or key")

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

// CreateAPIKey records a Claude credential lent to a client, found by id or
// name. It stores only the metadata; the value is echoed by the tool. See
// docs/peers/hoisted.md.
func (m *Manager) CreateAPIKey(client, credentialType, value string) (mcp.APIClient, error) {
	if m.apiStore == nil {
		return mcp.APIClient{}, ErrNoAPIStore
	}
	if !config.ValidCredentialType(credentialType) {
		return mcp.APIClient{}, errBadCredentialType
	}
	updated, err := m.apiStore.SetLentKey(client, credentialType, value)
	if err != nil {
		return mcp.APIClient{}, err
	}
	return apiClientView(updated), nil
}

// RevokeAPIKey clears the lent-credential metadata from a client, and reports
// whether it had one. It does not stop the credential at Anthropic, and it does
// not retract a copy a peer already holds. See docs/peers/hoisted.md.
func (m *Manager) RevokeAPIKey(client string) (bool, error) {
	if m.apiStore == nil {
		return false, ErrNoAPIStore
	}
	_, had, err := m.apiStore.ClearLentKey(client)
	return had, err
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
	if client.LentKey != nil {
		view.LentKey = &mcp.APILentKey{Type: client.LentKey.Type, Last4: client.LentKey.Last4}
		if !client.LentKey.IssuedAt.IsZero() {
			view.LentKey.IssuedAt = client.LentKey.IssuedAt.Format(time.RFC3339)
		}
	}
	return view
}
