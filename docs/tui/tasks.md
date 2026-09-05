# The side panel

A session keeps a background job list and a task list for itself. This panel
shows both lists on the right of the session pane, so you can see what the
session runs and what it plans. The jobs sit above the tasks.

```
┌─────────────────────────────────┬──────────────────────────────┐
│ › add the panel                 │ Jobs · 1/2                   │
│ → TaskCreate Draw the panel     │                              │
│ ← result                        │ ⚙ build the binary           │
│ echo: add the panel             │ ✓ run the tests              │
│                                 │                              │
│                                 │ Tasks · 1/3                  │
│                                 │                              │
│                                 │ ✔ Add the render type        │
│                                 │ ◐ Wiring the manager         │
│                                 │ ○ Draw the panel             │
└─────────────────────────────────┴──────────────────────────────┘
```

## The two sections

The panel holds up to two sections, in this order:

- The **jobs section** lists the background jobs of the session. See
  [sessions.md](sessions.md) for the job model and the other places a job shows.
- The **tasks section** lists the tasks the session keeps with its `TaskCreate`
  and `TaskUpdate` tools.

A section shows only when its list is not empty. So a session with jobs but no
tasks shows the jobs section alone, and a session with tasks but no jobs shows
the tasks section alone.

## The jobs section

The jobs section lists every background job, the running ones first, then the
finished ones, in start order. The header counts the running jobs against the
total, for example `Jobs · 1/2`.

Each row shows a status glyph, then the job description:

| Glyph | Status | Colour |
|---|---|---|
| `⚙` | running | Amber |
| `✓` | done | Green |
| `✗` | failed | Red |
| `⊗` | killed | Muted |

A finished job is muted, so the eye goes to the work that still runs.

## The tasks section

The section lists every task, one to a row, with a status glyph:

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

Each task has three parts:

- `content` — the command form, for example "Wire the manager".
- `activeForm` — the present form, for example "Wiring the manager".
- `status` — one of `pending`, `in_progress`, or `completed`.

A session keeps its task list with two tools, `TaskCreate` and `TaskUpdate`.
The list is incremental, not a whole-list replace:

- A `TaskCreate` tool_use block adds one task. The block holds the task
  `subject` (the `content`) and the `activeForm`. A new task starts `pending`.
- A `TaskUpdate` tool_use block changes one task. The block holds the task id
  and the new `status`.

The session gives each task an id in creation order. So the first `TaskCreate`
makes task `1`, the second makes task `2`, and a `TaskUpdate` names the task by
that id. See [protocol.md](../protocol.md).

`protocol.TaskTracker` folds the stream into the list. `Apply` adds a task for
each `TaskCreate` block, and sets the status for each `TaskUpdate` block.
`List` returns the tasks in creation order.

An older session uses one `TodoWrite` tool instead. That block holds the whole
list, so it replaces the last list. `TaskTracker.Apply` still reads it, so an
old transcript keeps its tasks section. An empty `TodoWrite` clears the list.

## When the panel shows and hides

The panel shows only when both of these are true:

- The selected session has at least one job or one task.
- The output pane keeps at least 40 columns after the panel takes its width.

The panel is a fixed 32 columns. The output shrinks by that width only when the
panel shows. A narrow terminal has no room, so the panel hides and the output
takes the full width. The panel returns when the terminal is wide enough.

The panel follows the selected session. A job or a task list from another
session does not open it.

## How the lists reach the panel

Both lists are derived, not stored, and each has its own source. The jobs come
from the session snapshot, and the tasks come from the manager bus.

```
session stream
  └─ TaskCreate / TaskUpdate block      task_* events
        │                                   │
        ▼                                   ▼
  manager pump                          session Job list
  trackTodos → TaskTracker                  │
        │                                   ▼
        ▼                          snapshot row.jobList  (m.selectedJobs)
  bus Event.Todos                          │
        │                                   │
        ▼                                   │
  TUI m.todos[session]                      │
        └──────────────┬────────────────────┘
                       ▼
                 sidePanelView   drawn beside the output when a list is not empty
```

The tasks list is the fold of every `TaskCreate` and `TaskUpdate` in the stream
(or the last `TodoWrite`, for an old session). The manager holds a `TaskTracker`
per live session, and publishes the list on every event, the same way it
publishes the streaming text. So the tasks section stays current for a live
session. The manager keeps the list on the close event, so a finished session
still shows what it did.

When you select a stored session, `Manager.Todos(name)` rebuilds the tasks list
from the transcript, beside the output rebuild. A resumed session gets its prior
list the same way, so the section is correct the moment the session starts
again. See [manager.md](../manager.md).

The jobs list is the `row.jobList` the snapshot carries. So the jobs section
shows only for a live session, and it clears when the session stops. See
[sessions.md](sessions.md) for how the session derives a job from the stream.

## The output re-wraps when the panel opens or closes

The panel changes the output width. So the output must re-wrap when the panel
opens or closes, and not only when the window changes size. The interface
compares the output width before and after each event. When the width changes
for the selected session, it rebuilds the pane instead of appending to it. See
[output.md](output.md) for the append and rebuild rules.
