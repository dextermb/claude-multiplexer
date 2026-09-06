# The layout settings

A layout is a named set of interface dimensions. `layouts` holds the named
layouts, and `activeLayout` names the one every session takes. A session names
its own layout in its `meta.json`, which overrides `activeLayout`. This page
covers the settings-file shape; for the layouts in the interface, and the
switcher, see [../tui/layouts.md](../tui/layouts.md) and [../manager.md](../manager.md).

```json
{
  "activeLayout": "wide",
  "layouts": {
    "wide":    { "sidebarWidth": 34, "diffWidth": 60, "promptMax": 6 },
    "compact": { "sidebarWidth": 20, "taskWidth": 24, "promptMax": 2 }
  }
}
```

| Field | What it sets | Built-in default |
|---|---|---|
| `promptMin` | The least rows the prompt bar draws | 1 |
| `promptMax` | The most rows the prompt bar grows to | 4 |
| `sidebarWidth` | The columns of the session list sidebar | 26 |
| `taskWidth` | The columns of the task and background job panel | 32 |
| `diffWidth` | The columns of the diff panel | 32 |

A layout sets only the fields it holds, and the rest take the built-in default.
The multiplexer resolves the dimensions in three steps: the session layout, then
`activeLayout`, then the built-in defaults. The `save_layout`, `set_layout`, and
other tools write these fields. See [../mcp/tools.md](../mcp/tools.md).
