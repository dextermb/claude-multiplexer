package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// addAPIDocsTool gives every session a structured description of the REST
// surface: the authentication, the routes, and their bodies and headers. See
// docs/mcp/api.md.
func (s *Server) addAPIDocsTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolAPIDocs,
		Description: "Describe the external REST API: the authentication, the guarded and unguarded routes, " +
			"their expected headers and bodies, and a short description of each endpoint.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, apiDocsOut, error) {
		return nil, apiDocs(s.sessions.APIEndpoint().URL), nil
	})
}

type apiDocsOut struct {
	BaseURL         string        `json:"base_url"`
	Authentication  apiAuthDocs   `json:"authentication"`
	UnguardedRoutes []apiRouteDoc `json:"unguarded_routes"`
	GuardedRoutes   []apiRouteDoc `json:"guarded_routes"`
}

type apiAuthDocs struct {
	Summary     string `json:"summary"`
	AdminSecret string `json:"admin_secret"`
	AccessToken string `json:"access_token"`
}

type apiRouteDoc struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        string `json:"auth"`
	Headers     string `json:"headers,omitempty"`
	Body        string `json:"body,omitempty"`
	Description string `json:"description"`
}

func apiDocs(baseURL string) apiDocsOut {
	return apiDocsOut{
		BaseURL: baseURL,
		Authentication: apiAuthDocs{
			Summary: "The API has two secret layers. The admin secret gates client management. " +
				"A client exchanges its id and secret at POST /token for a short-lived access token, " +
				"and the token reaches the REST surface and the MCP endpoint.",
			AdminSecret: "Send the admin secret as a bearer token in the Authorization header. " +
				"It gates the /admin/clients routes only.",
			AccessToken: "Run the grant at POST /token to get an access token. Send it as a bearer token " +
				"in the Authorization header. It reaches /api and /mcp, scoped to the client's own sessions.",
		},
		UnguardedRoutes: []apiRouteDoc{{
			Method:      "POST",
			Path:        "/token",
			Auth:        "the client id and secret, in the body",
			Headers:     "Content-Type: application/x-www-form-urlencoded, or application/json",
			Body:        "grant_type=client_credentials, client_id, client_secret",
			Description: "Exchange the client id and secret for a short-lived access token.",
		}},
		GuardedRoutes: []apiRouteDoc{
			{
				Method:      "GET",
				Path:        "/admin/clients",
				Auth:        "the admin secret",
				Headers:     "Authorization: Bearer <admin secret>",
				Description: "List every client. No secret is shown.",
			},
			{
				Method:      "POST",
				Path:        "/admin/clients",
				Auth:        "the admin secret",
				Headers:     "Authorization: Bearer <admin secret>",
				Body:        "name",
				Description: "Create a client. It returns the client id and secret once.",
			},
			{
				Method:      "PATCH",
				Path:        "/admin/clients/{id}",
				Auth:        "the admin secret",
				Headers:     "Authorization: Bearer <admin secret>",
				Body:        "name (optional), disabled (optional)",
				Description: "Rename a client, or disable it. A disabled client cannot grant a new token.",
			},
			{
				Method:      "POST",
				Path:        "/admin/clients/{id}/rotate",
				Auth:        "the admin secret",
				Headers:     "Authorization: Bearer <admin secret>",
				Description: "Issue a new secret for a client, returned once. The old secret stops at once.",
			},
			{
				Method:      "DELETE",
				Path:        "/admin/clients/{id}",
				Auth:        "the admin secret",
				Headers:     "Authorization: Bearer <admin secret>",
				Description: "Remove a client. Its live tokens stop at once.",
			},
			{
				Method:      "GET",
				Path:        "/api/usage",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "Read this host's Claude usage, the stat Claude Code shows in /usage. A peer reads it to share usage.",
			},
			{
				Method:      "GET",
				Path:        "/api/sessions",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "List the sessions the client owns.",
			},
			{
				Method:      "POST",
				Path:        "/api/sessions",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Body:        "dir (required), name (optional)",
				Description: "Create a session, owned by the client.",
			},
			{
				Method:      "PATCH",
				Path:        "/api/sessions/{name}",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Body:        "title",
				Description: "Set the display title of a session.",
			},
			{
				Method:      "GET",
				Path:        "/api/sessions/{name}/messages",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "Read the recent messages of a session, oldest first. The query parameter limit caps the count.",
			},
			{
				Method:      "GET",
				Path:        "/api/sessions/{name}/jobs",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "List the background jobs of a session.",
			},
			{
				Method:      "POST",
				Path:        "/api/sessions/{name}/message",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Body:        "text",
				Description: "Queue a prompt for a session.",
			},
			{
				Method:      "POST",
				Path:        "/api/sessions/{name}/stop",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "Stop a running session.",
			},
			{
				Method:      "POST",
				Path:        "/api/sessions/{name}/interrupt",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "Interrupt the running turn of a session.",
			},
			{
				Method:      "POST",
				Path:        "/api/sessions/{name}/archive",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Body:        "restore (optional)",
				Description: "Archive a session, or restore it when restore is true.",
			},
			{
				Method:      "POST",
				Path:        "/api/sessions/{name}/jobs/{id}/stop",
				Auth:        "an access token",
				Headers:     "Authorization: Bearer <access token>",
				Description: "Stop a background job of a session.",
			},
			{
				Method:      "POST",
				Path:        "/mcp",
				Auth:        "a session token, or an access token",
				Headers:     "Authorization: Bearer <token>",
				Description: "Reach the session tools over JSON-RPC (MCP). A client holds the session-only tool set, scoped to its own sessions.",
			},
		},
	}
}
