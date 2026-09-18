package mcp

import (
	"context"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) addReadTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRename,
		Description: "Set the display title of this session, so the human sees what it works on.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in renameIn) (*sdk.CallToolResult, okOut, error) {
		if err := s.sessions.SetTitle(caller, in.Title); err != nil {
			return nil, okOut{}, err
		}
		if in.Title == "" {
			return nil, okOut{OK: true, Message: "the title of " + caller + " is cleared"}, nil
		}
		return nil, okOut{OK: true, Message: caller + " is now titled " + in.Title}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolList,
		Description: "List the sessions the multiplexer runs now. Set stopped true to add the stored sessions, and archived true to add the archived ones. last_active drops a stored or archived session older than its window (1d, 1w, 1m, 1y, or unset; 1d by default).",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listOut, error) {
		kept, err := in.filter(s.sessions.List(), time.Now())
		if err != nil {
			return nil, listOut{}, err
		}
		return nil, listOut{Sessions: kept}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListInactive,
		Description: "List the stored sessions that do not run now and are not archived. last_active drops a session older than its window (1d, 1w, 1m, 1y, or unset; 1d by default).",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listCategoryIn) (*sdk.CallToolResult, listOut, error) {
		kept, err := in.filter(s.sessions.List(), time.Now(), isInactive)
		if err != nil {
			return nil, listOut{}, err
		}
		return nil, listOut{Sessions: kept}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListArchived,
		Description: "List the archived sessions. last_active drops a session older than its window (1d, 1w, 1m, 1y, or unset; 1d by default).",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listCategoryIn) (*sdk.CallToolResult, listOut, error) {
		kept, err := in.filter(s.sessions.List(), time.Now(), isArchived)
		if err != nil {
			return nil, listOut{}, err
		}
		return nil, listOut{Sessions: kept}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolMessages,
		Description: "Read the recent messages of a session, oldest first.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in messagesIn) (*sdk.CallToolResult, messagesOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, messagesOut{}, err
		}
		messages, err := s.sessions.Messages(target, clampLimit(in.Limit))
		if err != nil {
			return nil, messagesOut{}, err
		}
		return nil, messagesOut{Session: target, Messages: messages}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListJobs,
		Description: "List the background jobs of a session: id, description, task type, and status (running, done, failed, or killed). Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listJobsIn) (*sdk.CallToolResult, listJobsOut, error) {
		target, err := targetOrSelf(in.Session, caller)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		jobs, err := s.sessions.Jobs(target)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		return nil, listJobsOut{Session: target, Jobs: jobs}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolAPIURL,
		Description: "The base URL of the external API, on loopback. It grants nothing without a client secret. A control session manages the clients; see " + ToolListAPIClients + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, apiURLOut, error) {
		return nil, apiURLOut{URL: s.sessions.APIEndpoint().URL}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolPeerURL,
		Description: "The URL a peer on the network dials to reach this host, with the host and port inside it. The host is a routable LAN address, not the 0.0.0.0 the listener binds. It is empty when peering is off; a control session turns it on with " + ToolEnablePeering + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, peerURLOut, error) {
		e := s.sessions.PeerEndpoint()
		return nil, peerURLOut{URL: e.URL, Host: e.Host, Port: e.Port, Enabled: e.Enabled}, nil
	})
}
