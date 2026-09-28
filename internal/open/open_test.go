package open

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func TestTheFileManagerOfThePlatform(t *testing.T) {
	target := FileManager("/tmp/work")

	want := map[string]string{"darwin": "open", "windows": "explorer"}[runtime.GOOS]
	if want == "" {
		want = "xdg-open"
	}
	if target.Command != want {
		t.Fatalf("command = %q, want %q", target.Command, want)
	}
	if len(target.Args) != 1 || target.Args[0] != "/tmp/work" {
		t.Fatalf("args = %v, want the directory", target.Args)
	}
	if target.Terminal {
		t.Fatal("the file manager must not take the terminal")
	}
}

func TestTheBrowserOfThePlatform(t *testing.T) {
	target := Browser("https://host/pr/1")

	want := map[string]string{"darwin": "open", "windows": "explorer"}[runtime.GOOS]
	if want == "" {
		want = "xdg-open"
	}
	if target.Command != want {
		t.Fatalf("command = %q, want %q", target.Command, want)
	}
	if len(target.Args) != 1 || target.Args[0] != "https://host/pr/1" {
		t.Fatalf("args = %v, want the url", target.Args)
	}
	if target.Terminal {
		t.Fatal("the browser must not take the terminal")
	}
}

func TestTheEditorTakesTheDirectoryLast(t *testing.T) {
	target, err := Editor(config.Config{Editor: "code -n"}, "/tmp/work")
	if err != nil {
		t.Fatal(err)
	}
	if target.Command != "code" {
		t.Fatalf("command = %q, want code", target.Command)
	}
	if got := strings.Join(target.Args, " "); got != "-n /tmp/work" {
		t.Fatalf("args = %q, want %q", got, "-n /tmp/work")
	}
	if target.Terminal {
		t.Fatal("code must not take the terminal")
	}
}

func TestAKnownTerminalEditorTakesTheTerminal(t *testing.T) {
	for _, name := range []string{"vim", "nvim", "/usr/local/bin/nvim", "hx", "emacs -nw"} {
		target, err := Editor(config.Config{Editor: name}, "/tmp/work")
		if err != nil {
			t.Fatal(err)
		}
		if !target.Terminal {
			t.Errorf("%q must take the terminal", name)
		}
	}
}

func TestAnUnknownEditorIsAWindowProgram(t *testing.T) {
	for _, name := range []string{"zed", "code", "subl", "/Applications/x.app/Contents/MacOS/x"} {
		target, err := Editor(config.Config{Editor: name}, "/tmp/work")
		if err != nil {
			t.Fatal(err)
		}
		if target.Terminal {
			t.Errorf("%q must not take the terminal", name)
		}
	}
}

func TestTheConfigurationOverridesTheList(t *testing.T) {
	no := false
	target, err := Editor(config.Config{Editor: "vim", EditorTerminal: &no}, "/tmp/work")
	if err != nil {
		t.Fatal(err)
	}
	if target.Terminal {
		t.Fatal("editorTerminal false must beat the list")
	}

	yes := true
	target, err = Editor(config.Config{Editor: "myed", EditorTerminal: &yes}, "/tmp/work")
	if err != nil {
		t.Fatal(err)
	}
	if !target.Terminal {
		t.Fatal("editorTerminal true must beat the list")
	}
}

func TestNoEditorIsAnError(t *testing.T) {
	if _, err := Editor(config.Config{}, "/tmp/work"); !errors.Is(err, ErrNoEditor) {
		t.Fatalf("err = %v, want ErrNoEditor", err)
	}
	if _, err := Editor(config.Config{Editor: "   "}, "/tmp/work"); !errors.Is(err, ErrNoEditor) {
		t.Fatalf("err = %v, want ErrNoEditor", err)
	}
}

func TestAProjectEditorTakesEveryDirectoryAsOneWindow(t *testing.T) {
	targets, err := Editors(config.Config{Editor: "zed"}, []string{"/tmp/one", "/tmp/two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("built %d commands, want one project window", len(targets))
	}
	got := targets[0]
	if got.Command != "zed" {
		t.Fatalf("command = %q, want zed", got.Command)
	}
	if line := strings.Join(got.Args, " "); line != "/tmp/one /tmp/two" {
		t.Fatalf("args = %q, want both directories", line)
	}
	if got.Dir != "/tmp/one" {
		t.Fatalf("dir = %q, want the first directory", got.Dir)
	}
	if got.Terminal {
		t.Fatal("zed must not take the terminal")
	}
}

func TestAProjectEditorKeepsItsFlags(t *testing.T) {
	targets, err := Editors(config.Config{Editor: "/opt/bin/zed --wait"}, []string{"/tmp/one", "/tmp/two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("built %d commands, want one project window", len(targets))
	}
	if line := strings.Join(targets[0].Args, " "); line != "--wait /tmp/one /tmp/two" {
		t.Fatalf("args = %q, want the flag then both directories", line)
	}
}

func TestAWindowEditorGetsOneCommandForEachDirectory(t *testing.T) {
	targets, err := Editors(config.Config{Editor: "code -n"}, []string{"/tmp/one", "/tmp/two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("built %d commands, want one for each directory", len(targets))
	}
	if line := strings.Join(targets[0].Args, " "); line != "-n /tmp/one" {
		t.Fatalf("the first args = %q, want %q", line, "-n /tmp/one")
	}
	if line := strings.Join(targets[1].Args, " "); line != "-n /tmp/two" {
		t.Fatalf("the second args = %q, want %q", line, "-n /tmp/two")
	}
	if targets[1].Dir != "/tmp/two" {
		t.Fatalf("the second dir = %q, want %q", targets[1].Dir, "/tmp/two")
	}
}

func TestATerminalEditorGetsOneCommandForEveryDirectory(t *testing.T) {
	targets, err := Editors(config.Config{Editor: "nvim"}, []string{"/tmp/one", "/tmp/two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("built %d commands, want one that holds the terminal", len(targets))
	}
	if line := strings.Join(targets[0].Args, " "); line != "/tmp/one /tmp/two" {
		t.Fatalf("args = %q, want both directories", line)
	}
	if !targets[0].Terminal {
		t.Fatal("nvim must take the terminal")
	}
}

func TestOneDirectoryGivesOneCommand(t *testing.T) {
	for _, editor := range []string{"zed", "code", "nvim"} {
		targets, err := Editors(config.Config{Editor: editor}, []string{"/tmp/work"})
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 || strings.Join(targets[0].Args, " ") != "/tmp/work" {
			t.Errorf("%q built %v, want one command for the one directory", editor, targets)
		}
	}
}

func TestNoEditorIsAnErrorForTheSet(t *testing.T) {
	if _, err := Editors(config.Config{}, []string{"/tmp/one"}); !errors.Is(err, ErrNoEditor) {
		t.Fatalf("err = %v, want ErrNoEditor", err)
	}
	targets, err := Editors(config.Config{}, nil)
	if err != nil || targets != nil {
		t.Fatalf("no directory gave %v, %v, want nothing", targets, err)
	}
}

func TestTheProjectEditorList(t *testing.T) {
	for _, name := range []string{"zed", "/usr/local/bin/zed", "zed --wait", "zeditor"} {
		if !IsProjectEditor(name) {
			t.Errorf("%q must open several directories as one project", name)
		}
	}
	for _, name := range []string{"", "code", "nvim", "subl"} {
		if IsProjectEditor(name) {
			t.Errorf("%q must not open several directories as one project", name)
		}
	}
}
