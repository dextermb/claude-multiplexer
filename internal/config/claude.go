package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ClaudeFileName is the settings file of Claude Code.
const ClaudeFileName = "settings.json"

// ClaudePaths lists the settings files of Claude Code to look for. It reads
// $CLAUDE_CONFIG_DIR, and falls back to ~/.claude.
func ClaudePaths() []string {
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		dir = filepath.Join(home, ".claude")
	}
	return []string{filepath.Join(dir, ClaudeFileName)}
}

// claudeSettings is the part of the Claude Code settings this program reads.
type claudeSettings struct {
	Env map[string]string `json:"env"`
}

// LoadClaude reads the editor out of the environment block of the Claude Code
// settings. A file that is not there, or that does not parse, gives an empty
// configuration, because the file belongs to another program.
func LoadClaude(paths ...string) Config {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var settings claudeSettings
		if err := json.Unmarshal(data, &settings); err != nil {
			continue
		}
		if editor := firstOf(settings.Env["VISUAL"], settings.Env["EDITOR"]); editor != "" {
			return Config{Editor: editor}
		}
	}
	return Config{}
}
