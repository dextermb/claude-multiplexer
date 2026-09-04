package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

type launched struct {
	line     string
	dir      string
	terminal bool
}

// recordLaunches swaps the two launchers for one that records the command.
func recordLaunches(t *testing.T) *[]launched {
	t.Helper()
	var seen []launched
	oldTerminal, oldDetached := launchTerminal, launchDetached
	record := func(terminal bool) func(*exec.Cmd, func(error) tea.Msg) tea.Cmd {
		return func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
			seen = append(seen, launched{
				line:     strings.Join(cmd.Args, " "),
				dir:      cmd.Dir,
				terminal: terminal,
			})
			return func() tea.Msg { return done(nil) }
		}
	}
	launchTerminal, launchDetached = record(true), record(false)
	t.Cleanup(func() { launchTerminal, launchDetached = oldTerminal, oldDetached })
	return &seen
}

func openModel(t *testing.T, editor string) (Model, string) {
	t.Helper()
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	m, mgr := newTestModel(t, "")
	m.opts.Config = config.Config{Editor: editor}
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m.focus = focusSidebar
	m.prompt.Blur()
	return m, dir
}

func run(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("the action gave no command")
	}
	m, _ = step(t, m, cmd())
	return m
}

func TestTheFolderKeyOpensTheFileManager(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "")

	m, cmd := chord(t, m, "s", "f")
	m = run(t, m, cmd)

	if len(*seen) != 1 {
		t.Fatalf("launched %d programs, want 1", len(*seen))
	}
	got := (*seen)[0]
	if !strings.HasSuffix(got.line, " "+dir) {
		t.Errorf("command = %q, want it to end with the working directory %q", got.line, dir)
	}
	if got.terminal {
		t.Error("the file manager must not take the terminal")
	}
	if got.dir != dir {
		t.Errorf("cmd.Dir = %q, want %q", got.dir, dir)
	}
	if !strings.Contains(m.status, dir) {
		t.Errorf("status = %q, want the directory in it", m.status)
	}
}

func TestTheEditorKeyOpensAWindowEditor(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "code -n")

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if len(*seen) != 1 {
		t.Fatalf("launched %d programs, want 1", len(*seen))
	}
	got := (*seen)[0]
	if got.line != "code -n "+dir {
		t.Errorf("command = %q, want %q", got.line, "code -n "+dir)
	}
	if got.terminal {
		t.Error("code must not take the terminal")
	}
	if m.errText != "" {
		t.Errorf("errText = %q, want none", m.errText)
	}
}

func TestTheEditorKeyGivesATerminalEditorTheTerminal(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "nvim")

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if len(*seen) != 1 {
		t.Fatalf("launched %d programs, want 1", len(*seen))
	}
	if got := (*seen)[0]; !got.terminal || got.line != "nvim "+dir {
		t.Fatalf("launched %+v, want nvim in the terminal", got)
	}
	if m.status != "" {
		t.Errorf("status = %q, want none after a terminal editor", m.status)
	}
}

func TestTheEditorKeyReportsThatNoEditorIsSet(t *testing.T) {
	seen := recordLaunches(t)
	m, _ := openModel(t, "")

	m, cmd := chord(t, m, "s", "d")
	if cmd != nil {
		t.Fatal("no editor must start nothing")
	}
	if len(*seen) != 0 {
		t.Fatalf("launched %d programs, want none", len(*seen))
	}
	if !strings.Contains(m.errText, "no editor") {
		t.Fatalf("errText = %q, want it to say that no editor is set", m.errText)
	}
}

func TestAFailedLaunchShowsTheError(t *testing.T) {
	m, _ := openModel(t, "zed")

	m, _ = step(t, m, openedMsg{what: "editor", dir: "/tmp", err: exec.ErrNotFound})
	if !strings.Contains(m.errText, "editor: ") {
		t.Fatalf("errText = %q, want the failure of the editor", m.errText)
	}
}

func TestTheKeyListNamesBothKeys(t *testing.T) {
	m, _ := openModel(t, "")
	m, _ = step(t, m, key("?"))

	view := m.View()
	for _, want := range []string{"s f", "s d", "file manager", "editor"} {
		if !strings.Contains(view, want) {
			t.Errorf("the key list does not name %q:\n%s", want, view)
		}
	}
}

func TestTheDetachedLauncherStartsTheProgram(t *testing.T) {
	dir := t.TempDir()
	stamp := dir + "/stamp"
	cmd := exec.Command("sh", "-c", "echo hello > "+stamp)

	msg := launchDetached(cmd, func(err error) tea.Msg {
		return openedMsg{what: "editor", dir: dir, err: err}
	})()
	opened, ok := msg.(openedMsg)
	if !ok {
		t.Fatalf("msg = %T, want openedMsg", msg)
	}
	if opened.err != nil {
		t.Fatalf("err = %v, want none", opened.err)
	}
}

func TestTheDetachedLauncherReportsAMissingProgram(t *testing.T) {
	cmd := exec.Command("no-such-program-multiplexier")

	msg := launchDetached(cmd, func(err error) tea.Msg {
		return openedMsg{what: "editor", err: err}
	})()
	if opened := msg.(openedMsg); opened.err == nil {
		t.Fatal("a program that does not start must give an error")
	}
}

