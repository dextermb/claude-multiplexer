# The terminal user interface

`cmux` with no command starts the interface. It is a Bubble Tea program
in `internal/tui`. It talks only to the manager, which is described in
[manager.md](./manager.md).

## The layout

```
 MULTIPLEXER     WORKSPACE   REVIEW   KEYS                                    ■ 1 waiting
── SESSIONS (5) ─────────────┬── API ─ claude-opus-4-8 · auto ──── ■ busy 0:42 · queued (2) · $0.0212 ─
 ▾ C BOSS                   3│ › write the summary
  ■ boss               idle  │ ● 2127c615 · claude-opus-4-8 · 31 tools
  ■ api          q2 busy 0:42│ → Bash echo hello
  · invoices         stored  │ ← hello
 ▾ MULTIPLEXER              1│ [+] 4193 more lines
  ■ docs            waiting  │ The loader has three problems▌
 ▸ NOTES                  ■ 1│
─────────────────────────────┴───────────────────────────────────────────────────────────────────────
 api — press Enter or Tab to type
 > Type a prompt, then press Enter
 3 sessions · 1 busy · $0.0881           n new  t preset  s session  l list  o output  ? keys  q quit
```

The screen is one frame of rules. The top row is the band: the product name,
the screens (`WORKSPACE`, `REVIEW`, `KEYS`) with the current one inverted, and
on the right the live sessions that wait for an answer or failed. Below it, each
pane has its label set in its top rule: `SESSIONS (n)` for the list, the session
name for the output, and `JOBS` or `TASKS` for the side panel. The rules meet at
junctions (`┬`, `├`, `┴`), so two panes share one rule and never draw two.

The label of the pane that holds the focus inverts. There is no other focus
mark. The prompt shows its focus the same way, in its hint row. The code is in
`internal/tui/chrome.go`, and the colours are in [tui/theme.md](tui/theme.md).

The sidebar is 30 columns by default, and a layout can change its width, the
task panel width, the diff panel position and size, and the prompt bar height.
See [tui/layouts.md](tui/layouts.md). The sessions are grouped under a header that names
the group and counts its rows: one group for each repository, and one for the
work of each control session. Each row shows a state glyph, the display name,
the muted session flags (`H` hoisted, `S` scheduled, `C` control), `qn` when
prompts wait in the queue, and a state word such as `idle` or `busy 0:42`. The
selected row inverts when the list has the focus. The palette is the Blackline
grey ramp, with colour only for state. See [tui/sessions.md](tui/sessions.md) for the groups, the folds, and the
glyph legend, and [tui/theme.md](tui/theme.md) for the colours.

When the selected session has background jobs or a task list, a panel on the
right of the pane shows them, and the output shrinks to make room. See
[tui/tasks.md](tui/tasks.md).

The session bar shows the git diff count of the session, for example `+120 −30`.
`s d` opens a diff panel that lists the changed files and expands each one to its
diff. See [tui/diff.md](tui/diff.md).

The settings file composes the session bar and the status bar: it reorders the
built-in elements, removes one, and adds a custom element that runs a script.
See [config/bars.md](config/bars.md).

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
 30 cols  ├──────────────────────────────────────────┤
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

## The modal seam

Most dialogs share one seam. The model, effort and mode dialogs, the rename
dialog, the jobs list, the preset picker, the preset field form, and the layout
switcher each satisfy the `modal` interface (`internal/tui/modal.go`), and the
Model holds one `modal` field for the active dialog. The key router hands a key
to that one field, the view draws it in the region its `region()` names, and the
mouse guard blocks the wheel for a pane modal. So a new dialog is one adapter in
its own file, not a branch in the router, the view, and the mouse guard.

A message the Model does not handle goes to the active modal before the prompt.
That is how a [huh](https://charm.land) form works inside a modal: huh finishes a
form through a command whose message must come back to it.

The model, effort and mode dialogs and the rename dialog are huh forms. They are
built in `internal/tui/huhdialog.go`: `newHuhForm` gives a form the Blackline
theme and makes `esc` close it, `runForm` reads its state after each message, and
`formBox` draws it in a dialog box with a title and a hint. See
[tui/theme.md](tui/theme.md) for the theme.

Three dialogs stay outside the seam, because each has couplings beyond the
router. The question dialog is a per-session map, arrives from a manager event,
and keeps the side panel beside it. The new session form takes a dropped path
and suppresses the prompt while it is open. The stop confirmation guards the
paste and the mouse as a bare string. It draws as the danger dialog: a grey
`[ keep running ]` and a `stop session` button in the danger colour. `y` or
`enter` stops, and any other key keeps the session running.

## The update banner

A one-line banner sits below the status bar when a newer release exists. It
takes one row from the body, and `ctrl+g` dismisses it for a day. The check and
the banner are described in [version-updates.md](version-updates.md).

## The pages

| Page | Read it for |
|---|---|
| [tui/sessions.md](tui/sessions.md) | The sidebar: the sections, the groups, live and stored rows, and its pages for jobs and the bars |
| [tui/keys.md](tui/keys.md) | The key sequences, every single key, the searchable key list, scrolling, the mouse, and quitting |
| [tui/input.md](tui/input.md) | The prompt box, dropping a file, and the new session form |
| [tui/output.md](tui/output.md) | The colour of each line, streaming text, and the layout rule |
| [tui/theme.md](tui/theme.md) | The colour tokens, inversion, and why the greys are written as xterm numbers |
| [tui/tasks.md](tui/tasks.md) | The side panel: the session's jobs and task list, their glyphs, and when it shows |
| [tui/diff.md](tui/diff.md) | The git diff: the count in the bar, the file panel, the inline diff, and the refresh |
| [tui/review.md](tui/review.md) | The code review screen: the large diff, the hunk navigation, and the explanation pane |
| [tui/layouts.md](tui/layouts.md) | The layouts: the four dimensions, the precedence, the switcher, and where a layout lives |

Three things live outside this folder, because they are not only about the
screen. Preset prompts are in [templates.md](./templates.md), what is rendered
as markdown is in [markdown.md](./markdown.md), and the settings file that
names your editor is in [config.md](./config.md).
