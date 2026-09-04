package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name, FileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathsFollowTheConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/settings")

	want := []string{
		"/tmp/settings/multiplexer/config.json",
		"/tmp/settings/multiplexier/config.json",
	}
	got := Paths()
	if len(got) != len(want) {
		t.Fatalf("Paths() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Paths()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPathsFallBackToDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	want := filepath.Join(home, ".config", "multiplexer", FileName)
	if got := Paths()[0]; got != want {
		t.Fatalf("Paths()[0] = %q, want %q", got, want)
	}
}

func TestTheFirstSpellingWins(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "multiplexer", `{"editor": "nvim"}`)
	write(t, dir, "multiplexier", `{"editor": "zed"}`)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load(Paths()...)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "nvim" {
		t.Fatalf("editor = %q, want the multiplexer spelling", cfg.Editor)
	}
}

func TestTheOtherSpellingIsRead(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "multiplexier", `{"editor": "zed"}`)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load(Paths()...)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "zed" {
		t.Fatalf("editor = %q, want zed", cfg.Editor)
	}
}

func TestAMissingFileIsNotAnError(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nothing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "" {
		t.Fatalf("editor = %q, want an empty configuration", cfg.Editor)
	}
}

func TestAnEmptyFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "multiplexer", "  \n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "" {
		t.Fatalf("editor = %q, want an empty configuration", cfg.Editor)
	}
}

func TestABadFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "multiplexer", "{editor: nvim}")

	if _, err := Load(path); err == nil {
		t.Fatal("a file that does not parse must give an error")
	}
}

func TestTheEditorTerminalField(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "multiplexer", `{"editor": "myed", "editorTerminal": true}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EditorTerminal == nil || !*cfg.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want true", cfg.EditorTerminal)
	}
}

func TestTheLadderOrder(t *testing.T) {
	file := Config{Editor: "from-file"}

	cases := []struct {
		name   string
		flag   string
		visual string
		editor string
		want   string
	}{
		{"the flag wins", "from-flag", "from-visual", "from-editor", "from-flag"},
		{"then VISUAL", "", "from-visual", "from-editor", "from-visual"},
		{"then EDITOR", "", "", "from-editor", "from-editor"},
		{"then the file", "", "", "", "from-file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VISUAL", tc.visual)
			t.Setenv("EDITOR", tc.editor)

			if got := Resolve(Config{Editor: tc.flag}, file).Editor; got != tc.want {
				t.Fatalf("editor = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheFlagOverridesTheEditorTerminalField(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	yes, no := true, false
	got := Resolve(Config{EditorTerminal: &no}, Config{Editor: "myed", EditorTerminal: &yes})
	if got.EditorTerminal == nil || *got.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want false", got.EditorTerminal)
	}
	if got.Editor != "myed" {
		t.Fatalf("editor = %q, want myed", got.Editor)
	}
}
