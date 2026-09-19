package pullrequest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// dialTimeout bounds one batched GraphQL request.
const dialTimeout = 30 * time.Second

// transport sends a GraphQL query and returns the response "data" object. A
// query on many aliases returns nulls for the aliases with no match, so the
// caller reads the data even when some aliases error.
type transport interface {
	graphql(ctx context.Context, query string, vars map[string]string) (json.RawMessage, error)
}

// graphqlResp is the envelope both platforms return.
type graphqlResp struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// decodeGraphQL reads the envelope. It returns the data object, and an error
// only when the data is absent, so partial data survives a per-alias error.
func decodeGraphQL(body []byte) (json.RawMessage, error) {
	var resp graphqlResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("pullrequest: decode: %w", err)
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("pullrequest: %s", resp.Errors[0].Message)
		}
		return nil, errors.New("pullrequest: the response held no data")
	}
	return resp.Data, nil
}

// apiTransport sends the query over HTTP to the GraphQL endpoint, with a bearer
// token.
type apiTransport struct {
	endpoint string
	token    string
	client   *http.Client
}

func (t *apiTransport) graphql(ctx context.Context, query string, vars map[string]string) (json.RawMessage, error) {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)
	client := t.client
	if client == nil {
		client = &http.Client{Timeout: dialTimeout}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pullrequest: %s: %w", t.endpoint, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pullrequest: %s: status %d", t.endpoint, res.StatusCode)
	}
	return decodeGraphQL(body)
}

// cliTransport sends the query through the provider CLI: gh api graphql, or
// glab api graphql. The CLI carries the user login. See docs/pull-requests.md.
type cliTransport struct {
	tool string // "gh" or "glab"
	host string
}

func (t *cliTransport) graphql(ctx context.Context, query string, vars map[string]string) (json.RawMessage, error) {
	args := []string{"api", "graphql", "-f", "query=" + query}
	for k, v := range vars {
		args = append(args, "-f", k+"="+v)
	}
	var env []string
	switch t.tool {
	case "gh":
		if t.host != "" && t.host != config.DefaultGitHubHost {
			args = append(args, "--hostname", t.host)
		}
	case "glab":
		if t.host != "" && t.host != config.DefaultGitLabHost {
			env = append(env, "GITLAB_HOST="+t.host)
		}
	}
	out, err := runCLI(ctx, env, t.tool, args...)
	if err != nil {
		return nil, fmt.Errorf("pullrequest: %s: %w", t.tool, err)
	}
	return decodeGraphQL(out)
}

// runCLI runs a provider CLI and returns its standard output. It is a variable
// so a test records the arguments instead of running the binary.
var runCLI = func(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd.Output()
}

// lookPath finds a binary on the PATH. It is a variable so a test forces the
// answer.
var lookPath = exec.LookPath

func cliTool(provider string) string {
	if provider == config.PullRequestGitLab {
		return "glab"
	}
	return "gh"
}

func cliAvailable(provider string) bool {
	_, err := lookPath(cliTool(provider))
	return err == nil
}

func defaultHost(provider string) string {
	if provider == config.PullRequestGitLab {
		return config.DefaultGitLabHost
	}
	return config.DefaultGitHubHost
}

// usable reports whether a provider has a transport: a token for the API, or the
// CLI on the PATH, in line with the mode.
func (s *Set) usable(provider string) bool {
	_, ok := s.transportFor(provider, defaultHost(provider))
	return ok
}

// transportFor chooses the transport for a provider and a host. Auto takes the
// API when a token is set, else the CLI when the tool is on the PATH.
func (s *Set) transportFor(provider, host string) (transport, bool) {
	entry := s.cfg.Provider(provider)
	if entry == nil {
		return nil, false
	}
	hasToken := strings.TrimSpace(entry.Token) != ""
	api := func() (transport, bool) {
		return &apiTransport{endpoint: s.cfg.Endpoint(provider), token: strings.TrimSpace(entry.Token)}, true
	}
	cli := func() (transport, bool) {
		return &cliTransport{tool: cliTool(provider), host: host}, true
	}
	switch s.cfg.Mode(provider) {
	case config.PullRequestModeAPI:
		if hasToken {
			return api()
		}
	case config.PullRequestModeCLI:
		if cliAvailable(provider) {
			return cli()
		}
	default:
		if hasToken {
			return api()
		}
		if cliAvailable(provider) {
			return cli()
		}
	}
	return nil, false
}
