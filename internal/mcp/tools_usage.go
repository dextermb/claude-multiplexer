package mcp

import (
	"context"

	"github.com/dextermb/claude-multiplexer/internal/usage"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// peerUsageOut is the output of peer_usage: this host's own usage, and a report
// for every configured peer.
type peerUsageOut struct {
	Self  usage.Usage  `json:"self"`
	Peers []PeerReport `json:"peers"`
}

func (s *Server) addUsageTools(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolGetUsage,
		Description: "Read this host's Claude usage, the stat Claude Code shows in /usage: the percent left and the reset of the 5-hour and 7-day windows. The value comes from a cached poll, so it may lag a little.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, usage.Usage, error) {
		return nil, s.sessions.Usage(), nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolPeerUsage,
		Description: "Read the Claude usage of this host and every configured peer. A peer that is off or unreachable reports reachable=false with the error, and never blocks the others.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, peerUsageOut, error) {
		return nil, peerUsageOut{Self: s.sessions.Usage(), Peers: s.sessions.PeerUsage(ctx)}, nil
	})
}
