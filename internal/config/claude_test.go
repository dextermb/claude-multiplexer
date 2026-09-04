package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeClaude(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ClaudeFileName)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheClaudePathFollowsTheConfigDirectory(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/tmp/claude")

	got := ClaudePaths()
	if len(got) != 1 || got[0] != "/tmp/claude/settings.json" {
		t.Fatalf("ClaudePaths() = %v, want the settings of that directory", got)
	}
}

func TestTheClaudePathFallsBackToDotClaude(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	want := filepath.Join(home, ".claude", ClaudeFileName)
	if got := ClaudePaths(); len(got) != 1 || got[0] != want {
		t.Fatalf("ClaudePaths() = %v, want %q", got, want)
	}
}

func TestTheClaudeSettingsGiveTheEditor(t *testing.T) {
	path := writeClaude(t, `{"model": "opus", "env": {"EDITOR": "zed --wait"}}`)

	if got := LoadClaude(path).Editor; got != "zed --wait" {
		t.Fatalf("editor = %q, want %q", got, "zed --wait")
	}
}

func TestTheClaudeSettingsPreferVisual(t *testing.T) {
	path := writeClaude(t, `{"env": {"VISUAL": "nvim", "EDITOR": "vi"}}`)

	if got := LoadClaude(path).Editor; got != "nvim" {
		t.Fatalf("editor = %q, want nvim", got)
	}
}

func TestTheClaudeSettingsGiveNothingWhenTheyCannotBeRead(t *testing.T) {
	cases := map[string]string{
		"a file that does not parse": "{env: }",
		"no environment block":       `{"model": "opus"}`,
		"an empty editor":            `{"env": {"EDITOR": "  "}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if got := LoadClaude(writeClaude(t, body)).Editor; got != "" {
				t.Fatalf("editor = %q, want none", got)
			}
		})
	}
	if got := LoadClaude(filepath.Join(t.TempDir(), "nothing.json")).Editor; got != "" {
		t.Fatalf("editor = %q, want none for a missing file", got)
	}
}

func TestTheSettingsFileBeatsTheClaudeSettings(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	claude := LoadClaude(writeClaude(t, `{"env": {"EDITOR": "zed"}}`))
	if got := Resolve(Config{}, Config{Editor: "nvim"}, claude).Editor; got != "nvim" {
		t.Fatalf("editor = %q, want the settings file", got)
	}
	if got := Resolve(Config{}, Config{}, claude).Editor; got != "zed" {
		t.Fatalf("editor = %q, want the Claude settings", got)
	}
	if got := Resolve(Config{Editor: "code"}, Config{}, claude).Editor; got != "code" {
		t.Fatalf("editor = %q, want the flag", got)
	}
}

func TestTheEnvironmentBeatsTheClaudeSettings(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vi")

	claude := LoadClaude(writeClaude(t, `{"env": {"EDITOR": "zed"}}`))
	if got := Resolve(Config{}, Config{}, claude).Editor; got != "vi" {
		t.Fatalf("editor = %q, want the environment", got)
	}
}
