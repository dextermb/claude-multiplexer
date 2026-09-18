package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type shareSessionIn struct {
	Session      string   `json:"session,omitempty" jsonschema:"the session to share read-only; omit for this session. A plain session may share only itself."`
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

// addShareSessionTool registers share_session for every session. A control
// session shares any session; a plain session shares only itself, which is the
// default when the session argument is empty. See docs/peers/spectate.md.
func (s *Server) addShareSessionTool(server *sdk.Server, caller string, control bool) {
	description := "Mint a read-only share link for a session. Anyone with the link watches the session live, but cannot type, stop, or interrupt it. The link is shown once. It expires in 24 hours by default; give expires_hours to change it, or 0 for no expiry. Peering must be on."
	if control {
		description += " Name any session, or omit it for this one."
	} else {
		description += " It shares this session; a session may share only itself."
	}
	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolShareSession,
		Description: description,
	}, func(_ context.Context, _ *sdk.CallToolRequest, in shareSessionIn) (*sdk.CallToolResult, ShareCreated, error) {
		out, err := shareSession(s.sessions, caller, control, in)
		return nil, out, err
	})
}

// addShareAdminTools registers the share tools a control session holds: list,
// revoke, and watch. See docs/peers/spectate.md.
func (s *Server) addShareAdminTools(server *sdk.Server, _ string) {
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
		out, err := revokeShare(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolWatchShare,
		Description: "Watch a session another host shared, from its cmux://spectate/... link. The session appears read-only: you see it live, but you cannot type, stop, or interrupt it. Close it to stop watching; the session on the host is untouched.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in watchShareIn) (*sdk.CallToolResult, watchShareOut, error) {
		out, err := watchShare(s.sessions, in)
		return nil, out, err
	})
}

func shareSession(share SharePort, caller string, control bool, in shareSessionIn) (ShareCreated, error) {
	target := strings.TrimSpace(in.Session)
	if target == "" {
		target = caller
	}
	if !control && target != caller {
		return ShareCreated{}, ErrNotSelf
	}
	return share.ShareSession(target, in.ExpiresHours)
}

func revokeShare(share SharePort, in revokeShareIn) (revokeShareOut, error) {
	if strings.TrimSpace(in.ID) == "" {
		return revokeShareOut{}, ErrNoShare
	}
	had, err := share.RevokeShare(in.ID)
	if err != nil {
		return revokeShareOut{}, err
	}
	if !had {
		return revokeShareOut{OK: true, Message: "no share with id " + in.ID}, nil
	}
	return revokeShareOut{OK: true, Message: "revoked the share " + in.ID}, nil
}

func watchShare(share SharePort, in watchShareIn) (watchShareOut, error) {
	if strings.TrimSpace(in.Link) == "" {
		return watchShareOut{}, ErrNoShare
	}
	name, err := share.WatchShare(in.Link)
	if err != nil {
		return watchShareOut{}, err
	}
	return watchShareOut{OK: true, Name: name, Message: "watching " + name + " read-only"}, nil
}
