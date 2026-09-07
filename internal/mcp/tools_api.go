package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// addCredentialTools gives a control session the tools that manage the API: the
// admin secret, and the clients. A secret is shown once, in the tool result. See
// docs/mcp/api.md.
func (s *Server) addCredentialTools(server *sdk.Server, _ string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolCreateAPIAdmin,
		Description: "Turn the external API on: create the admin secret that gates the client-management endpoint. " +
			"It returns the secret once, and the base URL. The API stays off until this secret exists.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, adminSecretOut, error) {
		secret, err := s.sessions.CreateAPIAdmin()
		if err != nil {
			return nil, adminSecretOut{}, err
		}
		endpoint := s.sessions.APIEndpoint()
		return nil, adminSecretOut{OK: true, AdminSecret: secret, URL: endpoint.URL, Message: "the API is on"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRotateAPIAdmin,
		Description: "Replace the admin secret. The old secret stops working at once. It returns the new secret once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, adminSecretOut, error) {
		secret, err := s.sessions.RotateAPIAdmin()
		if err != nil {
			return nil, adminSecretOut{}, err
		}
		return nil, adminSecretOut{OK: true, AdminSecret: secret, Message: "the admin secret is rotated"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRevokeAPIAdmin,
		Description: "Turn the external API off: remove the admin secret. The clients stay on disk for a later create.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, okOut, error) {
		if err := s.sessions.RevokeAPIAdmin(); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: "the API is off"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreateAPIClient,
		Description: "Create an API client. It returns the client id and the client secret once. A client reaches only the sessions it creates.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createClientIn) (*sdk.CallToolResult, clientSecretOut, error) {
		client, secret, err := s.sessions.CreateAPIClient(strings.TrimSpace(in.Name))
		if err != nil {
			return nil, clientSecretOut{}, err
		}
		return nil, clientSecretOut{OK: true, ClientID: client.ClientID, Name: client.Name, ClientSecret: secret, Message: "created the client " + client.Name}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolUpdateAPIClient,
		Description: "Change a client: rename it, or disable it. A disabled client cannot grant a new token, and its live tokens stop at once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in updateClientIn) (*sdk.CallToolResult, clientOut, error) {
		id := strings.TrimSpace(in.ClientID)
		if id == "" {
			return nil, clientOut{}, ErrNoClient
		}
		client, err := s.sessions.UpdateAPIClient(id, in.Name, in.Disabled)
		if err != nil {
			return nil, clientOut{}, err
		}
		return nil, clientOut{OK: true, Client: client, Message: "updated the client " + client.Name}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRotateAPIClient,
		Description: "Issue a new secret for a client. The old secret stops working, and its live tokens stop at once. It returns the new secret once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in clientIDIn) (*sdk.CallToolResult, clientSecretOut, error) {
		id := strings.TrimSpace(in.ClientID)
		if id == "" {
			return nil, clientSecretOut{}, ErrNoClient
		}
		secret, err := s.sessions.RotateAPIClient(id)
		if err != nil {
			return nil, clientSecretOut{}, err
		}
		return nil, clientSecretOut{OK: true, ClientID: id, ClientSecret: secret, Message: "rotated the client secret"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRevokeAPIClient,
		Description: "Remove a client. Its live tokens stop at once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in clientIDIn) (*sdk.CallToolResult, okOut, error) {
		id := strings.TrimSpace(in.ClientID)
		if id == "" {
			return nil, okOut{}, ErrNoClient
		}
		if err := s.sessions.RevokeAPIClient(id); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: "removed the client"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListAPIClients,
		Description: "List every API client, with its id, its name, and whether it is disabled. It never shows a secret.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, listClientsOut, error) {
		return nil, listClientsOut{Clients: s.sessions.ListAPIClients()}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolAPIEndpoint,
		Description: "The base URL of the API, and the port range it binds inside. A client reads it to reach the token, admin, and REST surfaces.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, APIEndpoint, error) {
		return nil, s.sessions.APIEndpoint(), nil
	})
}

type createClientIn struct {
	Name string `json:"name" jsonschema:"a human label for the client, shown in the interface and the list"`
}

type updateClientIn struct {
	ClientID string  `json:"client_id" jsonschema:"the id of the client to change"`
	Name     *string `json:"name,omitempty" jsonschema:"a new name; a field left out stays as it is"`
	Disabled *bool   `json:"disabled,omitempty" jsonschema:"true blocks the client from a new token without deleting it; a field left out stays as it is"`
}

type clientIDIn struct {
	ClientID string `json:"client_id" jsonschema:"the id of the client"`
}

type adminSecretOut struct {
	OK          bool   `json:"ok"`
	AdminSecret string `json:"admin_secret"`
	URL         string `json:"url,omitempty"`
	Message     string `json:"message"`
}

type clientSecretOut struct {
	OK           bool   `json:"ok"`
	ClientID     string `json:"client_id"`
	Name         string `json:"name,omitempty"`
	ClientSecret string `json:"client_secret"`
	Message      string `json:"message"`
}

type clientOut struct {
	OK      bool      `json:"ok"`
	Client  APIClient `json:"client"`
	Message string    `json:"message"`
}

type listClientsOut struct {
	Clients []APIClient `json:"clients"`
}
