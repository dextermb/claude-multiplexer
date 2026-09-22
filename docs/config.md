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
[manager.md](manager.md). The state directory also holds the schedules, in
`schedules/`. See [scheduler.md](scheduler.md).

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
| `defaultScheduleModel` | The model a new schedule takes when it names none | [scheduler.md](scheduler.md) |
| `defaultToolProfile` | The open tools a new session carries: `minimal` or `standard` | [mcp/profiles.md](mcp/profiles.md) |
| `contextWarnPercent` | The context fill at which the governor raises a notice | [sessions/context.md](sessions/context.md) |
| `contextActPercent` | The context fill at which the governor takes its action | [sessions/context.md](sessions/context.md) |
| `contextAction` | What the governor does at the act threshold: `notify` or `hold` | [sessions/context.md](sessions/context.md) |
| `autoArchiveDays` | The days a stopped session waits, idle, before the multiplexer archives it | [sessions.md](sessions.md) |
| `costWindow` | The window the status bar total counts: `1d`, `7d`, `2w`, `1m`, or `all`. The boundary is UTC, and the default is `1d` | [cost.md](cost.md) |
| `archivedWindow` | The rolling window `l a` clamps the archived list to: `1d`, `1w`, `1m`, `1y`, or `unset` for no limit. The default is `1d` | [sessions.md](sessions.md) |
| `workItems` | The Jira and Linear work-item providers, keyed by provider, each with a token | [work-items.md](work-items.md) |
| `pullRequests` | The GitHub and GitLab pull-request providers, keyed by provider, each with a token or a CLI mode | [pull-requests.md](pull-requests.md) |
| `bars` | The composition of the session bar and the status bar: the ordered elements, and any custom script elements | [config/bars.md](config/bars.md) |
| `commands` | Key commands: a trigger, a label, and a script the press runs | [config/commands.md](config/commands.md) |

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
| `defaultScheduleModel` | The model a new schedule takes when it names none |
| `defaultToolProfile` | The open tools a new session carries |
| `contextWarnPercent` | The context fill that raises a notice |
| `contextActPercent` | The context fill that takes the action |
| `contextAction` | `notify` or `hold` |
| `autoArchiveDays` | The days before a stopped session is archived |
| `costWindow` | The window the status bar total counts, such as `1d` or `all` |
| `archivedWindow` | The rolling window `l a` clamps the archived list to, such as `1w` or `unset` |
| `workItems.linear.token` | A work-item provider token, keyed by provider |
| `pullRequests.github.token` | A pull-request provider token, keyed by provider (`github` or `gitlab`) |
| `pullRequests.gitlab.mode` | The transport of a pull-request provider: `auto`, `api`, or `cli` |
| `bars.session.left` | The ordered elements of the session bar left side |
| `bars.status.right` | The ordered elements of the status bar right side |
| `commands` | The whole list of key commands; `add_command` and `remove_command` are the tools that own it |

`set_config` also takes a `value`, as any JSON value: a string, a number, a
boolean, an array, an object, or `null`. It checks the path and the value against
the settings before it writes, so an unknown field or a wrong type fails and the
file stays as it was. A field inside a layout is checked the same way.

A client that cannot send an array or an object sends it as a string of that
JSON instead. So `set_config` reads a string that holds an array or an object as
the value it holds, and a scalar or a plain string passes through unchanged. So
`"[]"` sets an empty list, the same as `[]`.

The check cannot catch a mistyped map key, because `blockCaps` and `layouts`
take any key. So `blockCaps.tol` writes a key the program never reads. Read the
key back with `get_config_path` and the file to confirm it.

The interface reads the file again after each write, so a change takes effect at
once. See [mcp/tools/settings.md](mcp/tools/settings.md).

## An edit by hand reloads too

A tool write is not the only way the file changes. The manager also watches the
active settings file, so an edit by hand takes effect without a restart.

The watch is a poll, not a callback. Every second the manager fingerprints the
active file by its path, its modification time, and its size. When the
fingerprint changes, the manager publishes a reload, and the interface reads the
file again. The reload carries no notice, so it does not touch the status line.

The watch reads the active file only, which is the first path that is there. So
a new file at an earlier path, or a change to the file already there, both
reload. See [manager.md](manager.md).

## The update check

The `checkUpdates` key turns the hourly GitHub update check on or off. A nil
value, or true, keeps the check on; false turns it off. When the check is off, no
call goes to GitHub and no banner shows. See
[version-updates.md](version-updates.md).

```json
{
  "checkUpdates": false
}
```

## The pages

| Page | Read it for |
|---|---|
| [config/editor.md](config/editor.md) | Which editor `s E` opens, terminal against window editors, the file manager, and a launch that fails |
| [config/blocks.md](config/blocks.md) | The block cap: the default, a cap for one type, the question modal caps, and the tool |
| [config/layouts.md](config/layouts.md) | The named interface layouts and the global active layout |
| [config/new-session.md](config/new-session.md) | The option each field of the new session form opens on |
| [config/bars.md](config/bars.md) | The composition of the two status bars, the built-in elements, and custom script elements |
| [config/keybindings.md](config/keybindings.md) | Rebinding the keys of the interface, the reserved set, the action catalogue, and the precedence rule |
| [config/commands.md](config/commands.md) | Binding a key trigger to a script, the trigger forms, the payload the script reads, and the tools |
