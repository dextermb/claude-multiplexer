# The editor

`s d` opens the working directory of the selected session in your editor. This
page names the editor, says how the interface runs it, and covers the file
manager that `s f` opens. For where the settings file lives, see
[../config.md](../config.md).

## Which editor

Five sources name the editor. The first one that names one wins:

| Order | Source | Example |
|---|---|---|
| 1 | `--editor` | `multiplexer --editor "code -n"` |
| 2 | `$VISUAL` | `export VISUAL=nvim` |
| 3 | `$EDITOR` | `export EDITOR=vi` |
| 4 | `editor` in the settings file | `{"editor": "zed"}` |
| 5 | `env.VISUAL`, then `env.EDITOR`, in the settings of Claude Code | `{"env": {"EDITOR": "zed --wait"}}` |

When no source names an editor, `s d` opens nothing, and the status bar reads
`no editor: set --editor, $EDITOR, or the config file`.

### The settings of Claude Code

Many people already name their editor in `~/.claude/settings.json`, in the
`env` block that Claude Code puts into the environment of a shell. So the last
rung reads that file, and you get an editor without setting one twice.

`$CLAUDE_CONFIG_DIR` moves the file, the same as it does for Claude Code
itself. The program reads the global file only, not the settings of a project.

The file belongs to another program, so this one is careful with it: it never
writes it, and a file that is missing, or that does not parse, is not an error.
It simply names no editor. `set_editor` always writes the settings file of the
multiplexer.

The value is a command line, split on the spaces, such as `code -n`. The
directory is added as the last argument. The command runs without a shell, so
quotes, variables, and pipes are not read.

A session can write these two fields itself, with the `set_editor` tool, and
take them out again with `unset_editor`. `set_editor` makes the file when there
is none, and `unset_editor` makes none. The interface reads the file at each
`s d`, so the next one opens the new editor. A flag still wins, because the
flag sits above the file. See [../mcp/tools.md](../mcp/tools.md).

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
