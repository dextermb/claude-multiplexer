package manager

import (
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func managerWithToolProfile(t *testing.T, profile string) *Manager {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), config.FileName)
	if err := config.Write(configPath, config.Config{DefaultToolProfile: profile}); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m, err := New(Options{
		Root:        t.TempDir(),
		ClaudePath:  fakeClaude,
		ConfigPaths: []string{configPath},
		Renderer:    render.Renderer{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestResolveProfileTakesTheSetting(t *testing.T) {
	m := managerWithToolProfile(t, "minimal")
	if got := m.resolveProfile(""); got != mcp.ProfileMinimal {
		t.Fatalf("resolveProfile = %q, want the setting", got)
	}
}

func TestResolveProfilePrefersTheSpec(t *testing.T) {
	m := managerWithToolProfile(t, "minimal")
	if got := m.resolveProfile("standard"); got != mcp.ProfileStandard {
		t.Fatalf("resolveProfile = %q, want the profile of the spec", got)
	}
}

func TestResolveProfileFallsBackWhenTheSettingIsNotAProfile(t *testing.T) {
	m := managerWithToolProfile(t, "everything")
	if got := m.resolveProfile(""); got != mcp.DefaultProfile {
		t.Fatalf("resolveProfile = %q, want the default", got)
	}
}

func TestResolveProfileTakesTheDefaultWithoutASetting(t *testing.T) {
	m := managerWithToolProfile(t, "")
	if got := m.resolveProfile(""); got != mcp.DefaultProfile {
		t.Fatalf("resolveProfile = %q, want the default", got)
	}
}

func TestTheMinimalProfileNarrowsTheAllowedTools(t *testing.T) {
	m := withMCP(t)

	var minimal, standard session.Config
	if _, err := m.equipTools(&minimal, "small", mcp.ProfileMinimal, false); err != nil {
		t.Fatalf("equipTools minimal: %v", err)
	}
	if _, err := m.equipTools(&standard, "big", mcp.ProfileStandard, false); err != nil {
		t.Fatalf("equipTools standard: %v", err)
	}
	if len(minimal.AllowedTools) >= len(standard.AllowedTools) {
		t.Fatalf("minimal allowed %d tools, standard allowed %d, want fewer",
			len(minimal.AllowedTools), len(standard.AllowedTools))
	}
	if got, want := len(minimal.AllowedTools), len(mcp.MinimalTools); got != want {
		t.Fatalf("minimal allowed %d tools, want %d", got, want)
	}
}
