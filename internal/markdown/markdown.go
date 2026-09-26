package markdown

import (
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

const maxCache = 512

type Renderer struct {
	style ansi.StyleConfig
	mu    sync.Mutex
	width int
	term  *glamour.TermRenderer
	cache map[string]string
}

func New() *Renderer {
	return &Renderer{style: paneStyle(darkTone), cache: make(map[string]string)}
}

// NewMuted renders markdown in one muted grey, for content that recedes, such
// as a loaded skill. See docs/markdown.md.
func NewMuted() *Renderer {
	return &Renderer{style: mutedStyle(darkTone), cache: make(map[string]string)}
}

func (r *Renderer) Render(text string, width int) string {
	if strings.TrimSpace(text) == "" || width <= 0 {
		return text
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if width != r.width || r.term == nil {
		term, err := glamour.NewTermRenderer(
			glamour.WithStyles(r.style),
			glamour.WithWordWrap(width),
			glamour.WithPreservedNewLines(),
		)
		if err != nil {
			return text
		}
		r.term = term
		r.width = width
		r.cache = make(map[string]string)
	}

	if done, seen := r.cache[text]; seen {
		return done
	}
	out, err := r.term.Render(text)
	if err != nil {
		return text
	}
	out = trimBlankLines(out)
	if out == "" {
		return text
	}
	if len(r.cache) >= maxCache {
		r.cache = make(map[string]string)
	}
	r.cache[text] = out
	return out
}

var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func trimBlankLines(text string) string {
	lines := strings.Split(text, "\n")
	blank := func(line string) bool {
		return strings.TrimSpace(ansiCodes.ReplaceAllString(line, "")) == ""
	}
	for len(lines) > 0 && blank(lines[0]) {
		lines = lines[1:]
	}
	for len(lines) > 0 && blank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
