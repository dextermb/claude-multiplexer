package manager

import (
	"context"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/usage"
)

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

func (b *bridge) Usage() usage.Usage { return b.m.Usage() }

func (b *bridge) PeerUsage(ctx context.Context) []mcp.PeerReport { return b.m.PeerUsage(ctx) }

func (b *bridge) Peers() mcp.PeersView { return b.m.Peers() }

func (b *bridge) EnablePeering(port int) (string, error) { return b.m.EnablePeering(port) }

func (b *bridge) DisablePeering() (string, bool, error) { return b.m.DisablePeering() }

func (b *bridge) AddPeer(in mcp.PeerHostInput) (string, error) { return b.m.AddPeer(in) }

func (b *bridge) UpdatePeer(in mcp.PeerHostUpdate) (string, error) { return b.m.UpdatePeer(in) }

func (b *bridge) RemovePeer(name string) (string, bool, error) { return b.m.RemovePeer(name) }

func (b *bridge) SetReserve(window string, minPercent int) (string, error) {
	return b.m.SetReserve(window, minPercent)
}

func (b *bridge) UnsetReserve() (string, bool, error) { return b.m.UnsetReserve() }

func (b *bridge) HostingPaused() bool { return b.m.HostingPaused() }
