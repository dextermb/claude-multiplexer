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
| Your prompt | Inline emphasis only |
| A tool call, and its result | No |
| The init line, the turn result, and errors | No |

A tool result is a log, a diff, or a file. Markdown would fold its blank lines
and eat its asterisks, so it goes to the screen exactly as it arrived.

## A prompt gets inline emphasis only

A prompt keeps its identity: the violet colour, the bold weight, and the `› `
marker. On top of that, three inline forms render:

| Markup | Result |
|---|---|
| `_italic_` or `*italic*` | italic |
| `**bold**` or `__bold__` | bold |
| `` `code` `` | a code span |

A prompt is not a full markdown document. A heading mark, a list mark, or a code
fence stays as plain text. So `## notes` in a prompt shows the `##`, and does not
become a heading.

An underscore in a word does not open emphasis, so `some_var_name` stays plain.
A mark that does not close, such as a lone `_`, also stays plain. The whole
prompt renders on the code path in `internal/tui/inline.go`.

## Every heading is bold, and nothing more

A terminal pane is narrow, and a large heading block looks wrong in it. So every
heading level, from `#` to `######`, renders the same way: bold text, with one
blank line after it. There is no background block, no coloured bar, and no `#`
marks.

So a document with headings keeps its structure, and the pane keeps one voice.

## The raw toggle

Press `m` to switch the pane between rendered markdown and the exact text that
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
