package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

func plain(text string) string {
	var out []string
	for _, line := range strings.Split(ansiCodes.ReplaceAllString(text, ""), "\n") {
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

func TestRenderDropsTheMarkupAndKeepsTheWords(t *testing.T) {
	r := New()
	out := plain(r.Render("## The cache layer\n\nThe loader has **three** problems.\n", 60))
	if strings.Contains(out, "##") {
		t.Errorf("the hashes survived:\n%s", out)
	}
	if strings.Contains(out, "**") {
		t.Errorf("the asterisks survived:\n%s", out)
	}
	for _, want := range []string{"The cache layer", "three", "problems"} {
		if !strings.Contains(out, want) {
			t.Errorf("the words were lost, %q is missing:\n%s", want, out)
		}
	}
}

func TestRenderKeepsAListAndACodeFence(t *testing.T) {
	r := New()
	out := plain(r.Render("1. first\n2. second\n\n```go\nfunc main() {}\n```\n", 60))
	for _, want := range []string{"first", "second", "func main"} {
		if !strings.Contains(out, want) {
			t.Errorf("%q is missing:\n%s", want, out)
		}
	}
	if strings.Contains(out, "```") {
		t.Errorf("the fence markers survived:\n%s", out)
	}
}

func TestRenderWrapsToTheGivenWidth(t *testing.T) {
	r := New()
	text := strings.Repeat("a long sentence that must wrap. ", 8)
	for _, width := range []int{40, 80} {
		out := r.Render(text, width)
		for _, line := range strings.Split(out, "\n") {
			if lipgloss.Width(line) > width {
				t.Errorf("width %d: a line is %d wide: %q", width, lipgloss.Width(line), plain(line))
			}
		}
	}
}

func TestRenderLeavesPlainTextReadable(t *testing.T) {
	r := New()
	out := plain(r.Render("just a plain sentence", 60))
	if !strings.Contains(out, "just a plain sentence") {
		t.Fatalf("out = %q", out)
	}
}

func TestRenderReturnsEmptyInputUntouched(t *testing.T) {
	r := New()
	if got := r.Render("   ", 60); got != "   " {
		t.Fatalf("out = %q", got)
	}
	if got := r.Render("text", 0); got != "text" {
		t.Fatalf("out = %q", got)
	}
}

func TestRenderCachesByWidth(t *testing.T) {
	r := New()
	first := r.Render("**bold**", 60)
	if got := r.Render("**bold**", 60); got != first {
		t.Fatal("the cache returned something different")
	}
	if len(r.cache) != 1 {
		t.Fatalf("cache holds %d entries, want 1", len(r.cache))
	}
	if wider := r.Render("**bold**", 90); len(r.cache) != 1 {
		t.Fatalf("a new width must clear the cache, it holds %d (%q)", len(r.cache), wider)
	}
}

func TestEveryHeadingLevelRendersTheSame(t *testing.T) {
	r := New()
	first := r.Render("# Findings", 60)
	for _, level := range []string{"## Findings", "### Findings", "###### Findings"} {
		if got := r.Render(level, 60); got != first {
			t.Errorf("%q renders differently from a level one heading:\n%q\n%q",
				level, plain(got), plain(first))
		}
	}
	if !strings.Contains(first, ";1m") && !strings.Contains(first, "[1m") {
		t.Errorf("a heading must be bold: %q", first)
	}
	if strings.Contains(first, "48;5;") {
		t.Errorf("a heading must have no background block: %q", first)
	}
	if got := plain(first); !strings.Contains(got, "Findings") || strings.Contains(got, "#") {
		t.Errorf("heading = %q", got)
	}
}

func TestRenderDoesNotEndWithBlankLines(t *testing.T) {
	r := New()
	out := r.Render("# Title\n\nBody text.\n\n\n", 60)
	lines := strings.Split(plain(out), "\n")
	if strings.TrimSpace(lines[0]) == "" {
		t.Fatalf("the render starts with a blank line: %q", plain(out))
	}
	if strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatalf("the render ends with a blank line: %q", plain(out))
	}
}

func TestUnclassifiedCharacterHasNoBackground(t *testing.T) {
	r := New()
	for _, lang := range []string{"json", "toml", "make"} {
		out := r.Render("```"+lang+"\nfoo | bar\n```", 60)
		if strings.Contains(out, "48;5;203") || strings.Contains(out, "48;2;240;91;91") {
			t.Errorf("a %s block gives a pipe a red background: %q", lang, out)
		}
		if !strings.Contains(plain(out), "foo | bar") {
			t.Errorf("a %s block lost its text: %q", lang, plain(out))
		}
	}
}

func TestPaneStyleLeavesTheSharedChromaAlone(t *testing.T) {
	paneStyle()
	if styles.DarkStyleConfig.CodeBlock.Chroma.Error.BackgroundColor == nil {
		t.Fatal("paneStyle changed the shared dark style")
	}
}
