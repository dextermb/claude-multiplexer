# The output pane

What each line looks like, how text arrives before it settles, and why the
pane can be trusted.

## The output pane

The pane is a viewport over the rendered lines of the selected session. It
follows the newest line, unless you scroll up. Then it holds your position until
you return to the bottom.

The renderer gives each line a class, and the interface gives each class a
colour. So the words the model writes stay bright, and everything around them
recedes:

| Class | What it is | Colour |
|---|---|---|
| `ClassPrompt` | What you asked, marked `› ` | Violet, bold, with inline emphasis (see [markdown.md](../markdown.md)) |
| `ClassText` | What the assistant says | Rendered as markdown |
| `ClassToolUse` | A tool call, such as `→ Bash ls` | Blue |
| `ClassToolResult` | What the tool returned | Muted |
| `ClassMeta` | The `init` line, the turn result, and state changes | Muted, darkest |
| `ClassThinking` | Thinking, when the verbose flag is on | Muted, italic |
| `ClassStderr` | A line from the child stderr | Amber |
| `ClassError` | A failure, or a line that is not JSON | Red |

What the assistant says is rendered as markdown, so a heading, a list, and a
code fence all read as themselves. Press `o m` for the raw text. See
[markdown.md](../markdown.md).

Your prompt appears the moment you send it. The interface holds a copy and
shows it at once, so there is no wait for the round trip through Claude Code.
Claude Code then echoes the prompt back through the stream. See
[protocol.md](../protocol.md). When the echo lands, the interface drops the held
copy, so the prompt is never shown twice. The echoed prompt is in the transcript
for a later replay, and the pane reads as a conversation.

The class travels with the line from the renderer, through the manager buffer,
to the screen. The one-shot `run` command prints one line for each event, with
no colour, because it usually writes to a pipe. A line that holds a whole body
carries a `Summary` field, and that is the line the `run` command prints:
`← 4213 lines`.

## A block past the cap opens in the pane

A **block** is one piece of content: your prompt, one message from the
assistant, one tool result, or the output of a `!` command. The renderer marks
each line that continues the block the line before it started, so the pane knows
where a block ends.

A block of more than 20 rows draws its first 20 rows and a marker row under
them. 20 is the default, and `blockCap` in the settings file, `--block-cap`, and
the `set_block_cap` tool each change it. A cap of `0` caps nothing. See
[../config.md](../config.md).

```
→ Bash ./scripts/build.sh
← go: downloading github.com/charmbracelet/bubbletea v1.3.4
  go: downloading github.com/charmbracelet/lipgloss v1.1.0
  … 18 more rows of the body …
▸ ⋯ 4193 more lines · enter to open
```

The cap counts the rows the pane draws, after the text is wrapped to the width
of the pane. So one block never takes more than the cap and one marker row,
however wide its lines are.

An open block draws every row, and its marker says `⋯ show less`. The first row
of the block holds its place on the screen while it opens and closes, so the
text under your eyes does not jump.

**The cursor.** The marker row of one block carries `▸` and a highlight. That is
the block cursor, and it names the block that `Enter` opens. `]` and `[` move it
to the next capped block and to the block before, and the pane scrolls to it
only when it is out of sight. A click on any marker row opens that block.

The cursor sits on the newest capped block. It returns there at the end of every
turn, and when you select another session, because that is the block you most
often want. So `[` is for going back over the answer you just read.

**What the pane holds.** A tool result carries its whole body, because the body
travels with the line and the pane opens it without a second read of the
transcript. The buffer caps the line count, so the extra memory is bounded. See
[manager.md](../manager.md).

A change of selection closes every open block, because the buffer may have
dropped lines from the front, and a block is named by its place in the list.

## Text as it arrives

Before the first word arrives, the pane shows a spinner and the word
`thinking…` below your prompt. It marks the gap between the send and the first
token, so the session never looks stuck. The spinner shows while the session is
busy and no text streams yet. The first token replaces it.

The pane shows the answer while the model writes it. The unfinished text sits
below the settled lines and ends with a `▌`, so you can tell what is still
moving. It grows in place, and it is never repeated.

Unfinished text is shown plain, and not as markdown, because a half-written
code fence has no end yet. When the message lands, the `▌` goes and the whole
block is rendered properly. See [markdown.md](../markdown.md).

The pane follows the growing text only while you sit at the bottom, so a stream
never drags you away from something you scrolled back to read.

## How the output stays correct

The interface subscribes to the manager bus and appends the lines of each event
for the selected session. Two rules keep the pane honest:

- A change of selection rebuilds the pane from `Lines(name)`.
- A gap in the sequence number means the bus dropped an event, so the pane is
  rebuilt as well, and not appended to.

See [manager.md](../manager.md) for the drop policy that makes the gap possible.

## The view fills the window exactly

The interface draws every line itself, so the view must be exactly as tall as
the window and no line wider than it. A test asserts both, on every render.

This is not fussiness. The rule caught the layout one line too tall and one
column too narrow, and it caught the session bar quietly losing the cost: the
bar had grown past the width, wrapped, and the one-line cap then cut the tail
off. A pane that silently drops what it cannot fit is worse than one that looks
wrong, because you cannot see what is missing.

So anything drawn into a fixed space sheds detail on purpose. The session bar
has an order for what to drop first, and the sidebar cuts a name short only
after the rest has gone.
