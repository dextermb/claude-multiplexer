package mcp

import (
	"context"
	"strconv"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// addPullRequestConfigTools registers the always-open configure tools. A session
// calls one to set a provider token or mode, so it can turn the feature on. See
// docs/pull-requests.md.
func (s *Server) addPullRequestConfigTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolConfigureGitHub,
		Description: "Configure the GitHub pull-request provider, so this multiplexer can read the pull request of a branch. " +
			"Give a token with repo read scope for the API, or leave it empty and set mode cli to use the gh CLI login. " +
			"It writes the settings file.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in configureGitHubIn) (*sdk.CallToolResult, configurePROut, error) {
		out, err := configureGitHub(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolConfigureGitLab,
		Description: "Configure the GitLab pull-request provider, so this multiplexer can read the merge request of a branch. " +
			"Give a token with read_api scope for the API, or leave it empty and set mode cli to use the glab CLI login. " +
			"It writes the settings file.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in configureGitLabIn) (*sdk.CallToolResult, configurePROut, error) {
		out, err := configureGitLab(s.sessions, caller, in)
		return nil, out, err
	})
}

func (s *Server) addPullRequestTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolGetPR,
		Description: "The pull request of a session's branch, with its number, state, url, and count of unresolved review threads. " +
			"Give a session name, or leave it empty for this session.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in getPRIn) (*sdk.CallToolResult, pullRequestOut, error) {
		out, err := getPR(ctx, s.sessions, caller, in)
		return nil, out, err
	})
}

func configureGitHub(port PullRequestPort, caller string, in configureGitHubIn) (configurePROut, error) {
	return configurePullRequest(port, caller, "github", in.Token, in.Mode, in.URL)
}

func configureGitLab(port PullRequestPort, caller string, in configureGitLabIn) (configurePROut, error) {
	return configurePullRequest(port, caller, "gitlab", in.Token, in.Mode, in.URL)
}

func configurePullRequest(port PullRequestPort, caller, provider, token, mode, url string) (configurePROut, error) {
	path, err := port.ConfigurePullRequest(provider, strings.TrimSpace(token), strings.TrimSpace(mode), strings.TrimSpace(url), caller)
	if err != nil {
		return configurePROut{}, err
	}
	return configurePROut{OK: true, Provider: provider, Path: path,
		Message: provider + " is configured, in " + path + "; a new session carries the pull-request tools"}, nil
}

func getPR(ctx context.Context, port PullRequestPort, caller string, in getPRIn) (pullRequestOut, error) {
	target, err := targetOrSelf(in.Session, caller)
	if err != nil {
		return pullRequestOut{}, err
	}
	pr, err := port.PullRequest(ctx, target)
	if err != nil {
		return pullRequestOut{}, err
	}
	return pullRequestOut{OK: true, Found: pr.Found, PR: pr, Message: prMessage(target, pr)}, nil
}

func prMessage(session string, pr PullRequest) string {
	if !pr.Found {
		return session + " has no pull request for its branch"
	}
	msg := session + " has PR " + strconv.Itoa(pr.Number) + " (" + pr.State + ")"
	if pr.Unresolved > 0 {
		msg += ", with " + strconv.Itoa(pr.Unresolved) + " unresolved threads"
	}
	return msg
}
