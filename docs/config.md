# The settings file

One file holds the settings that are not a flag on the command line. Today it
holds the editor, which `s d` opens on the working directory of the selected
session. See [tui/keys.md](tui/keys.md).

## Where the file is

The file is `config.json` in the XDG configuration directory. The project
writes its name two ways, so both spellings are read:

| Order | Path |
|---|---|
| 1 | `$XDG_CONFIG_HOME/multiplexer/config.json` |
| 2 | `$XDG_CONFIG_HOME/multiplexier/config.json` |

`$XDG_CONFIG_HOME` defaults to `~/.config`, so the usual path is
`~/.config/multiplexer/config.json`. The first file that is there wins, and the
other one is not read.

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
  "editorTerminal": true
}
```

| Field | What it holds |
|---|---|
| `editor` | The command line that opens a directory, such as `code -n` |
| `editorTerminal` | `true` when the editor draws in the terminal. Leave it out to let the list below decide |

## Which editor

Four sources name the editor. The first one that names one wins:

| Order | Source | Example |
|---|---|---|
| 1 | `--editor` | `multiplexier --editor "code -n"` |
| 2 | `$VISUAL` | `export VISUAL=nvim` |
| 3 | `$EDITOR` | `export EDITOR=vi` |
| 4 | `editor` in the settings file | `{"editor": "zed"}` |

When no source names an editor, `s d` opens nothing, and the status bar reads
`no editor: set --editor, $EDITOR, or the config file`.

The value is a command line, split on the spaces, such as `code -n`. The
directory is added as the last argument. The command runs without a shell, so
quotes, variables, and pipes are not read.

A session can write these two fields itself, with the `set_editor` tool. It
makes the file when there is none. The interface reads the file at each `s d`,
so the next one opens the new editor. A flag still wins, because the flag sits
above the file. See [mcp.md](mcp.md).

## The terminal editor, and the window editor

A terminal editor (vim, nvim, helix) draws in the terminal, so the interface
steps aside: it releases the terminal, the editor runs in its place, and the
interface draws itself again when the editor stops.

A window editor (code, zed) has its own window, so it starts beside the
interface. The interface stays on the screen, and the status bar shows
`opened <dir>`. The editor keeps running after the interface stops.

Three sources say which kind an editor is. The first one that speaks wins:

| Order | Source | Value |
|---|---|---|
| 1 | `--editor-terminal` | `yes`, `no`, or `auto` (the default) |
| 2 | `editorTerminal` in the settings file | `true` or `false` |
| 3 | The list of known terminal editors | `ed`, `emacs`, `helix`, `hx`, `joe`, `kak`, `micro`, `nano`, `nvim`, `vi`, `vim`, `vis` |

The list reads the base name of the command, so `/usr/local/bin/nvim` is a
terminal editor. An editor the list does not name is a window editor, so say
`--editor-terminal yes` for a terminal editor that is not on the list.

## The file manager

`s f` opens the working directory in the file manager. It is not a setting,
because each platform has one:

| Platform | Command |
|---|---|
| darwin | `open <dir>` |
| windows | `explorer <dir>` |
| other | `xdg-open <dir>` |

The file manager always starts beside the interface, and never takes the
terminal.

## When a program does not start

A command that does not start — no such binary, or no permission — leaves the
reason in the status bar, such as
`editor: exec: "zed": executable file not found in $PATH`. The interface keeps
running.
