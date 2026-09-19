# Tests without a network

`internal/testutil/fakeclaude` is a small program that answers like Claude Code.
The tests build it, and they point `ClaudePath` at it. `FAKECLAUDE_MODE` selects
the behaviour:

| Mode | Behaviour |
|---|---|
| unset | Emit `init`, then echo each prompt with an `assistant` and a `result` event. |
| `lazyinit` | Emit `init` only after the first prompt, as the real binary does. |
| `crash` | Write to stderr, and exit 1. |
| `garbage` | Write a line that is not JSON, and then work as normal. |
| `exit-after-init` | Emit `init`, and exit 0. |
| `noinit` | Exit 0 at once. |
| `bigline` | Emit a very long line. |

`FAKECLAUDE_ARGS_FILE` records the working directory and the arguments, so a
test can check what the child received.
