package manager

import (
	"context"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func (b *bridge) PullRequestsEnabled() bool { return b.m.PullRequestsEnabled() }

func (b *bridge) ConfigurePullRequest(provider, token, mode, url, by string) (string, error) {
	path, err := b.m.ConfigurePullRequest(provider, token, mode, url)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" configured the "+provider+" pull-request provider", true)
	return path, nil
}

func (b *bridge) PullRequestsFor(ctx context.Context, session string) ([]mcp.PullRequest, error) {
	return b.m.PullRequestsFor(ctx, session)
}
