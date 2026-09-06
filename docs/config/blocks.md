# The block cap

The session pane draws at most so many rows of one block, and then a marker
row that opens the rest in place. A block is one piece of content: your prompt,
one message, one tool result, or the output of a `!` command. See
[../tui/output.md](../tui/output.md). For where the settings file lives, see
[../config.md](../config.md).

## The default cap, for every type

The `blockCap` default caps every block the same way. Three sources name it, and
the first one that names one wins:

| Order | Source | Example |
|---|---|---|
| 1 | `--block-cap` | `multiplexer --block-cap 40` |
| 2 | `blockCap` in the settings file | `{"blockCap": 40}` |
| 3 | The built-in default | 20 rows |

A default of `0` caps nothing, so every block draws in full. A default below
zero is an error, from the flag and from the tool.

## A cap for one type

`blockCaps` sets a separate cap for one type of block. A block takes the bucket
of its first line:

| Bucket | The content it holds |
|---|---|
| `prompt` | Your prompt. |
| `message` | The text and the thinking of the assistant. |
| `tool` | A tool call, and its result. |
| `meta` | The session line, the token line, and the result line. |
| `bash` | The output of a `!` command. |
| `error` | An error, and a line from stderr. |

Each bucket takes one of three states:

- a number `N` draws `N` rows, then the marker. `0` draws only the marker, so
  you open the block to read any of it.
- `null` never caps the bucket, so the block draws in full.
- no entry takes the `blockCap` default.

For example, this file collapses the token and result lines to a marker, and
never caps a message:

```json
{
  "blockCap": 20,
  "blockCaps": {
    "meta": 0,
    "message": null
  }
}
```

The `--block-cap-type name=rows` flag sets one bucket for one run (`rows` of `-1`
means `null`). The flag is repeatable, and it wins over the file.

## The question modal caps

Two more buckets cap the question modal, not the pane. A long option label or a
long option description wraps across lines, and these buckets cap the lines it
draws before a marker:

| Bucket | The content it holds |
|---|---|
| `question_option` | One option label in the question modal. |
| `question_description` | One option description in the question modal. |

They take the same three states as the pane buckets, but an absent bucket takes
a default of `2` lines, not the `blockCap` default. So the global `blockCap` does
not reach them. The option under the cursor draws in full, so you read the rest
by moving to it. See [../tui/input.md](../tui/input.md).

## A session writes the cap

A session can write the cap itself, with the `set_block_cap` tool, and take it
out again with `unset_block_cap`. Both tools take an optional `type`, so a
session sets or clears one bucket, or the default. The interface reads the file
again at each notice, so a new cap reaches the pane at once, and the pane draws
itself again. See [../mcp/tools.md](../mcp/tools.md).