// settingsFile writes a settings file, and points the model at it.
func settingsFile(t *testing.T, m Model, body string) Model {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	m.opts.ConfigPaths = []string{path}
	return m
}

func TestTheEditorKeyReadsTheSettingsFile(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "")
	m = settingsFile(t, m, `{"editor": "zed"}`)

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if len(*seen) != 1 {
		t.Fatalf("launched %d programs, want 1", len(*seen))
	}
	if got := (*seen)[0].line; got != "zed "+dir {
		t.Fatalf("command = %q, want %q", got, "zed "+dir)
	}
}

func TestTheEditorKeyFollowsAFileWrittenAfterTheStart(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "")
	m = settingsFile(t, m, `{"editor": "zed"}`)

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if err := os.WriteFile(m.opts.ConfigPaths[0], []byte(`{"editor": "nvim"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, cmd = chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if len(*seen) != 2 {
		t.Fatalf("launched %d programs, want 2", len(*seen))
	}
	second := (*seen)[1]
	if second.line != "nvim "+dir {
		t.Fatalf("command = %q, want %q", second.line, "nvim "+dir)
	}
	if !second.terminal {
		t.Error("nvim must take the terminal")
	}
}

func TestTheFlagStillBeatsTheSettingsFile(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "code")
	m = settingsFile(t, m, `{"editor": "zed"}`)

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if got := (*seen)[0].line; got != "code "+dir {
		t.Fatalf("command = %q, want the flag %q", got, "code "+dir)
	}
}

func TestABadSettingsFileShowsTheReason(t *testing.T) {
	seen := recordLaunches(t)
	m, _ := openModel(t, "")
	m = settingsFile(t, m, "{editor: zed}")

	m, cmd := chord(t, m, "s", "d")
	if cmd != nil {
		t.Fatal("a file that does not parse must start nothing")
	}
	if len(*seen) != 0 {
		t.Fatalf("launched %d programs, want none", len(*seen))
	}
	if !strings.Contains(m.errText, "config.json") {
		t.Fatalf("errText = %q, want the name of the file in it", m.errText)
	}
}

// claudeFile writes a Claude Code settings file, and points the model at it.
func claudeFile(t *testing.T, m Model, body string) Model {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m.opts.ClaudePaths = []string{path}
	return m
}

func TestTheEditorKeyFallsBackToTheClaudeSettings(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "")
	m = claudeFile(t, m, `{"env": {"EDITOR": "zed --wait"}}`)

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if len(*seen) != 1 {
		t.Fatalf("launched %d programs, want 1", len(*seen))
	}
	if got := (*seen)[0].line; got != "zed --wait "+dir {
		t.Fatalf("command = %q, want %q", got, "zed --wait "+dir)
	}
}

func TestTheSettingsFileBeatsTheClaudeSettingsAtTheKeyPress(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "")
	m = settingsFile(t, m, `{"editor": "nvim"}`)
	m = claudeFile(t, m, `{"env": {"EDITOR": "zed"}}`)

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if got := (*seen)[0].line; got != "nvim "+dir {
		t.Fatalf("command = %q, want the settings file %q", got, "nvim "+dir)
	}
}

func TestTheKeysOpenTheWorkingDirectory(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "code")
	work := filepath.Join(dir, ".worktrees", "feature")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := m.mgr.SetWorkingDir(m.sel, ".worktrees/feature"); err != nil {
		t.Fatalf("SetWorkingDir: %v", err)
	}
	m.refresh()

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)
	m, cmd = chord(t, m, "s", "f")
	m = run(t, m, cmd)

	if len(*seen) != 2 {
		t.Fatalf("launched %d programs, want 2", len(*seen))
	}
	for _, got := range *seen {
		if !strings.HasSuffix(got.line, " "+work) {
			t.Errorf("command = %q, want it to end with the working directory %q", got.line, work)
		}
		if got.dir != work {
			t.Errorf("cmd.Dir = %q, want %q", got.dir, work)
		}
	}
}

func TestTheKeysFallBackWhenTheWorkingDirectoryIsGone(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "code")
	work := filepath.Join(dir, ".worktrees", "feature")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := m.mgr.SetWorkingDir(m.sel, ".worktrees/feature"); err != nil {
		t.Fatalf("SetWorkingDir: %v", err)
	}
	m.refresh()
	if err := os.RemoveAll(work); err != nil {
		t.Fatal(err)
	}

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if got := (*seen)[0].line; got != "code "+dir {
		t.Fatalf("command = %q, want the directory the session started in %q", got, "code "+dir)
	}
}

func TestTheKeysUseTheStartDirectoryWithNoWorkingDirectory(t *testing.T) {
	seen := recordLaunches(t)
	m, dir := openModel(t, "code")

	m, cmd := chord(t, m, "s", "d")
	m = run(t, m, cmd)

	if got := (*seen)[0].line; got != "code "+dir {
		t.Fatalf("command = %q, want %q", got, "code "+dir)
	}
}
