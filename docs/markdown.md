# Markdown in the output pane

Claude Code answers in markdown. The interface renders it with
[glamour](https://github.com/charmbracelet/glamour), so a heading reads as a
heading and a code fence reads as code. The package `internal/markdown` holds
the renderer and the style.

## What is rendered, and what is not

Assistant text is treated as full markdown. That is the class `ClassText`
described in [tui.md](./tui.md). A prompt gets inline emphasis only.

| Line | Treated as markdown |
|---|---|
| What the assistant says | Yes, full markdown |
| The content of a loaded skill | Yes, in one muted grey |
| Your prompt | Inline emphasis only |
| A tool call, and its result | No |
| The init line, the turn result, and errors | No |

A tool result is a log, a diff, or a file. Markdown would fold its blank lines
and eat its asterisks, so it goes to the screen exactly as it arrived.

## A loaded skill renders in one muted grey

Claude Code loads a skill by writing its whole content into the transcript, so
the content is noise, not the answer. `markdown.NewMuted` renders it in one grey
(the colour of the meta lines), so it recedes while it keeps its structure. The
document sets the grey, the child elements drop their own colours so they
inherit it, and the code block drops its highlighter. See
[tui/output.md](./tui/output.md).

## A prompt gets inline emphasis only

A prompt keeps its identity: the heading grey and the `› ` marker. On top of
that, three inline forms render. The interface sets no bold and no italic (see
[tui/theme.md](./tui/theme.md)), so emphasis is white:

| Markup | Result |
|---|---|
| `_italic_` or `*italic*` | white, with the marks removed |
| `**bold**` or `__bold__` | white, with the marks removed |
| `` `code` `` | a code span, white on the code grey |

A prompt is not a full markdown document. A heading mark, a list mark, or a code
fence stays as plain text. So `## notes` in a prompt shows the `##`, and does not
become a heading.

An underscore in a word does not open emphasis, so `some_var_name` stays plain.
A mark that does not close, such as a lone `_`, also stays plain. The whole
prompt renders on the code path in `internal/tui/inline.go`.

## Every heading is uppercase, and nothing more

A terminal pane is narrow, and a large heading block looks wrong in it. So every
heading level, from `#` to `######`, renders the same way: uppercase text in the
heading grey, with one blank line after it. There is no background block, no coloured bar, and no `#`
marks.

So a document with headings keeps its structure, and the pane keeps one voice.

## Code is highlighted in greys

Chroma highlights a code fence with the lexer for its language. The style gives
each kind of token a grey, not a hue: keywords and function names are white,
types and numbers are the heading grey, strings and punctuation are softer, and
comments are dim. Only a diff keeps its two state colours for the inserted and
the deleted lines. The fence sits on the code grey (`#262626`).

The style is in `internal/markdown/style.go`. Glamour takes a grey as an xterm
number, so a 256-colour terminal shows it exactly. Chroma reads only hex, so the
chroma greys are written as hex, and each one is an exact xterm grey.

A character that the lexer cannot classify becomes an `Error` token. A pipe is
legal in a shell, but not in JSON, TOML, or a makefile. The style gives the
`Error` token the grey of plain code text and no background, so an unclassified
character looks like the rest of the fence.

## One set of greys

The renderer always uses the dark greys, because the interface always paints a
black ground. See [tui/theme.md](./tui/theme.md).

## The raw toggle

Press `o m` to switch the pane between rendered markdown and the exact text that
arrived. The session bar shows `raw` while the markdown is off.

Use it to copy a code fence exactly, or to see what the model really wrote when
a render looks wrong.

## Width, and the cache

Markdown is rendered at display time, and not when the event arrives, because
only the interface knows how wide the pane is. Each block is rendered once and
kept. A change of width clears the cache and rebuilds the renderer, which is why
a resize re-renders the pane.

The cache holds 512 blocks. Past that it is emptied, because a long session
otherwise grows without a bound.

An empty block, a width of zero, or a render that fails all give back the
original text. A bad render therefore costs the styling, and never the words.

## What it costs

Glamour brings goldmark and chroma with it: four direct dependencies, about
thirty modules, and a binary that grows from 3.4 MB to about 14 MB. Most of that
is the syntax highlighter's language list. This is a local tool, so the size
costs disk and nothing else.
