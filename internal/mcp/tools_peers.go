package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type enablePeeringIn struct {
	Port int `json:"port,omitempty" jsonschema:"the port the peer listener binds on 0.0.0.0; omit for the default 51900"`
}

type addPeerIn struct {
	Name         string `json:"name" jsonschema:"the label for the peer"`
	URL          string `json:"url" jsonschema:"the base URL of the peer's peer listener, e.g. http://192.168.1.20:51900"`
	ClientID     string `json:"client_id" jsonschema:"the client id the peer provisioned for this host"`
	ClientSecret string `json:"client_secret" jsonschema:"the client secret the peer provisioned for this host"`
}

type removePeerIn struct {
	Name string `json:"name" jsonschema:"the name of the peer to remove"`
}

type setReserveIn struct {
	Window     string `json:"window" jsonschema:"the window the reserve guards: 5h or 7d"`
	MinPercent int    `json:"min_percent" jsonschema:"the percent-remaining floor the gate trips below"`
}

// peerOut is the result of a peer-config change: the path written and a message.
type peerOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

func (s *Server) addPeerTools(server *sdk.Server, _ string) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListPeers,
		Description: "List the peer settings: whether peering is on and its port, the reserve, and the peer hosts. No secret is shown.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, PeersView, error) {
		return nil, s.sessions.Peers(), nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolEnablePeering,
		Description: "Turn the peer listener on. It binds 0.0.0.0 so peers on the network reach this host. Give a port to override the default 51900. It takes effect on the next restart.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in enablePeeringIn) (*sdk.CallToolResult, peerOut, error) {
		path, err := s.sessions.EnablePeering(in.Port)
		if err != nil {
			return nil, peerOut{}, err
		}
		message := "peering is on for the next restart"
		if in.Port > 0 {
			message = fmt.Sprintf("peering is on, on port %d, for the next restart", in.Port)
		}
		return nil, peerOut{OK: true, Path: path, Message: message}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolDisablePeering,
		Description: "Turn the peer listener off, so the peering surface stays loopback-only.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, peerOut, error) {
		path, had, err := s.sessions.DisablePeering()
		if err != nil {
			return nil, peerOut{}, err
		}
		if !had {
			return nil, peerOut{OK: true, Path: path, Message: "peering was already off"}, nil
		}
		return nil, peerOut{OK: true, Path: path, Message: "peering is off on the next restart"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolAddPeer,
		Description: "Add a peer host this host reaches, with the credentials the peer provisioned. It takes effect on the next restart.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in addPeerIn) (*sdk.CallToolResult, peerOut, error) {
		if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.URL) == "" {
			return nil, peerOut{}, ErrNoPeer
		}
		path, err := s.sessions.AddPeer(PeerHostInput{Name: in.Name, URL: in.URL, ClientID: in.ClientID, ClientSecret: in.ClientSecret})
		if err != nil {
			return nil, peerOut{}, err
		}
		return nil, peerOut{OK: true, Path: path, Message: "added the peer " + in.Name}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRemovePeer,
		Description: "Remove a peer host by name.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in removePeerIn) (*sdk.CallToolResult, peerOut, error) {
		if strings.TrimSpace(in.Name) == "" {
			return nil, peerOut{}, ErrNoPeer
		}
		path, had, err := s.sessions.RemovePeer(in.Name)
		if err != nil {
			return nil, peerOut{}, err
		}
		if !had {
			return nil, peerOut{OK: true, Path: path, Message: "no peer named " + in.Name}, nil
		}
		return nil, peerOut{OK: true, Path: path, Message: "removed the peer " + in.Name}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSetReserve,
		Description: "Set the usage reserve: the window (5h or 7d) and the percent-remaining floor. Below the floor, this host pauses its hosted sessions after their turn and refuses a new peer session. It takes effect on the next poll.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setReserveIn) (*sdk.CallToolResult, peerOut, error) {
		path, err := s.sessions.SetReserve(in.Window, in.MinPercent)
		if err != nil {
			return nil, peerOut{}, err
		}
		return nil, peerOut{OK: true, Path: path, Message: "the reserve guards the " + in.Window + " window"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolUnsetReserve,
		Description: "Clear the usage reserve, so this host hosts with no floor.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, peerOut, error) {
		path, had, err := s.sessions.UnsetReserve()
		if err != nil {
			return nil, peerOut{}, err
		}
		if !had {
			return nil, peerOut{OK: true, Path: path, Message: "there was no reserve"}, nil
		}
		return nil, peerOut{OK: true, Path: path, Message: "the reserve is cleared"}, nil
	})
}
