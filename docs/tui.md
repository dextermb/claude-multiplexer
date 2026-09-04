# The terminal user interface

`multiplexier` with no command starts the interface. It is a Bubble Tea program
in `internal/tui`. It talks only to the manager, which is described in
[manager.md](./manager.md).

## The layout

```
┌──────────────────────────┬──────────────────────────────────────────────┐
│ ● api                    │▎api · claude-opus-4-8 · auto                 │
│ ⠹ docs               ⇢2  │        busy · 11.6k in 0.6k out · $0.0669    │
│ ● worker                 ├──────────────────────────────────────────────┤
│ ○ invoices               │ › write the summary                          │
│ · landing                │ ● 2127c615 · claude-opus-4-8 · 31 tools      │
│                          │ → Bash echo hello                            │
│                          │ ← hello                                      │
│                          │ ✓ success · $0.0669                          │
├──────────────────────────┴──────────────────────────────────────────────┤
│ api                                                                     │
│ > write the summary                                                     │
├─────────────────────────────────────────────────────────────────────────┤
│ 3 sessions · 1 busy · $0.0881 · 1 stored   n new · t preset · ? keys    │
└─────────────────────────────────────────────────────────────────────────┘
```

The sidebar is 26 columns. Each row shows a state glyph, the name, and `⇢n`
when prompts wait in the queue. The selected row has a highlighted background,
and the focused pane carries a blue left edge. The palette is the Tailwind gray
and blue scale. See [tui/sessions.md](tui/sessions.md) for the glyph legend.

When the selected session keeps a task list, a panel on the right of the pane
shows it, and the output shrinks to make room. See [tui/tasks.md](tui/tasks.md).

## The pages

| Page | Read it for |
|---|---|
| [tui/sessions.md](tui/sessions.md) | The sidebar: live rows, stored rows, archived rows, and the two bars |
| [tui/keys.md](tui/keys.md) | Every key, the searchable key list, scrolling, the mouse, and quitting |
| [tui/input.md](tui/input.md) | The prompt box, dropping a file, and the new session form |
| [tui/output.md](tui/output.md) | The colour of each line, streaming text, and the layout rule |
| [tui/tasks.md](tui/tasks.md) | The task panel: the list a session keeps, its glyphs, and when it shows |

Two things live outside this folder, because they are not only about the
screen. Preset prompts are in [templates.md](./templates.md), and what is
rendered as markdown is in [markdown.md](./markdown.md).
