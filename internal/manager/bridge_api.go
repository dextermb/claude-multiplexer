package manager

import "github.com/dextermb/claude-multiplexer/internal/mcp"

func (b *bridge) CreateAPIAdmin() (string, error) { return b.m.CreateAPIAdmin() }

func (b *bridge) RotateAPIAdmin() (string, error) { return b.m.RotateAPIAdmin() }

func (b *bridge) RevokeAPIAdmin() error { return b.m.RevokeAPIAdmin() }

func (b *bridge) CreateAPIClient(name string) (mcp.APIClient, string, error) {
	return b.m.CreateAPIClient(name)
}

func (b *bridge) UpdateAPIClient(id string, name *string, disabled *bool) (mcp.APIClient, error) {
	return b.m.UpdateAPIClient(id, name, disabled)
}

func (b *bridge) RotateAPIClient(id string) (string, error) { return b.m.RotateAPIClient(id) }

func (b *bridge) RevokeAPIClient(id string) error { return b.m.RevokeAPIClient(id) }

func (b *bridge) ListAPIClients() []mcp.APIClient { return b.m.ListAPIClients() }

func (b *bridge) APIEndpoint() mcp.APIEndpoint { return b.m.APIEndpoint() }
