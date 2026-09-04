package markdown

import (
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

const maxCache = 512

type Renderer struct {
	mu    sync.Mutex
	width int
	term  *glamour.TermRenderer
	cache map[string]string
}

func New() *Renderer {
	return &Renderer{cache: make(map[string]string)}
}

func (r *Renderer) Render(text string, width int) string {
	if strings.TrimSpace(text) == "" || width <= 0 {
		return text
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if width != r.width || r.term == nil {
		term, err := glamour.NewTermRenderer(
			glamour.WithStyles(paneStyle()),
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

func paneStyle() ansi.StyleConfig {
	style := styles.DarkStyleConfig
	none := uint(0)
	style.Document.Margin = &none
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	style.CodeBlock.Margin = &none
	style.CodeBlock.Chroma = plainErrors(style.CodeBlock.Chroma)

	heading := headingStyle()
	style.Heading = heading
	style.H1 = heading
	style.H2 = heading
	style.H3 = heading
	style.H4 = heading
	style.H5 = heading
	style.H6 = heading
	return style
}

// plainErrors copies the chroma style and gives an unclassified character the
// colour of plain code text. See docs/markdown.md.
func plainErrors(chroma *ansi.Chroma) *ansi.Chroma {
	if chroma == nil {
		return nil
	}
	next := *chroma
	next.Error = next.Text
	return &next
}

func headingStyle() ansi.StyleBlock {
	bold := true
	return ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Bold:        &bold,
		},
	}
}
