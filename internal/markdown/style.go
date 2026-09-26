package markdown

import (
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
)

// MutedGrey is the one grey NewMuted draws every element in, so a skill dump
// recedes and matches its launching line. See docs/markdown.md.
const MutedGrey = "245"

// A tone is the Blackline greys: xterm-256 numbers for glamour, and hex for
// chroma, which reads only hex. See docs/markdown.md.
type tone struct {
	body, heading, strong, muted, dimmed, rule, code string
	chroma                                           chromaTone
}

type chromaTone struct {
	text, bright, heading, muted, dimmed, str, inserted, deleted, background string
}

var darkTone = tone{
	body: "251", heading: "255", strong: "231", muted: "245", dimmed: "242", rule: "237", code: "235",
	chroma: chromaTone{
		text: "#c6c6c6", bright: "#ffffff", heading: "#eeeeee", muted: "#8a8a8a", dimmed: "#6c6c6c",
		str: "#a8a8a8", inserted: "#4ade80", deleted: "#ff6666", background: "#262626",
	},
}

func paneStyle(t tone) ansi.StyleConfig {
	style := styles.DarkStyleConfig
	none := uint(0)
	no := false
	style.Document.Margin = &none
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	style.Document.Color = &t.body

	heading := headingStyle(t)
	style.Heading = heading
	style.H1 = heading
	style.H2 = heading
	style.H3 = heading
	style.H4 = heading
	style.H5 = heading
	style.H6 = heading

	style.Strong = ansi.StylePrimitive{Color: &t.strong}
	style.Emph = ansi.StylePrimitive{Color: &t.strong, Italic: &no}
	style.BlockQuote.Color = &t.muted
	quote := "│ "
	style.BlockQuote.IndentToken = &quote
	style.HorizontalRule = ansi.StylePrimitive{Color: &t.rule, Format: "\n────────\n"}
	style.Item = ansi.StylePrimitive{BlockPrefix: "· "}
	style.Enumeration = ansi.StylePrimitive{BlockPrefix: ". ", Color: &t.dimmed}
	style.Link = ansi.StylePrimitive{Color: &t.muted, Underline: boolPtr(true)}
	style.LinkText = ansi.StylePrimitive{Color: &t.strong}
	style.Image = ansi.StylePrimitive{Color: &t.muted, Underline: boolPtr(true)}
	style.ImageText = ansi.StylePrimitive{Color: &t.dimmed, Format: "image: {{.text}} ↗"}
	style.Code = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: " ", Suffix: " ", Color: &t.strong, BackgroundColor: &t.code,
	}}
	style.CodeBlock.Margin = &none
	style.CodeBlock.Color = &t.body
	style.CodeBlock.Chroma = greyChroma(t.chroma)
	style.Table.Color = &t.body
	return style
}

// mutedStyle draws every element in one grey, so a skill dump recedes. The
// document sets the grey, the child elements drop their own colours so they
// inherit it, and the code block drops its highlighter. See docs/markdown.md.
func mutedStyle(t tone) ansi.StyleConfig {
	grey := MutedGrey
	style := paneStyle(t)
	style.Document.Color = &grey
	style.Text.Color = &grey
	for _, colour := range []**string{
		&style.Paragraph.Color, &style.BlockQuote.Color, &style.Emph.Color,
		&style.Strong.Color, &style.Item.Color, &style.Enumeration.Color,
		&style.Link.Color, &style.LinkText.Color, &style.Image.Color,
		&style.ImageText.Color, &style.Code.Color, &style.Code.BackgroundColor,
		&style.Heading.Color, &style.H1.Color, &style.H2.Color, &style.H3.Color,
		&style.H4.Color, &style.H5.Color, &style.H6.Color, &style.Table.Color,
	} {
		*colour = nil
	}
	style.CodeBlock.Chroma = nil
	style.CodeBlock.Color = &grey
	return style
}

// headingStyle sets every level the same way: uppercase in the heading grey,
// with one blank line after it. See docs/markdown.md.
func headingStyle(t tone) ansi.StyleBlock {
	return ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       &t.heading,
			Upper:       boolPtr(true),
		},
	}
}

// greyChroma highlights code by weight of grey, not by hue: keywords and names
// bright, strings and punctuation softer, comments dim. Only a diff keeps its
// two state colours. An unclassified character takes the plain text grey, with
// no background. See docs/markdown.md.
func greyChroma(c chromaTone) *ansi.Chroma {
	p := func(hex string) ansi.StylePrimitive { return ansi.StylePrimitive{Color: &hex} }
	return &ansi.Chroma{
		Text:                p(c.text),
		Error:               p(c.text),
		Comment:             p(c.dimmed),
		CommentPreproc:      p(c.muted),
		Keyword:             p(c.bright),
		KeywordReserved:     p(c.bright),
		KeywordNamespace:    p(c.bright),
		KeywordType:         p(c.heading),
		Operator:            p(c.muted),
		Punctuation:         p(c.muted),
		Name:                p(c.text),
		NameBuiltin:         p(c.heading),
		NameTag:             p(c.heading),
		NameAttribute:       p(c.text),
		NameClass:           p(c.heading),
		NameConstant:        p(c.heading),
		NameDecorator:       p(c.muted),
		NameException:       p(c.heading),
		NameFunction:        p(c.bright),
		NameOther:           p(c.text),
		Literal:             p(c.text),
		LiteralNumber:       p(c.heading),
		LiteralDate:         p(c.text),
		LiteralString:       p(c.str),
		LiteralStringEscape: p(c.heading),
		GenericDeleted:      p(c.deleted),
		GenericEmph:         p(c.text),
		GenericInserted:     p(c.inserted),
		GenericStrong:       p(c.bright),
		GenericSubheading:   p(c.muted),
		Background:          ansi.StylePrimitive{BackgroundColor: &c.background},
	}
}

func boolPtr(b bool) *bool { return &b }
