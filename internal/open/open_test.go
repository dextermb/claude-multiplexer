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
