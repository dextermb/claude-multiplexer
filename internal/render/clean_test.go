package render

import (
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestCleanDropsACarriageReturnAndKeepsTheWords(t *testing.T) {
	got := Clean("downloading 10%\rdownloading 99%\rdone")
	if strings.ContainsAny(got, "\r") {
		t.Fatalf("the carriage return survived: %q", got)
	}
	if !strings.Contains(got, "done") {
		t.Fatalf("the words are gone: %q", got)
	}
}

func TestCleanDropsAnEscapeSequenceAndTheOtherControls(t *testing.T) {
	got := Clean("a \x1b[31mred\x1b[0m word\x07 with \x08 a \x1b[2J screen")
	for _, bad := range []string{"\x1b", "\x07", "\x08"} {
		if strings.Contains(got, bad) {
			t.Fatalf("%q survived: %q", bad, got)
		}
	}
	if !strings.Contains(got, "red word") {
		t.Fatalf("the words are gone: %q", got)
	}
}

func TestCleanKeepsANewLineATabAndPlainText(t *testing.T) {
	text := "one\n\ttwo · три ✓"
	if got := Clean(text); got != text {
		t.Fatalf("Clean changed plain text: %q", got)
	}
}

func TestLinesCleanTheTextOfAnEvent(t *testing.T) {
	var r Renderer
	ev := session.Event{
		Kind: session.KindStderr,
		Line: "boom\rover the pane",
		At:   time.Now(),
	}
	for _, line := range r.Lines(ev) {
		if strings.Contains(line.Text, "\r") || strings.Contains(line.Summary, "\r") {
			t.Fatalf("a carriage return reached the pane: %q", line.Text)
		}
	}
}
