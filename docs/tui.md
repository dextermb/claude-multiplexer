# The terminal user interface

`multiplexier` with no command starts the interface. It is a Bubble Tea program
in `internal/tui`. It talks only to the manager, which is described in
[manager.md](./manager.md).

## The layout

```
┌──────────────────────────┬──────────────────────────────────────────────┐
│ SESSIONS                 │ api · claude-opus-4-8 · auto                 │
│ ▸ api               busy │        busy · 3 turns · 1.4s · 11.6k in · $… │
│   docs           idle +2 ├──────────────────────────────────────────────┤
│   worker          failed │ › write the summary                          │
│   invoices       stored  │ ● 2127c615 · claude-opus-4-8 · 31 tools      │
│   landing        stored  │ → Bash echo hello                            │
│                          │ ← hello                                      │
│                          │ ✓ success · 1.4s · $0.0669 · 1 turn          │
├──────────────────────────┴──────────────────────────────────────────────┤
│ api ⌁                                                                   │
│ > write the summary                                                     │
├─────────────────────────────────────────────────────────────────────────┤
│ 3 sessions · 1 busy · $0.0881 · n new · x stop · tab focus · q quit     │
└─────────────────────────────────────────────────────────────────────────┘
```

The sidebar is 26 columns. Each row shows the name, the state, and `+n` when
prompts wait in the queue. The colour follows the state: blue for `starting`,
green for `idle`, orange for `busy`, red for `failed`, and grey for `exited`.

## The pages

| Page | Read it for |
|---|---|
| [tui/sessions.md](tui/sessions.md) | The sidebar: live rows, stored rows, archived rows, and the two bars |
| [tui/keys.md](tui/keys.md) | Every key, the searchable key list, scrolling, the mouse, and quitting |
| [tui/input.md](tui/input.md) | The prompt box, dropping a file, and the new session form |
| [tui/output.md](tui/output.md) | The colour of each line, streaming text, and the layout rule |

Two things live outside this folder, because they are not only about the
screen. Preset prompts are in [templates.md](./templates.md), and what is
rendered as markdown is in [markdown.md](./markdown.md).
