# The terminal user interface

`multiplexer` with no command starts the interface. It is a Bubble Tea program
in `internal/tui`. It talks only to the manager, which is described in
[manager.md](./manager.md).

## The layout

```
▎▾ ⇄ boss               3│  api · claude-opus-4-8 · auto     busy · ⇢2 · 4.2k in 0.3k out · $0.0212
▎ ● boss                 │ › write the summary
▎ ⠋ api                ⇢2│ ● 2127c615 · claude-opus-4-8 · 31 tools
▎ ○ invoices             │ → Bash echo hello
▎▾ multiplexer          1│ ← hello
▎ ● docs                 │ ▸ ⋯ 4193 more lines
▎▸ notes              ○ 1│ The loader has three problems▌
▎                        │
▎                        │
 api — press Enter or Tab to type
 > Type a prompt, then press Enter
 3 sessions · 1 busy · $0.0881  n new · t preset · s session · l list · o output · ? keys · q quit
```

The sidebar is 26 columns by default, and a layout can change its width, the
task panel width, the diff panel width, and the prompt bar height. See
[tui/layouts.md](tui/layouts.md). The sessions are grouped under a header that names
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

The session bar shows the git diff count of the session, for example `+120 −30`.
`s d` opens a diff panel that lists the changed files and expands each one to its
diff. See [tui/diff.md](tui/diff.md).

## Where a dialog draws

A dialog draws in one of two regions.

A **session dialog** names one session, so it draws in the pane, under the
session bar, in place of the output. The sidebar, the session bar, the prompt
and the status bar stay on the screen. The jobs list, the model,
effort and mode dialogs, the rename dialog, and the stop confirmation draw here,
and each one covers the side panel as well. The question dialog also
draws here, but it keeps the side panel beside it. See
[tui/input.md](tui/input.md).

```
 sidebar  │ bar                                      │
 26 cols  ├──────────────────────────────────────────┤
          │                                          │
          │           a session dialog               │
          │                                          │
──────────┴──────────────────────────────────────────┤
 prompt                                              │
 status                                              │
```

A **body dialog** names no session, so it covers the sidebar and the pane
together. The new session form, the preset picker, the preset field form, and
the key list draw here.

A dialog is at most two columns narrower than its region, so a narrow terminal
never pushes the sidebar out of line.

## The pages

| Page | Read it for |
|---|---|
| [tui/sessions.md](tui/sessions.md) | The sidebar: live rows, stored rows, archived rows, and the two bars |
| [tui/keys.md](tui/keys.md) | The key sequences, every single key, the searchable key list, scrolling, the mouse, and quitting |
| [tui/input.md](tui/input.md) | The prompt box, dropping a file, and the new session form |
| [tui/output.md](tui/output.md) | The colour of each line, streaming text, and the layout rule |
| [tui/tasks.md](tui/tasks.md) | The side panel: the session's jobs and task list, their glyphs, and when it shows |
| [tui/diff.md](tui/diff.md) | The git diff: the count in the bar, the file panel, the inline diff, and the refresh |
| [tui/layouts.md](tui/layouts.md) | The layouts: the four dimensions, the precedence, the switcher, and where a layout lives |

Three things live outside this folder, because they are not only about the
screen. Preset prompts are in [templates.md](./templates.md), what is rendered
as markdown is in [markdown.md](./markdown.md), and the settings file that
names your editor is in [config.md](./config.md).
