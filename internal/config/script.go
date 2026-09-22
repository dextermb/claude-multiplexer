package config

import (
	"path/filepath"
	"strings"
)

// scriptRunners maps a script extension to the command that runs it. A custom
// bar element and a key command share this map, so both run the same file types.
// See docs/config/bars.md and docs/config/commands.md.
var scriptRunners = map[string][]string{
	".go": {"go", "run"},
	".py": {"python3"},
	".sh": {"bash"},
}

// ScriptRunner gives the command and its leading arguments for a script path, by
// the file extension, and false when the extension is not one it runs.
func ScriptRunner(path string) ([]string, bool) {
	cmd, ok := scriptRunners[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return nil, false
	}
	out := make([]string, len(cmd))
	copy(out, cmd)
	return out, true
}
