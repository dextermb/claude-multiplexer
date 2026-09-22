# Key commands

A key command binds a key trigger to a script, so a press runs the script. For
example, `b o` runs a script that opens the default browser. A command runs the
same kind of script a custom bar element runs, but on a key press, not on a
refresh. See [bars.md](bars.md) and [keybindings.md](keybindings.md).

The `commands` block is a list. Each entry pairs a trigger with a label and a
script:

```json
{
  "commands": [
    { "keys": "b o", "label": "browser", "script": "scripts/open-browser.sh" },
    { "keys": "ctrl+g", "label": "gitui", "script": "~/bin/gitui.sh" }
  ]
}
```

## The fields

| Field | What it is |
|---|---|
| `keys` | The trigger: one or two space-separated key names. |
| `label` | A name for the command, shown in the help overlay and the notice. |
| `script` | The file to run. |

A key is a bubbletea key name, such as `ctrl+g`, `b`, or `f2`. A `label` is
unique; a second command with the same label replaces the first.

## The trigger

A trigger takes one of two forms:

- **A standalone key**, such as `ctrl+g`. The press runs the command on its own.
- **A leader and a second key**, such as `b o`. The first key (`b`) starts a
  sequence, and the second key (`o`) runs the command. A one-second timeout
  cancels the sequence, and `esc` cancels it at once. This is the same two-key
  pattern the built-in targets use (`s`, `l`, `o`, `d`).

A command fires from the main view, the same place a built-in target fires. It
does not fire inside a dialog. Inside the prompt, only a control-form trigger
(such as `ctrl+g`) fires, so a plain key stays text.

## The reserved set and clashes

The keys `?`, `esc`, and `ctrl+c` are reserved, the same as for a keybinding, so
a trigger cannot name one. See [keybindings.md](keybindings.md).

A trigger key must be free in the main view. The multiplexer refuses a command
whose standalone key or whose leader is already a key of the interface — a
target, a global key, or a key of the list, the output pane, the task panel, the
diff panel, or the prompt. This keeps a command reachable, because a built-in key
of the same name would run first.

A refused command does not load. The multiplexer keeps the commands it had, and
shows the reason on the status bar, the same way a bad keymap keeps the previous
keymap. One bad command in the list does not lose the others.

## The script

The interface picks the runner from the file extension:

| Extension | Runner |
|---|---|
| `.sh` | `bash` |
| `.py` | `python3` |
| `.go` | `go run` |

An extension outside this set is refused. The path may be absolute, start with
`~` for the home directory, or be relative to the settings directory.

The script reads a JSON object on its stdin: the selected session, or the totals
when no session is selected. This is the same payload a custom bar element reads,
so a script works for either. See the payload in [bars.md](bars.md).

The interface reads the first line of the script's stdout and shows it as a
status notice, next to the label (`browser: opened`). Empty output shows
nothing. A non-zero exit, or a run that fails, shows a one-line error notice. The
script runs off the main loop, so a slow script does not block the interface. A
ten-second timeout stops a script that runs too long. The working directory is
the settings directory.

For a worked example, see
[../example-scripts/commands/open-browser.sh](../example-scripts/commands/open-browser.sh).

## The tools

| Tool | What it does |
|---|---|
| `get_commands` | Lists the commands: each trigger, label, and script. |
| `add_command` | Binds a trigger to a script; a command with the same label is replaced. |
| `remove_command` | Removes a command by its label. |

`add_command` validates the whole set against the keymap before it writes, so it
refuses a reserved key, a bad trigger, or a clash with a key of the interface.
`set_config` reaches the same block by the path `commands`, but the two tools own
it and check each command, so prefer them. The interface reads the file again
after each write, so a change takes effect at once. See [../config.md](../config.md).
