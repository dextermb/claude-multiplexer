# Worktree working directory

The multiplexer opens the working directory of a session when the human presses
`s f` (open in Finder) or `s d` (open in editor). Keep this directory correct.

- After you move into a git worktree, call `mcp__cmux__set_working_dir` with the
  full path of the worktree.
- After you collapse the worktree, call `mcp__cmux__unset_working_dir`.

A stale working directory makes `s f` and `s d` open the wrong directory.
