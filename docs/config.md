# The settings file

One file holds the settings that are not a flag on the command line: the editor,
which `s d` opens on the working directory of the selected session, the block cap
of the session pane, and the interface layouts.

## Where the file is

The file is `config.json` in the XDG configuration directory. The directory is
named for the repository, and the two older names are still read:

| Order | Path |
|---|---|
| 1 | `$XDG_CONFIG_HOME/claude-multiplexer/config.json` |
| 2 | `$XDG_CONFIG_HOME/multiplexer/config.json` |
| 3 | `$XDG_CONFIG_HOME/multiplexier/config.json` |

`$XDG_CONFIG_HOME` defaults to `~/.config`, so the usual path is
`~/.config/claude-multiplexer/config.json`. The first file that is there wins,
and the others are not read. A tool writes the first file that is there, so a
settings file you already have keeps its place. With no file at all, the first
path is written.

`--config <path>` names one file and skips the search. A file it names must be
there, or the program stops.

`--root` moves the state directory, which holds the sessions. It does not move
this file. The sessions and the settings live apart. See
[manager.md](manager.md).

A file that is not there is not an error, and neither is an empty file. A file
that does not parse stops the program at start, because a silent fallback hides
a typing mistake.

```json
{
  "editor": "nvim",
  "editorTerminal": true,
  "blockCap": 20,
  "activeLayout": "wide"
}
```

| Field | What it holds | Read it in |
|---|---|---|
| `editor` | The command line that opens a directory, such as `code -n` | [config/editor.md](config/editor.md) |
| `editorTerminal` | `true` when the editor draws in the terminal | [config/editor.md](config/editor.md) |
| `blockCap` | The rows one block draws in the session pane before the pane caps it | [config/blocks.md](config/blocks.md) |
| `blockCaps` | A separate cap for one type of block | [config/blocks.md](config/blocks.md) |
| `layouts` | The named interface layouts, keyed by name | [config/layouts.md](config/layouts.md) |
| `activeLayout` | The layout every session takes, unless the session names its own | [config/layouts.md](config/layouts.md) |
| `defaultModel` | The model the new session form opens on | [config/new-session.md](config/new-session.md) |
| `defaultPermissionMode` | The permission mode the new session form opens on | [config/new-session.md](config/new-session.md) |
| `defaultEffort` | The effort the new session form opens on | [config/new-session.md](config/new-session.md) |
| `defaultControl` | `true` when the new session form opens on a control grant | [config/new-session.md](config/new-session.md) |

## Write any key by its path

A session writes one field with the tool that owns it, such as `set_editor` or
`set_block_cap`. It writes any field with `set_config`, and it removes any field
with `unset_config`. Both tools take a dot path and reach a nested key:

| Path | The key it writes |
|---|---|
| `editor` | The editor command |
| `blockCap` | The default block cap |
| `blockCaps.tool` | The cap for one block type |
| `layouts.wide.sidebarSize` | One dimension of the layout named `wide` |
| `defaultModel` | The model the new session form opens on |

`set_config` also takes a `value`, as any JSON value: a string, a number, a
boolean, an object, or `null`. It checks the path and the value against the
settings before it writes, so an unknown field or a wrong type fails and the
file stays as it was. A field inside a layout is checked the same way.

The check cannot catch a mistyped map key, because `blockCaps` and `layouts`
take any key. So `blockCaps.tol` writes a key the program never reads. Read the
key back with `get_config_path` and the file to confirm it.

The interface reads the file again after each write, so a change takes effect at
once. See [mcp/tools.md](mcp/tools.md).

## The pages

| Page | Read it for |
|---|---|
| [config/editor.md](config/editor.md) | Which editor `s E` opens, terminal against window editors, the file manager, and a launch that fails |
| [config/blocks.md](config/blocks.md) | The block cap: the default, a cap for one type, the question modal caps, and the tool |
| [config/layouts.md](config/layouts.md) | The named interface layouts and the global active layout |
| [config/new-session.md](config/new-session.md) | The option each field of the new session form opens on |
