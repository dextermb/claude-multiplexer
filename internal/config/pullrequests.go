package config

import "strings"

// The provider a pull request belongs to. See docs/pull-requests.md.
const (
	PullRequestGitHub = "github"
	PullRequestGitLab = "gitlab"
)

// The default GraphQL endpoint of each provider. A provider entry with no url
// takes its default. See docs/pull-requests.md.
const (
	DefaultGitHubGraphQLURL = "https://api.github.com/graphql"
	DefaultGitLabGraphQLURL = "https://gitlab.com/api/graphql"
)

// The default host of each provider, matched against the git remote to infer
// the provider. A hosts entry adds to these. See docs/pull-requests.md.
const (
	DefaultGitHubHost = "github.com"
	DefaultGitLabHost = "gitlab.com"
)

// The transport a provider uses. Auto takes the API when a token is set, else
// the CLI when the tool is on the PATH. See docs/pull-requests.md.
const (
	PullRequestModeAuto = "auto"
	PullRequestModeAPI  = "api"
	PullRequestModeCLI  = "cli"
)

// PullRequests configures the pull-request providers, keyed by provider. Both
// may be set at once. See docs/pull-requests.md.
type PullRequests struct {
	GitHub *PRProvider `json:"github,omitempty"`
	GitLab *PRProvider `json:"gitlab,omitempty"`
}

// PRProvider holds one provider's token, transport mode, GraphQL endpoint, and
// extra hosts. An empty token, with mode api, keeps the provider off. See
// docs/pull-requests.md.
type PRProvider struct {
	Token string   `json:"token,omitempty"`
	Mode  string   `json:"mode,omitempty"`
	URL   string   `json:"url,omitempty"`
	Hosts []string `json:"hosts,omitempty"`
}

// Provider reads one provider's settings by name, or nil when it is not set.
func (p *PullRequests) Provider(name string) *PRProvider {
	if p == nil {
		return nil
	}
	switch name {
	case PullRequestGitHub:
		return p.GitHub
	case PullRequestGitLab:
		return p.GitLab
	}
	return nil
}

// SetProvider writes one provider entry, and makes the block when there is
// none. It returns the changed PullRequests, so a caller writes the settings.
func (p *PullRequests) SetProvider(name string, entry *PRProvider) *PullRequests {
	if p == nil {
		p = &PullRequests{}
	}
	switch name {
	case PullRequestGitHub:
		p.GitHub = entry
	case PullRequestGitLab:
		p.GitLab = entry
	}
	return p
}

// Mode gives the transport of a provider: its own mode, or auto when it names
// none.
func (p *PullRequests) Mode(name string) string {
	entry := p.Provider(name)
	if entry == nil {
		return PullRequestModeAuto
	}
	mode := strings.TrimSpace(strings.ToLower(entry.Mode))
	switch mode {
	case PullRequestModeAPI, PullRequestModeCLI:
		return mode
	default:
		return PullRequestModeAuto
	}
}

// Endpoint gives the GraphQL url of a provider: its own url, or the provider
// default when it names none.
func (p *PullRequests) Endpoint(name string) string {
	entry := p.Provider(name)
	if entry != nil && strings.TrimSpace(entry.URL) != "" {
		return strings.TrimSpace(entry.URL)
	}
	switch name {
	case PullRequestGitHub:
		return DefaultGitHubGraphQLURL
	case PullRequestGitLab:
		return DefaultGitLabGraphQLURL
	}
	return ""
}

// ProviderForHost infers the provider from a remote host: the default hosts, or
// an extra host of a configured provider. It returns "" for an unknown host.
func (p *PullRequests) ProviderForHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	switch host {
	case DefaultGitHubHost:
		return PullRequestGitHub
	case DefaultGitLabHost:
		return PullRequestGitLab
	}
	if hostInList(host, p.Provider(PullRequestGitHub)) {
		return PullRequestGitHub
	}
	if hostInList(host, p.Provider(PullRequestGitLab)) {
		return PullRequestGitLab
	}
	return ""
}

func hostInList(host string, entry *PRProvider) bool {
	if entry == nil {
		return false
	}
	for _, h := range entry.Hosts {
		if strings.EqualFold(strings.TrimSpace(h), host) {
			return true
		}
	}
	return false
}

// ValidPullRequestProvider reports whether a name is a provider the program
// knows.
func ValidPullRequestProvider(name string) bool {
	return name == PullRequestGitHub || name == PullRequestGitLab
}
