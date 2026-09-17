# The working directory and the project tools

These tools say which directories a session works in, so `s f` and `s E` open
the right one. See [../../tui/keys.md](../../tui/keys.md).

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `set_working_dir` | `path` | Says which directory the calling session works in now. | open |
| `unset_working_dir` | — | Takes the working directory off the calling session. | open |
| `list_project` | `session` | The directories of a session's project, in order. An empty session means the caller. | open |
| `add_project_dir` | `path` | Adds one directory to the caller's project. | open |
| `remove_project_dir` | `path` | Takes one directory out of the caller's project. | open |
| `set_project` | `paths` | Replaces the whole ordered set of the caller's project directories. | open |
| `clear_project` | — | Empties the caller's project, so it works in one directory again. | open |

### The working directory

A session starts in one directory, and the stream never says that the agent
moved. The `cwd` field arrives on the first event of a session and on no other,
and a `Bash` tool call carries no directory. So the multiplexer reads the
worktree tools the agent runs, and a session names another directory itself with
`set_working_dir`.

The agent enters a git worktree with the `EnterWorktree` tool, and leaves it
with `ExitWorktree`. The multiplexer watches both in the stream, the same way it
reads a background job. On a `EnterWorktree` that succeeds, it sets the working
directory to the new worktree. On an `ExitWorktree` that succeeds, it clears the
working directory. The path comes from the `EnterWorktree` result, which names
the worktree as `worktree at <path> on branch <branch>` for a new worktree and
for one it enters by path, and falls back to the tool input. The multiplexer
reads these tools in the manager pump. See [../../manager.md](../../manager.md).

`set_working_dir` names a move the worktree tools do not, and a person can call
it too. `path` is the directory. A relative path is resolved against the
directory the session started in, so a session in `~/work/api` sets
`.worktrees/feature` and means `~/work/api/.worktrees/feature`. The directory
must exist, or the tool answers with an error and nothing changes.

`s f` and `s E` then open that directory. They fall back to the directory the
session started in when no working directory is set, and when the one that is
set is gone, because a collapsed worktree leaves a path that no longer opens.
See [../../tui/keys.md](../../tui/keys.md).

`unset_working_dir` takes it off again, and answers with `changed: false` when
there was none. The working directory sits in `meta.json`, so a resumed session
keeps it. See [../../manager.md](../../manager.md).

### The project

A session works in one directory by default. A project lets one session work in
several directories at once, so one change can span two or more code bases. The
diff panel groups the changes by directory, one section for each. See
[../../tui/diff.md](../../tui/diff.md).

`add_project_dir` adds one directory, and `remove_project_dir` takes one out.
`set_project` replaces the whole ordered set with `paths`, and an empty list
clears it. `clear_project` empties the project, and answers with
`changed: false` when there was none. `list_project` reads the directories, in
order.

Each tool resolves and validates a path the same way `set_working_dir` does: a
relative path is resolved against the directory the session started in, and the
directory must exist. The set holds no duplicate, and it keeps the order the
directories were added. The project sits in the `working_dirs` field of
`meta.json`, so a resumed session keeps it. See
[../../manager.md](../../manager.md).

When a session has a project, `s f` and `s E` open the first directory of the
set, unless `set_working_dir` names another. See
[../../tui/keys.md](../../tui/keys.md).
