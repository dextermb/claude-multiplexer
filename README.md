# multiplexier

`multiplexier` supervises many Claude Code sessions at the same time. Each
session is a `claude` child process in headless mode, with its own directory and
its own conversation. One terminal interface shows them all: a sidebar of
sessions, the output of the selected one, and a prompt box.

```
┌──────────────────────────┬─────────────────────────────────────────────┐
│ SESSIONS                 │ api · claude-opus-4-8 · auto                │
│ ▸ api               busy │       busy · 3 turns · 1.4s · 11.6k in · $… │
│   docs           idle +2 ├─────────────────────────────────────────────┤
│   worker          failed │ › write the summary                         │
│   invoices       stored  │ → Bash echo hello                           │
│                          │ ← hello                                     │
│                          │ The loader has three problems▌              │
├──────────────────────────┴─────────────────────────────────────────────┤
│ api ⌁                                                                  │
│ > _                                                                    │
│ 3 sessions · 1 busy · $0.0881 · n new · x stop · tab focus · q quit    │
└────────────────────────────────────────────────────────────────────────┘
```

## What you need

- Go 1.25 or later.
- Claude Code 2.1.176 or later, on the path as `claude`.
- A terminal that supports 256 colours.

## Start it

```sh
just build            # build bin/multiplexier
just tui              # start the interface
just tui-in ~/code    # start it, and open one session in ~/code
just tui-fake         # start it against a fake binary: no network, no cost
```

Or run the binary directly:

```sh
multiplexier                            # the interface
multiplexier --dir ~/code --model …     # the interface, with one session open
multiplexier run --dir . "your prompt"  # one session, one prompt, plain output
```

`multiplexier run` exists to drive the engine from a script or a pipe. It prints
plain text, and it never renders markdown.

## What it does

- **Many sessions at once.** Each one has a directory, a model, and a permission
  mode. Prompts queue for each session, so you never wait to type.
- **Streaming.** The answer appears as it is written, marked with `▌`, and
  settles into rendered markdown when it is complete.
- **Markdown.** Headings, lists, quotes, tables, and code fences with syntax
  colours. Press `m` for the raw text.
- **Work that survives.** A session with at least one turn is remembered. It
  comes back in the sidebar as `stored`, shows its history, and `Enter` resumes
  the same conversation. Press `a` to archive one, and `A` to see archived rows.
- **Every key in one place.** Press `?` for a searchable list of the key
  bindings.
- **Paths that complete.** The new session form suggests directories as you
  type, and `Tab` grows the path.
- **Preset prompts.** Keep a prompt with holes in it, such as `Look up Linear
  issue {{issue}}`. Press `t` to pick one and fill it in, or type
  `/linear ENG-123`. A new session can start with one.
- **Files by drag.** Drop a file on the window and its path goes into the
  prompt, unescaped and quoted where needed.
- **Costs in view.** Each session shows its turns, its last duration, its
  tokens, and its cost. The bottom bar totals them.

## Where things live

State goes under `~/.multiplexier/sessions/<name>/`:

| File | What it holds |
|---|---|
| `transcript.jsonl` | Every event of the conversation, as JSON Lines |
| `meta.json` | The directory, the model, the Claude session id, the totals, and the archive flag |

`multiplexier --root <path>` moves that directory. `just clean-state` removes it.

## Documentation

| Page | Read it for |
|---|---|
| [docs/tui.md](docs/tui.md) | The panes, every key, the mouse, and dropping files |
| [docs/templates.md](docs/templates.md) | Writing a preset prompt, its fields, and the three ways in |
| [docs/markdown.md](docs/markdown.md) | What is rendered, the heading rule, and the raw toggle |
| [docs/manager.md](docs/manager.md) | Sessions in memory, the event bus, storage, and archiving |
| [docs/sessions.md](docs/sessions.md) | One child process: its states, its queue, and its transcript |
| [docs/protocol.md](docs/protocol.md) | The stream-json wire format, and what Claude Code really sends |

`docs/plans/` holds the thinking that came before the code. It records what is
still ahead. It is not a specification, so do not follow it.

## Working on it

```sh
just              # list every recipe
just test         # the tests
just check        # gofmt, vet, and three race runs
just cover        # a coverage report
```

The tests never reach the network. `internal/testutil/fakeclaude` is a small
program that answers like Claude Code, and the tests point the session at it.
`FAKECLAUDE_MODE` selects how it behaves, which is how a crash, a late `init`,
and a line that is not JSON are all covered.

One test does use the real binary, and it costs money:

```sh
just test-real    # one real session, about a penny
```
