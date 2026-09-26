package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestToolLineSetsTheNameAsALabel(t *testing.T) {
	cases := []struct{ text, want string }{
		{"→ Read internal/auth/middleware.go", "→ READ   internal/auth/middleware.go"},
		{"→ Bash go test ./...", "→ BASH   go test ./..."},
		{"→ TodoWrite 3 todos", "→ TODOWRITE 3 todos"},
		{"→ Skill", "→ SKILL"},
		{"not a tool call", "not a tool call"},
	}
	for _, c := range cases {
		got := strings.TrimRight(ansi.Strip(toolLineView(c.text, 80)), " ")
		if got != c.want {
			t.Errorf("toolLineView(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestToolLineColoursTheArgumentsApartFromTheName(t *testing.T) {
	line := toolLineView("→ Read a.go", 80)
	name := labelStyle.Render("read")
	args := toolArgsStyle.Render("a.go")
	if !strings.Contains(line, name) || !strings.Contains(line, args) {
		t.Fatalf("line = %q, want the label %q and the arguments %q", line, name, args)
	}
}

func TestRawOutputKeepsTheToolLineAsWritten(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 90, 24)
	m.showRaw = true
	got := ansi.Strip(m.wrap([]render.Line{{Class: render.ClassToolUse, Text: "→ Read a.go"}}))
	if !strings.HasPrefix(got, "→ Read a.go") {
		t.Fatalf("raw tool line = %q", got)
	}
}
