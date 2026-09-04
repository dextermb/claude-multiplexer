# The task panel

A session keeps a task list for itself. This panel shows that list on the right
of the session pane, so you can see what the session plans and where it is.

```
┌─────────────────────────────────┬──────────────────────────────┐
│ › add the panel                 │ Tasks · 1/3                  │
│ → TodoWrite 3 tasks             │                              │
│ ← result                        │ ✔ Add the render type        │
│ echo: add the panel             │ ◐ Wiring the manager         │
│                                 │ ○ Draw the panel             │
└─────────────────────────────────┴──────────────────────────────┘
```

## Where the list comes from

A session keeps its task list with the `TodoWrite` tool. Each time the list
changes, the session sends an assistant message with a `TodoWrite` tool_use
block. The block holds the whole list, so the newest block replaces the last
one. There is no partial update to merge. See [protocol.md](../protocol.md).

`protocol.Event.TodoWrite` reads the list from such a block. It returns the
tasks and `true`. It returns `false` for any other message. An empty list still
returns `true`, so the panel can clear when the session drops all its tasks.

Each task has three parts:

- `content` — the command form, for example "Wire the manager".
- `activeForm` — the present form, for example "Wiring the manager".
- `status` — one of `pending`, `in_progress`, or `completed`.

## What the panel shows

The panel lists every task, one to a row, with a status glyph:

| Glyph | Status | Colour |
|---|---|---|
| `○` | pending | Muted |
| `◐` | in progress | Amber, bold |
| `✔` | completed | Green |

The header counts the completed tasks against the total, for example
`Tasks · 1/3`. A completed task is muted, so the eye goes to the work that is
left.

A pending or a completed task shows its `content`. The one in-progress task
shows its `activeForm` instead, so the running task reads in the present. When
the session is busy, the `◐` becomes the running spinner, so the active task
moves. This matches the spinner in the sidebar.

## When the panel shows and hides

The panel shows only when both of these are true:

- The selected session has at least one task.
- The output pane keeps at least 40 columns after the panel takes its width.

The panel is a fixed 32 columns. The output shrinks by that width only when the
panel shows. A narrow terminal has no room, so the panel hides and the output
takes the full width. The panel returns when the terminal is wide enough.

The panel follows the selected session. A task list from another session does
not open it.

## How the list reaches the panel

The list is derived, not stored. It is always the last `TodoWrite` in the
stream. The manager holds it in memory for a live session, and it rebuilds the
list from the transcript for a stored session.

```
session stream
  └─ TodoWrite block
        │
        ▼
  manager pump          trackTodos reads the block, updates entry.todos
        │
        ▼
  bus Event.Todos       the current list, published on every event
        │
        ▼
  TUI  m.todos[session] = ev.Todos
        │
        ▼
  taskPanelView         drawn beside the output, only when the list is not empty
```

The manager publishes the current list on every event, the same way it
publishes the streaming text. So the panel stays current for a live session.
The manager keeps the list on the close event, so a finished session still shows
what it did.

When you select a stored session, `Manager.Todos(name)` rebuilds the list from
the transcript, beside the output rebuild. A resumed session gets its prior list
the same way, so the panel is correct the moment the session starts again. See
[manager.md](../manager.md).

## The output re-wraps when the panel opens or closes

The panel changes the output width. So the output must re-wrap when the panel
opens or closes, and not only when the window changes size. The interface
compares the output width before and after each event. When the width changes
for the selected session, it rebuilds the pane instead of appending to it. See
[output.md](output.md) for the append and rebuild rules.
