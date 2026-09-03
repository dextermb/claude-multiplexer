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
code fence all read as themselves. Press `m` for the raw text. See
[markdown.md](../markdown.md).

Your prompt appears the moment you send it. The interface holds a copy and
shows it at once, so there is no wait for the round trip through Claude Code.
Claude Code then echoes the prompt back through the stream. See
[protocol.md](../protocol.md). When the echo lands, the interface drops the held
copy, so the prompt is never shown twice. The echoed prompt is in the transcript
for a later replay, and the pane reads as a conversation.

The class travels with the line from the renderer, through the manager buffer,
to the screen. The one-shot `run` command prints the same lines with no colour,
because it usually writes to a pipe.

## A large result is collapsed, and opens on demand

A tool result of more than one line does not fill the pane. The renderer shows a
short summary, such as `← 4213 lines`, and keeps the whole body on the line in a
`Full` field. A summary line carries a `⏎` mark, so you can see that it opens.

The body travels with the line, so the pane holds it without a second read of
the transcript. Only a collapsed line carries a body, and the buffer caps the
line count, so the extra memory is bounded.

Press `Enter` in the output pane to open a result. A dialog lists every result
that carries a body, newest last. Choose one to page it in a scrollable view.
See [keys.md](keys.md) for the keys.

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
