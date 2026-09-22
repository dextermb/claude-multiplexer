package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func TestRunCommandScriptReadsStdinAndFirstLine(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "echo.sh")
	body := "#!/bin/sh\ngrep -o '\"bar\":\"[a-z]*\"'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	c := config.Command{Keys: "b o", Label: "browser", Script: script}
	msg, ok := runCommandScript(c, []byte(`{"bar":"status"}`), "")().(commandOutputMsg)
	if !ok {
		t.Fatalf("run gave %T, want commandOutputMsg", msg)
	}
	if msg.err != nil {
		t.Fatalf("run error: %v", msg.err)
	}
	if msg.label != "browser" || msg.text != `"bar":"status"` {
		t.Fatalf("msg = %+v, want the label and the first line", msg)
	}
}

func TestRunCommandScriptReportsAFailure(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "boom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := config.Command{Label: "boom", Script: script}
	msg := runCommandScript(c, nil, "")().(commandOutputMsg)
	if msg.err == nil {
		t.Fatal("run gave no error, want the non-zero exit reported")
	}
}

func TestHandleCommandOutputWritesTheNotice(t *testing.T) {
	var m Model
	next, _ := m.handleCommandOutput(commandOutputMsg{label: "browser", text: "opened"})
	if got := next.(Model).status; got != "browser: opened" {
		t.Fatalf("status = %q, want the notice", got)
	}
	empty, _ := m.handleCommandOutput(commandOutputMsg{label: "browser", text: ""})
	if got := empty.(Model).status; got != "" {
		t.Fatalf("status = %q, want nothing on empty output", got)
	}
}
