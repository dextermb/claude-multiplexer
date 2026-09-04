# multiplexier

`multiplexier` supervises many Claude Code sessions at the same time. Each
session is a `claude` child process in headless mode, with its own directory and
its own conversation. One terminal interface shows them all: a list of sessions,
the output of the selected one, and a prompt box.

```
▎▾ ⇄ boss               3│  api · claude-opus-4-8 · auto     busy · ⇢2 · 4.2k in 0.3k out · $0.0212
▎ ● boss                 │ › write the summary
▎ ⠋ api                ⇢2│ ● 2127c615 · claude-opus-4-8 · 31 tools
▎ ○ invoices             │ → Bash echo hello
▎▾ multiplexer          1│ ← hello
▎ ● docs                 │ ← 4213 lines  ⏎
▎▸ notes              ○ 1│ The loader has three problems▌
▎                        │
▎                        │
 api — press Enter or Tab to type
 > Type a prompt, then press Enter
 3 sessions · 1 busy · $0.0881   n new · t preset · r resume · a archive · x stop · ? keys · q quit
```

The list groups the sessions: one group for each repository, and one group for
the work of each control session. A glyph gives the state of each row, and the
blue left edge marks the pane that has the focus. See
[docs/tui.md](docs/tui.md).

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
just install-as cmux  # build it into ~/.local/bin under the name cmux
```

Or run the binary directly:

```sh
multiplexier                            # the interface
multiplexier --dir ~/code --model …     # the interface, with one session open
multiplexier run --dir . "your prompt"  # one session, one prompt, plain output
multiplexier templates                  # the preset prompts, and their fields
```

`multiplexier run` exists to drive the engine from a script or a pipe. It prints
plain text, and it never renders markdown.

## What it does

- **Many sessions at once.** Each one has a directory, a model, a permission
  mode, and an effort level. Prompts queue for each session, so you never wait
  to type.
- **Sessions in groups.** A row sits under its repository, or under the control
  session that created it. Press `l f` to fold a group, and `l F` to fold them
  all.
- **Streaming.** The answer appears as it is written, marked with `▌`, and
  settles into rendered markdown when it is complete.
- **Markdown.** Headings, lists, quotes, tables, and code fences with syntax
  colours. Press `o m` for the raw text.
- **Large results on demand.** A long tool result collapses to a summary marked
  `⏎`. Press `Enter` in the output pane to page the whole body.
- **Questions you answer.** When a session asks a question, the row turns to
  `?`, and the pane shows the options. Choose one, or type your own answer.
- **Work that survives.** A session with at least one turn is remembered. It
  comes back in the list as `stored`, shows its history, and `Enter` resumes the
  same conversation. Press `s a` to archive one, `l a` to see archived rows, and
  `s n` to give a row a title.
- **Settings you change while it runs.** `s m` sets the model, `s e` sets the
  effort, and `s p` sets the permission mode of a running session.
- **Two keys for every action.** The first key names the target, and the second
  names the action: `s j` shows the jobs of the session, and `s m` sets its
  model. Press `?` for a searchable list of the key bindings.
- **Paths that complete.** The new session form suggests directories as you
  type, and `Tab` grows the path. In the prompt box, an `@` word completes to a
  path in the same way.
- **Preset prompts.** Keep a prompt with holes in it, such as `Look up Linear
  issue {{issue}}`. Press `t` to pick one and fill it in, or type
  `/linear ENG-123`. A new session can start with one.
- **Files by drag.** Drop a file on the window and its path goes into the
  prompt, unescaped and quoted where needed.
- **Sessions that talk.** A session can name itself, list its neighbours, and
  read their recent messages. Give it the control grant and it can also prompt,
  stop, archive, and create them. See [docs/mcp.md](docs/mcp.md).
- **Jobs and tasks in view.** A panel on the right of the pane lists the
  background shell commands of the session and its task list. Press `s j` for the
  jobs of the selected session.
- **Costs in view.** The session bar shows the context, the tokens, and the
  cost. The bottom bar totals the cost of every session.
- **The mouse.** Click a row to select it, click a header to fold it, and roll
  the wheel to scroll. Drag over the output to copy the text.

## Where things live

State goes under `~/.multiplexier/sessions/<name>/`:

| File | What it holds |
|---|---|
| `transcript.jsonl` | Every event of the conversation, as JSON Lines |
| `meta.json` | The directory, the model, the effort, the title, the Claude session id, the totals, the creator, the control grant, and the archive flag |
| `mcp.json` | Where the session reaches the multiplexer's own tools, and the token that names it |

`multiplexier --root <path>` moves that directory. `just clean-state` removes it.

## Documentation

| Page | Read it for |
|---|---|
| [docs/tui.md](docs/tui.md) | The interface: the layout, and a page for each part of it |
| [docs/templates.md](docs/templates.md) | Writing a preset prompt, its fields, and the three ways in |
| [docs/markdown.md](docs/markdown.md) | What is rendered, the heading rule, and the raw toggle |
| [docs/manager.md](docs/manager.md) | Sessions in memory, the event bus, storage, and archiving |
| [docs/sessions.md](docs/sessions.md) | One child process: its states, its queue, and its transcript |
| [docs/protocol.md](docs/protocol.md) | The stream-json wire format, and what Claude Code really sends |
| [docs/mcp.md](docs/mcp.md) | The tools a session can call: renaming itself, and driving its neighbours |

`docs/plans/` holds the thinking that came before the code. It records what is
still ahead. It is not a specification, so do not follow it.

## Working on it

```sh
just              # list every recipe
just test         # the tests
just check        # gofmt, vet, and three race runs
just cover        # a coverage report
```

Each piece of work goes in its own git worktree: `just worktree <name>` starts
one, and `just collapse <name>` merges it back and removes it.

The tests never reach the network. `internal/testutil/fakeclaude` is a small
program that answers like Claude Code, and the tests point the session at it.
`FAKECLAUDE_MODE` selects how it behaves, which is how a crash, a late `init`,
and a line that is not JSON are all covered.

Three recipes do use the real binary, and they cost money:

```sh
just test-real    # one real session through the interface, about a penny
just probe-mcp    # one real session that calls the mux tools
just smoke        # one prompt, one word back
```

## Licence

`multiplexier` is free software under the GNU Affero General Public License,
version 3. See [LICENSE](LICENSE).

You may use it, change it, and share it. Two rules come with that freedom. If
you share a changed version, you must offer its source under the same licence.
If you run a changed version as a service that other people reach over a
network, you must offer them its source as well.

The licence covers `multiplexier` itself. It does not cover the work the
sessions do, or the code they write.

Copyright © 2026 Dexter Marks-Barber.
