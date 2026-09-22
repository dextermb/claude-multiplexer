package tui

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// runScriptSync runs a script with the payload on stdin and returns its first
// stdout line. A custom bar element and a key command share it. See
// docs/config/bars.md and docs/config/commands.md.
func runScriptSync(script string, payload []byte, timeout time.Duration, baseDir string) (string, error) {
	runner, ok := config.ScriptRunner(script)
	if !ok {
		return "", &scriptError{"unknown script type"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	args := append(runner[1:], expandScriptPath(script, baseDir))
	cmd := exec.CommandContext(ctx, runner[0], args...)
	cmd.Stdin = bytes.NewReader(payload)
	if baseDir != "" {
		cmd.Dir = baseDir
	}
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return firstLine(data), nil
}

type scriptError struct{ msg string }

func (e *scriptError) Error() string { return e.msg }

// expandScriptPath resolves a script path: a leading ~ against the home
// directory, and a relative path against the settings directory.
func expandScriptPath(path, baseDir string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	if filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}

func firstLine(data []byte) string {
	text := string(data)
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	return strings.TrimSpace(text)
}
