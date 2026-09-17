# The settings tools

These tools read and write the settings file of the multiplexer, and they name
the files a session reads. See [../../config.md](../../config.md).

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `get_config_path` | — | The settings files, in the order they are read, the one that is read now, and the one a write goes to. | open |
| `get_template_path` | `session` | The directories a session reads a preset prompt from, in the order they are read. | open |
| `set_config` | `path`, `value` | Sets one settings key by a dot path, such as `blockCaps.tool`. It rejects a key or a type the settings do not allow. | open |
| `unset_config` | `path` | Removes one settings key by a dot path, so that key takes its default again. | open |
| `set_editor` | `editor`, `terminal` | Sets the editor the human opens a directory with, in the settings file. | open |
| `unset_editor` | `field` | Takes the editor, the terminal flag, or both out of the settings file. | open |
| `set_block_cap` | `rows` | Sets the rows one block draws in the session pane before the pane caps it. `0` caps nothing. | open |
| `unset_block_cap` | — | Takes the block cap out of the settings file, so the pane returns to 20 rows. | open |
| `set_auto_archive` | `days` | Archives a stopped session after it is idle for this many days. `days` must be one or more. | open |
| `unset_auto_archive` | — | Turns auto-archive off, so a stopped session stays until the human archives it. | open |

### The paths a session reads

A session cannot see where its own settings and preset prompts come from, and
the paths follow the flags the human started the program with. Two tools answer
that, so a session names a real file before it writes one.

`get_config_path` answers with three fields:

| Field | What it holds |
|---|---|
| `paths` | Every settings file, in the order they are read |
| `active` | The file that is read now. It is absent when there is none |
| `target` | The file `set_editor` and `set_block_cap` write |

`get_template_path` answers with `dirs`, the directories one session reads a
preset prompt from, in the order they are read, and the last one wins. It also
names the `root` and the `dir` of the session. Give a session name, or leave it
empty for the calling session.

That `dir` is the one the session started in, which is the one the interface
reads, and not the one `set_working_dir` names. Both tools only read. See
[../../config.md](../../config.md) and [../../templates.md](../../templates.md).

### The editor

`set_editor` writes the settings file of the multiplexer, and makes that file
when there is none. `editor` is the command line, such as `code -n`.
`terminal` says whether that editor draws in the terminal. A call must give at
least one of the two, and a field it does not give keeps its value. The tool
answers with the path it wrote.

`unset_editor` takes those fields out again. `field` is `editor`,
`terminal`, or `both`, and `both` is the default. The human then falls back to
the rung below in the ladder: the environment, and then the settings of Claude
Code. The tool answers with `changed: false` when the field was not set, and it
makes no file in that case, so a call is safe to repeat.

The interface reads that file each time the human presses `s d`, so a change
takes effect at once, with no restart. `--editor` and `$EDITOR` still sit above
the file, so a program started with `--editor` opens what the flag names, and
the tool cannot change that. The tool writes the settings file of the
multiplexer, never the settings of Claude Code. See
[../../config.md](../../config.md).

### The block cap

The session pane caps a long block and offers to open it in place. A block is
one piece of content: a prompt, one message, one tool result, or the output of a
`!` command. See [../../tui/output.md](../../tui/output.md).

`set_block_cap` writes the cap to the settings file, the same file `set_editor`
writes. A `type` (prompt, message, tool, meta, bash, or error) caps one kind of
block; no type sets the default for the rest. The `skill` type caps the content
of a skill loaded into the transcript, and defaults to 1 row. The
`question_option` and `question_description` types cap the question modal, not
the pane, and default to 2 lines. Give `rows` to draw that many rows (`0` draws only the marker), or
`unlimited: true` to never cap. Give one, not both. A number below zero is an
error. So a session that is about to print a large report can raise the cap, and
lower it again after.

`unset_block_cap` takes a cap out again. A `type` clears that one kind, so it
takes the default again; no type clears the default, and the pane returns to 20
rows. It answers with `changed: false` when the file held no such cap.

The interface reads the settings file again at each notice, so a new cap reaches
the pane at once and the pane draws itself again. `--block-cap` still sits above
the file. See [../../config.md](../../config.md).

### Auto-archive

`set_auto_archive` writes the number of days a stopped session waits, idle,
before the multiplexer archives it. Give `days` as one or more; a value below one
is an error. The setting holds for every session, and an hourly sweep archives
each stopped session past the limit. See [../../sessions.md](../../sessions.md).

`unset_auto_archive` turns the feature off again, so a stopped session stays until
the human archives it. It answers with `changed: false` when the file held no
setting. `set_config` reaches the same field by the path `autoArchiveDays`.
