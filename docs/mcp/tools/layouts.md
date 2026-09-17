# The layout tools

A layout is a named set of interface dimensions: the prompt bar height, the
session list width, the task panel width, the diff panel position, and the diff
panel size. The multiplexer resolves one layout for the selected session. See
[../../tui/layouts.md](../../tui/layouts.md).

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `list_layouts` | `session` | The named layouts, the global active layout, and the layout of one session. | open |
| `save_layout` | `name`, dimensions | Creates or replaces a layout. It captures the current dimensions, and a dimension overrides the captured one. | open |
| `delete_layout` | `name` | Removes a named layout from the settings file. | open |
| `set_layout` | `name`, `scope` | Activates a layout for this session (`scope: session`) or for all sessions (`scope: all`). | open |
| `unset_layout` | `scope` | Clears the layout of this session, or the global default. | open |

`save_layout` creates or replaces a layout. It captures the current dimensions
of the calling session, so a call with only a `name` saves what the session
shows now. A dimension given to the tool (`promptMin`, `promptMax`,
`sidebarSize`, `taskSize`, `diffSize`, or `diffPosition`) overrides the captured
one, so the same tool edits one field of a layout. The `diffPosition` is `left`,
`right`, `top`, or `bottom`. It writes the settings file.

`set_layout` activates a layout. The `scope` is `session` for the calling
session, or `all` for every session. A session layout overrides the global one.
The layout must exist, or the tool answers with an error. `unset_layout` clears
the layout of the calling session (`scope: session`) or the global default
(`scope: all`), and answers with `changed: false` when there was none.

`delete_layout` removes a layout. A session or the global default that named it
falls back to the built-in defaults. `list_layouts` reports the named layouts,
the global active layout, and the layout of one session. The named layouts and
the global choice sit in the settings file; a session layout sits in `meta.json`.
See [../../config.md](../../config.md) and [../../manager.md](../../manager.md).
