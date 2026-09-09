package session

import (
	"slices"
	"testing"
)

func TestBuildEnvOverridesAndScrubs(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"ANTHROPIC_API_KEY=borrower-key",
		"HOME=/home/borrower",
	}
	add := []string{
		"CLAUDE_CODE_OAUTH_TOKEN=lender-token",
		"CLAUDE_CONFIG_DIR=/state/hoisted/studio",
	}
	scrub := []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"}

	got := buildEnv(base, add, scrub)

	if has(got, "ANTHROPIC_API_KEY", "borrower-key") {
		t.Error("the borrower's ANTHROPIC_API_KEY must be scrubbed")
	}
	if count(got, "ANTHROPIC_API_KEY") != 0 {
		t.Error("no ANTHROPIC_API_KEY must remain when only a token is injected")
	}
	if !has(got, "CLAUDE_CODE_OAUTH_TOKEN", "lender-token") {
		t.Error("the injected token must be present")
	}
	if !has(got, "CLAUDE_CONFIG_DIR", "/state/hoisted/studio") {
		t.Error("the injected config dir must be present")
	}
	if !has(got, "PATH", "/usr/bin") || !has(got, "HOME", "/home/borrower") {
		t.Error("an unrelated inherited variable must survive")
	}
}

func TestBuildEnvOverrideWinsForSameKey(t *testing.T) {
	base := []string{"CLAUDE_CODE_OAUTH_TOKEN=old"}
	add := []string{"CLAUDE_CODE_OAUTH_TOKEN=new"}
	got := buildEnv(base, add, nil)
	if count(got, "CLAUDE_CODE_OAUTH_TOKEN") != 1 {
		t.Fatalf("a key set by both base and add must appear once, got %v", got)
	}
	if !has(got, "CLAUDE_CODE_OAUTH_TOKEN", "new") {
		t.Error("the added value must win over the inherited one")
	}
}

func TestBuildEnvNoScrubNoAdd(t *testing.T) {
	base := []string{"PATH=/usr/bin"}
	got := buildEnv(base, nil, nil)
	if !slices.Equal(got, base) {
		t.Errorf("an empty add and scrub must leave base unchanged, got %v", got)
	}
}

func has(env []string, key, value string) bool {
	return slices.Contains(env, key+"="+value)
}

func count(env []string, key string) int {
	n := 0
	for _, entry := range env {
		if envKey(entry) == key {
			n++
		}
	}
	return n
}
