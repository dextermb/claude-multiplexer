# Worktree working directory

The multiplexer opens the working directory of a session when the human presses
`s f` (open in Finder) or `s E` (open in editor).

The multiplexer keeps this directory correct on its own. It sets the working
directory when you enter a git worktree with `EnterWorktree`, and it clears the
directory when you leave with `ExitWorktree`. So you do not need to call
`mcp__cmux__set_working_dir` for a worktree.

Call `mcp__cmux__set_working_dir` only when you move into a directory that the
worktree tools do not name. Call `mcp__cmux__unset_working_dir` to clear it.
