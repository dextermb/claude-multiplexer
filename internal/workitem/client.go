package workitem

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// clientName and clientVersion name the multiplexer to a provider server on the
// MCP handshake.
const (
	clientName    = "cmux"
	clientVersion = "0.1.0"
)

// dialTimeout bounds one provider operation: the handshake and the tool calls.
const dialTimeout = 30 * time.Second

// connector holds the endpoint and the Authorization header of one provider. It
// opens a fresh MCP connection per operation, so no dead connection is kept. See
// docs/work-items.md.
type connector struct {
	endpoint      string
	authorization string
}

// conn is one live MCP session to a provider server.
type conn struct {
	session *sdk.ClientSession
}

// open connects to the provider server and completes the MCP handshake.
func (c connector) open(ctx context.Context) (*conn, error) {
	httpClient := &http.Client{
		Timeout:   dialTimeout,
		Transport: &authTransport{base: http.DefaultTransport, authorization: c.authorization},
	}
	client := sdk.NewClient(&sdk.Implementation{Name: clientName, Version: clientVersion}, nil)
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{
		Endpoint:   c.endpoint,
		HTTPClient: httpClient,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("workitem: connect %s: %w", c.endpoint, err)
	}
	return &conn{session: session}, nil
}

func (c *conn) close() {
	_ = c.session.Close()
}

// call runs one tool and returns its result as JSON. It prefers the structured
// content, and falls back to the text content. An error result becomes a Go
// error with the text the server returned.
func (c *conn) call(ctx context.Context, name string, args any) (json.RawMessage, error) {
	res, err := c.session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, fmt.Errorf("workitem: call %s: %w", name, err)
	}
	if res.IsError {
		return nil, fmt.Errorf("workitem: %s: %s", name, resultText(res))
	}
	return resultJSON(res)
}

// resultJSON reads a tool result as JSON: the structured content when the server
// sends it, else the text content parsed as JSON.
func resultJSON(res *sdk.CallToolResult) (json.RawMessage, error) {
	if res.StructuredContent != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	text := strings.TrimSpace(resultText(res))
	if text == "" {
		return json.RawMessage("null"), nil
	}
	if json.Valid([]byte(text)) {
		return json.RawMessage(text), nil
	}
	quoted, err := json.Marshal(text)
	if err != nil {
		return nil, err
	}
	return quoted, nil
}

// resultText joins the text content blocks of a result.
func resultText(res *sdk.CallToolResult) string {
	var parts []string
	for _, content := range res.Content {
		if text, ok := content.(*sdk.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// authTransport adds the provider Authorization header to every request.
type authTransport struct {
	base          http.RoundTripper
	authorization string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.authorization != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", t.authorization)
	}
	return t.base.RoundTrip(req)
}

// authorization builds the Authorization header for a provider. An email selects
// Basic auth (email:token), the Jira personal token. With no email the client
// sends a bearer token. See docs/work-items.md.
func authorization(p *config.WorkItemProvider) string {
	token := strings.TrimSpace(p.Token)
	if token == "" {
		return ""
	}
	if email := strings.TrimSpace(p.Email); email != "" {
		raw := email + ":" + token
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
	}
	return "Bearer " + token
}
