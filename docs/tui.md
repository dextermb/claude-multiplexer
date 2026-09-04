# The terminal user interface

`multiplexier` with no command starts the interface. It is a Bubble Tea program
in `internal/tui`. It talks only to the manager, which is described in
[manager.md](./manager.md).

## The layout

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

The sidebar is 26 columns. The sessions are grouped under a header that names
the group and counts its rows: one group for each repository, and one for the
work of each control session. Each row shows a state glyph, the display name,
and `⇢n` when prompts wait in the queue. The selected row has a
highlighted background, and the focused pane carries a blue left edge. The
palette is the Tailwind gray and blue scale. See
[tui/sessions.md](tui/sessions.md) for the groups, the folds, and the glyph
legend.

When the selected session has background jobs or a task list, a panel on the
right of the pane shows them, and the output shrinks to make room. See
[tui/tasks.md](tui/tasks.md).

## The pages

| Page | Read it for |
|---|---|
| [tui/sessions.md](tui/sessions.md) | The sidebar: live rows, stored rows, archived rows, and the two bars |
| [tui/keys.md](tui/keys.md) | Every key, the searchable key list, scrolling, the mouse, and quitting |
| [tui/input.md](tui/input.md) | The prompt box, dropping a file, and the new session form |
| [tui/output.md](tui/output.md) | The colour of each line, streaming text, and the layout rule |
| [tui/tasks.md](tui/tasks.md) | The side panel: the session's jobs and task list, their glyphs, and when it shows |

Two things live outside this folder, because they are not only about the
screen. Preset prompts are in [templates.md](./templates.md), and what is
rendered as markdown is in [markdown.md](./markdown.md).
