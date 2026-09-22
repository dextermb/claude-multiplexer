package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/markdown"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

// crlfHunk is the diff of a file with Windows line endings.
const crlfHunk = "@@ -1,2 +1,3 @@ func A()\r\n one\r\n+two\r\n three\r\n"

// TestFrameHoldsNoControlCharacter guards the review screen: a carriage return
// in the text moves the cursor to the start of the row, so the explanation
// paints over the diff. See docs/tui/output.md.
func TestFrameHoldsNoControlCharacter(t *testing.T) {
	m := reviewModel()
	m.ready = true
	m.md = markdown.New()
	m.mdMuted = markdown.NewMuted()
	next, _ := m.handleFileDiff(fileDiffMsg{name: "a", dir: "", path: "a.go", text: crlfHunk})
	m = next.(Model)
	m.shownLines = []render.Line{
		{Class: render.ClassToolResult, Text: "downloading 10%\rdownloading 99%\rdone"},
		{Class: render.ClassText, Text: "a \x1b[2J screen wipe and a bell \x07", Cont: true},
	}
	m.partials = map[string]string{}
	m.setPartial("a", "streaming\rover the pane")
	m.output.Width = m.outputWidth()
	m.output.Height = m.outputHeight()
	m.redrawBlocks()
	m.setContent()
	for i, row := range strings.Split(m.reviewSplit(), "\n") {
		for _, bad := range []struct {
			name string
			char string
		}{{"a carriage return", "\r"}, {"a bell", "\x07"}, {"a backspace", "\x08"}} {
			if strings.Contains(row, bad.char) {
				t.Fatalf("row %d holds %s: %q", i, bad.name, row)
			}
		}
	}
}
