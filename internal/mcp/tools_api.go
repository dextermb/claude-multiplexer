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
		out, err := createAPIAdmin(s.sessions)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRotateAPIAdmin,
		Description: "Replace the admin secret. The old secret stops working at once. It returns the new secret once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, adminSecretOut, error) {
		out, err := rotateAPIAdmin(s.sessions)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRevokeAPIAdmin,
		Description: "Turn the external API off: remove the admin secret. The clients stay on disk for a later create.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, okOut, error) {
		out, err := revokeAPIAdmin(s.sessions)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreateAPIClient,
		Description: "Create an API client. It returns the client id and the client secret once. A client reaches only the sessions it creates.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createClientIn) (*sdk.CallToolResult, clientSecretOut, error) {
		out, err := createAPIClient(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolUpdateAPIClient,
		Description: "Change a client: rename it, or disable it. A disabled client cannot grant a new token, and its live tokens stop at once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in updateClientIn) (*sdk.CallToolResult, clientOut, error) {
		out, err := updateAPIClient(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRotateAPIClient,
		Description: "Issue a new secret for a client. The old secret stops working, and its live tokens stop at once. It returns the new secret once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in clientIDIn) (*sdk.CallToolResult, clientSecretOut, error) {
		out, err := rotateAPIClient(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRevokeAPIClient,
		Description: "Remove a client. Its live tokens stop at once.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in clientIDIn) (*sdk.CallToolResult, okOut, error) {
		out, err := revokeAPIClient(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListAPIClients,
		Description: "List every API client, with its id, its name, whether it is disabled, and whether it lent a Claude credential (the type and last four, never the value). It never shows a secret.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, listClientsOut, error) {
		return nil, listClientsOut{Clients: s.sessions.ListAPIClients()}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolCreateAPIKey,
		Description: "Lend a Claude credential to an existing API client, so the client may run a hoisted session as this host. " +
			"Give the client (id or name), the type (token for a Claude access token, or key for an Anthropic API key), and the value. " +
			"It stores only the type and the last four characters, and returns the value once to share with the peer.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createAPIKeyIn) (*sdk.CallToolResult, apiKeyOut, error) {
		out, err := createAPIKey(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRevokeAPIKey,
		Description: "Remove the lent Claude credential from a client (id or name). It stops this host from lending it, but it does not stop the credential at Anthropic, and it does not retract a copy the peer already holds.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in clientRefIn) (*sdk.CallToolResult, okOut, error) {
		out, err := revokeAPIKey(s.sessions, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolAPIEndpoint,
		Description: "The base URL of the API, and the port range it binds inside. A client reads it to reach the token, admin, and REST surfaces.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, APIEndpoint, error) {
		return nil, s.sessions.APIEndpoint(), nil
	})
}

func createAPIAdmin(api APIPort) (adminSecretOut, error) {
	secret, err := api.CreateAPIAdmin()
	if err != nil {
		return adminSecretOut{}, err
	}
	return adminSecretOut{OK: true, AdminSecret: secret, URL: api.APIEndpoint().URL, Message: "the API is on"}, nil
}

func rotateAPIAdmin(api APIPort) (adminSecretOut, error) {
	secret, err := api.RotateAPIAdmin()
	if err != nil {
		return adminSecretOut{}, err
	}
	return adminSecretOut{OK: true, AdminSecret: secret, Message: "the admin secret is rotated"}, nil
}

func revokeAPIAdmin(api APIPort) (okOut, error) {
	if err := api.RevokeAPIAdmin(); err != nil {
		return okOut{}, err
	}
	return okOut{OK: true, Message: "the API is off"}, nil
}

func createAPIClient(api APIPort, in createClientIn) (clientSecretOut, error) {
	client, secret, err := api.CreateAPIClient(strings.TrimSpace(in.Name))
	if err != nil {
		return clientSecretOut{}, err
	}
	return clientSecretOut{OK: true, ClientID: client.ClientID, Name: client.Name, ClientSecret: secret, Message: "created the client " + client.Name}, nil
}

func updateAPIClient(api APIPort, in updateClientIn) (clientOut, error) {
	id := strings.TrimSpace(in.ClientID)
	if id == "" {
		return clientOut{}, ErrNoClient
	}
	client, err := api.UpdateAPIClient(id, in.Name, in.Disabled)
	if err != nil {
		return clientOut{}, err
	}
	return clientOut{OK: true, Client: client, Message: "updated the client " + client.Name}, nil
}

func rotateAPIClient(api APIPort, in clientIDIn) (clientSecretOut, error) {
	id := strings.TrimSpace(in.ClientID)
	if id == "" {
		return clientSecretOut{}, ErrNoClient
	}
	secret, err := api.RotateAPIClient(id)
	if err != nil {
		return clientSecretOut{}, err
	}
	return clientSecretOut{OK: true, ClientID: id, ClientSecret: secret, Message: "rotated the client secret"}, nil
}

func revokeAPIClient(api APIPort, in clientIDIn) (okOut, error) {
	id := strings.TrimSpace(in.ClientID)
	if id == "" {
		return okOut{}, ErrNoClient
	}
	if err := api.RevokeAPIClient(id); err != nil {
		return okOut{}, err
	}
	return okOut{OK: true, Message: "removed the client"}, nil
}

func createAPIKey(api APIPort, in createAPIKeyIn) (apiKeyOut, error) {
	client := strings.TrimSpace(in.Client)
	if client == "" {
		return apiKeyOut{}, ErrNoClient
	}
	ktype := strings.TrimSpace(in.Type)
	value := strings.TrimSpace(in.Value)
	if value == "" {
		return apiKeyOut{}, ErrNoCredential
	}
	updated, err := api.CreateAPIKey(client, ktype, value)
	if err != nil {
		return apiKeyOut{}, err
	}
	out := apiKeyOut{OK: true, ClientID: updated.ClientID, Name: updated.Name, Value: value, Message: "lent a " + ktype + " to " + updated.Name}
	if updated.LentKey != nil {
		out.Type = updated.LentKey.Type
		out.Last4 = updated.LentKey.Last4
	}
	return out, nil
}

func revokeAPIKey(api APIPort, in clientRefIn) (okOut, error) {
	client := strings.TrimSpace(in.Client)
	if client == "" {
		return okOut{}, ErrNoClient
	}
	had, err := api.RevokeAPIKey(client)
	if err != nil {
		return okOut{}, err
	}
	if !had {
		return okOut{OK: true, Message: "the client had no lent credential"}, nil
	}
	return okOut{OK: true, Message: "removed the lent credential"}, nil
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

type createAPIKeyIn struct {
	Client string `json:"client" jsonschema:"the API client to lend to, by id or by name"`
	Type   string `json:"type" jsonschema:"the credential type: token for a Claude access token, or key for an Anthropic API key"`
	Value  string `json:"value" jsonschema:"the credential value; it is stored as metadata only, and echoed once to share"`
}

type clientRefIn struct {
	Client string `json:"client" jsonschema:"the API client, by id or by name"`
}

type apiKeyOut struct {
	OK       bool   `json:"ok"`
	ClientID string `json:"client_id"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Last4    string `json:"last4,omitempty"`
	Value    string `json:"value"`
	Message  string `json:"message"`
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
