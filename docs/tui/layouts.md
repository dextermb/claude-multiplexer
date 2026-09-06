# The layouts

A layout is a named set of interface dimensions. The multiplexer resolves one
layout for the selected session, and it draws the sidebar, the panels, and the
prompt at those dimensions.

## The four dimensions

| Dimension | What it sets | Built-in default |
|---|---|---|
| prompt bar | the least and the most rows the prompt bar draws | 1 and 4 |
| session list | the width of the session list sidebar | 26 |
| task panel | the width of the task and background job panel | 32 |
| diff panel | the width of the diff panel | 32 |

The prompt bar still grows to fit the text, between the least and the most rows.
The diff panel width is the width `s d` opens. `d +` and `d -` change it live,
and that change is transient, so it does not write the layout. See
[diff.md](diff.md).

## The precedence

The multiplexer resolves the dimensions of the selected session in three steps:

```
session override  →  global active  →  built-in defaults
```

1. Start from the built-in defaults.
2. Overlay the global active layout, if the settings file names one.
3. Overlay the session layout, if the session names one.

A layout sets only the dimensions it holds, and the rest fall through to the next
step. A name that no layout holds falls through too, so a deleted layout never
breaks a session. Every width stays inside the terminal, so a large layout does
not break the render.

## Where a layout lives

The named layouts and the global active layout live in the settings file, with
the editor and the block caps. See [../config.md](../config.md). The layout of
one session lives in that session's `meta.json`, with its model and its mode.
See [../manager.md](../manager.md).

## The switcher: `o l`

`o l` opens the layout switcher. It lists the named layouts, and it marks the one
active for the selected session. It only activates a layout. It does not create
or edit one.

- `↑` and `↓` move the cursor.
- `Tab` switches the scope between "this session" and "all sessions".
- `Enter` activates the layout at the scope.
- `Esc` closes the switcher.

The scope "this session" sets the session override. The scope "all sessions"
sets the global active layout. An empty list names the `save_layout` tool,
because a layout is made by a tool or by the settings file, not by the interface.

## Create and edit a layout

The interface has no key that creates or edits a layout. Use the MCP tools, or
edit the settings file. See [../mcp/tools.md](../mcp/tools.md) for `list_layouts`,
`save_layout`, `delete_layout`, `set_layout`, and `unset_layout`.

`save_layout` captures the current dimensions of the calling session, so a call
with only a name saves what the session shows now. A dimension given to the tool
overrides the captured one, so the same tool edits one field of a layout.
