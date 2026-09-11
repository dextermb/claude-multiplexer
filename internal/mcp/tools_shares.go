package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type shareSessionIn struct {
	Session      string   `json:"session" jsonschema:"the session to share read-only"`
	ExpiresHours *float64 `json:"expires_hours,omitempty" jsonschema:"hours until the share expires; omit for the default 24, or 0 for no expiry"`
}

type revokeShareIn struct {
	ID string `json:"id" jsonschema:"the id of the share to revoke, from list_shares"`
}

type watchShareIn struct {
	Link string `json:"link" jsonschema:"a cmux://spectate/... link a host gave you"`
}

// watchShareOut is the result of watch_share: the local name of the read-only
// session, so the human finds it in the sidebar.
type watchShareOut struct {
	OK      bool   `json:"ok"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// shareListOut is the result of list_shares.
type shareListOut struct {
	Shares []ShareView `json:"shares"`
}

// revokeShareOut is the result of revoke_share.
type revokeShareOut struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (s *Server) addShareTools(server *sdk.Server, _ string) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolShareSession,
		Description: "Mint a read-only share link for a session. Anyone with the link watches the session live, but cannot type, stop, or interrupt it. The link is shown once. It expires in 24 hours by default; give expires_hours to change it, or 0 for no expiry. Peering must be on.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in shareSessionIn) (*sdk.CallToolResult, ShareCreated, error) {
		if strings.TrimSpace(in.Session) == "" {
			return nil, ShareCreated{}, ErrNoTarget
		}
		created, err := s.sessions.ShareSession(in.Session, in.ExpiresHours)
		if err != nil {
			return nil, ShareCreated{}, err
		}
		return nil, created, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListShares,
		Description: "List the active read-only shares: the id, the session, and the expiry. No secret is shown.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, shareListOut, error) {
		return nil, shareListOut{Shares: s.sessions.ListShares()}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRevokeShare,
		Description: "Revoke a read-only share by id, so its link stops working. A live viewer is dropped within a few seconds.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in revokeShareIn) (*sdk.CallToolResult, revokeShareOut, error) {
		if strings.TrimSpace(in.ID) == "" {
			return nil, revokeShareOut{}, ErrNoShare
		}
		had, err := s.sessions.RevokeShare(in.ID)
		if err != nil {
			return nil, revokeShareOut{}, err
		}
		if !had {
			return nil, revokeShareOut{OK: true, Message: "no share with id " + in.ID}, nil
		}
		return nil, revokeShareOut{OK: true, Message: "revoked the share " + in.ID}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolWatchShare,
		Description: "Watch a session another host shared, from its cmux://spectate/... link. The session appears read-only: you see it live, but you cannot type, stop, or interrupt it. Close it to stop watching; the session on the host is untouched.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in watchShareIn) (*sdk.CallToolResult, watchShareOut, error) {
		if strings.TrimSpace(in.Link) == "" {
			return nil, watchShareOut{}, ErrNoShare
		}
		name, err := s.sessions.WatchShare(in.Link)
		if err != nil {
			return nil, watchShareOut{}, err
		}
		return nil, watchShareOut{OK: true, Name: name, Message: "watching " + name + " read-only"}, nil
	})
}
