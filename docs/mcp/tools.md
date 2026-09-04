# Every tool

Each tool reaches the model as `mcp__cmux__<tool>`. A tool marked `open` goes to
every session, and a tool marked `control` goes only to a session that holds the
grant. See [grant.md](grant.md).

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `rename_session` | `title` | Sets the title of the calling session. An empty title clears it. | open |
| `list_sessions` | `live_only` | Every session, running and stored: name, title, directory, state, model, turns, cost, and the archive flag. | open |
| `get_messages` | `session`, `limit` | The recent messages of a session, oldest last. 20 by default, 200 at most. | open |
| `list_jobs` | `session` | The background jobs of a session: id, description, task type, and status. An empty session means the caller. | open |
| `get_config_path` | — | The settings files, in the order they are read, the one that is read now, and the one a write goes to. | open |
| `get_template_path` | `session` | The directories a session reads a preset prompt from, in the order they are read. | open |
| `set_editor` | `editor`, `terminal` | Sets the editor the human opens a directory with, in the settings file. | open |
| `unset_editor` | `field` | Takes the editor, the terminal flag, or both out of the settings file. | open |
| `set_block_cap` | `rows` | Sets the rows one block draws in the session pane before the pane caps it. `0` caps nothing. | open |
| `unset_block_cap` | — | Takes the block cap out of the settings file, so the pane returns to 20 rows. | open |
| `set_working_dir` | `path` | Says which directory the calling session works in now. | open |
| `unset_working_dir` | — | Takes the working directory off the calling session. | open |
| `send_message` | `session`, `text` | Queues a prompt for another session, and returns the queue length. | control |
| `stop_session` | `session` | Ends another child in a clean way. Its transcript is kept. | control |
| `archive_session` | `session`, `restore` | Takes a stopped session out of the list, or with `restore` brings it back. | control |
| `create_session` | `path`, `name` | Starts a new session in a directory. Returns the name it takes. | control |
| `stop_job` | `session`, `job` | Interrupts a session and asks it to kill one background job. An empty session means the caller. | control |

`create_session` takes a directory path and an optional name. The directory
must exist. The manager makes the name unique, and it falls back to the last
element of the path when the name is empty, so the tool returns the real name
the session takes. The new session starts without the control grant.

The manager writes the name of the caller into the record of the new session,
as its creator. The sidebar groups a session under the control session that
created it, and the record keeps that group after a restart. See
[tui/sessions.md](../tui/sessions.md) and [manager.md](../manager.md).

`get_messages` reads the transcript on disk, so it answers for a stored session
as well as a live one. It returns one entry for each user prompt, each assistant
message, and each result. A tool call becomes the short line `[used Bash]`,
because the whole input of a tool call is large and it is rarely what a reader
wants.

`list_jobs` reads the background jobs of a session; see
[sessions.md](../sessions.md). It reads the live session, so it returns an empty
list for a stored session, which has no running jobs.

`stop_job` cannot reach a Claude Code background shell directly, because the
shell runs inside the child. So the tool interrupts the current turn of the
owning session, and it queues an instruction to run `KillShell` on the exact
shell. The interrupt ends the turn at once, so the instruction runs on the next
turn. The tool marks the pane with `← stop job <id> from <caller>`, the same way
`send_message` marks a prompt. It finds the job by its id first, so it never
interrupts a turn for a job that does not exist or already stopped.

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
multiplexer, never the settings of Claude Code. See [config.md](../config.md).

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
[config.md](../config.md) and [templates.md](../templates.md).

### The block cap

The session pane caps a long block and offers to open it in place. A block is
one piece of content: a prompt, one message, one tool result, or the output of a
`!` command. See [tui/output.md](../tui/output.md).

`set_block_cap` writes `rows` to the settings file, the same file `set_editor`
writes. `0` caps nothing, and a number below zero is an error. So a session that
is about to print a large report can raise the cap, and lower it again after.

`unset_block_cap` takes the cap out again, and the pane returns to its default
of 20 rows. It answers with `changed: false` when the file held none.

The interface reads the settings file again at each notice, so a new cap reaches
the pane at once and the pane draws itself again. `--block-cap` still sits above
the file. See [config.md](../config.md).

### The working directory

A session starts in one directory, and the stream never says that the agent
moved. The `cwd` field arrives on the first event of a session and on no other,
and a `Bash` tool call carries no directory. So a session that moves into a
worktree says so itself, with `set_working_dir`.

`path` is the directory. A relative path is resolved against the directory the
session started in, so a session in `~/work/api` sets `.worktrees/feature` and
means `~/work/api/.worktrees/feature`. The directory must exist, or the tool
answers with an error and nothing changes.

`s f` and `s d` then open that directory. They fall back to the directory the
session started in when no working directory is set, and when the one that is
set is gone, because a collapsed worktree leaves a path that no longer opens.
See [tui/keys.md](../tui/keys.md).

`unset_working_dir` takes it off again, and answers with `changed: false` when
there was none. The working directory sits in `meta.json`, so a resumed session
keeps it. See [manager.md](../manager.md).
