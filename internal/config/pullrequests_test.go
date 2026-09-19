package config

import "testing"

func TestPullRequestEndpointFallsBackToTheDefault(t *testing.T) {
	pr := &PullRequests{GitHub: &PRProvider{Token: "t"}}
	if got := pr.Endpoint(PullRequestGitHub); got != DefaultGitHubGraphQLURL {
		t.Fatalf("github endpoint = %q, want the default", got)
	}
	pr.GitHub.URL = "https://ghe.corp/api/graphql"
	if got := pr.Endpoint(PullRequestGitHub); got != "https://ghe.corp/api/graphql" {
		t.Fatalf("github endpoint = %q, want the override", got)
	}
	if got := pr.Endpoint(PullRequestGitLab); got != DefaultGitLabGraphQLURL {
		t.Fatalf("gitlab endpoint = %q, want the default", got)
	}
}

func TestPullRequestModeDefaultsToAuto(t *testing.T) {
	var pr *PullRequests
	if got := pr.Mode(PullRequestGitHub); got != PullRequestModeAuto {
		t.Fatalf("nil mode = %q, want auto", got)
	}
	pr = &PullRequests{GitLab: &PRProvider{Mode: "CLI"}}
	if got := pr.Mode(PullRequestGitLab); got != PullRequestModeCLI {
		t.Fatalf("mode = %q, want cli", got)
	}
	pr.GitLab.Mode = "nonsense"
	if got := pr.Mode(PullRequestGitLab); got != PullRequestModeAuto {
		t.Fatalf("bad mode = %q, want auto", got)
	}
}

func TestProviderForHostReadsDefaultsAndExtras(t *testing.T) {
	pr := &PullRequests{
		GitHub: &PRProvider{Token: "t", Hosts: []string{"github.corp"}},
		GitLab: &PRProvider{Token: "t", Hosts: []string{"gitlab.corp"}},
	}
	cases := map[string]string{
		"github.com":  PullRequestGitHub,
		"GitHub.com":  PullRequestGitHub,
		"gitlab.com":  PullRequestGitLab,
		"github.corp": PullRequestGitHub,
		"gitlab.corp": PullRequestGitLab,
		"example.com": "",
		"":            "",
	}
	for host, want := range cases {
		if got := pr.ProviderForHost(host); got != want {
			t.Errorf("host %q = %q, want %q", host, got, want)
		}
	}
}

func TestSetProviderMakesTheBlock(t *testing.T) {
	var pr *PullRequests
	pr = pr.SetProvider(PullRequestGitHub, &PRProvider{Token: "t"})
	if pr == nil || pr.GitHub == nil || pr.GitHub.Token != "t" {
		t.Fatalf("set github = %+v, want a token", pr)
	}
}
