# Work on many code bases in one project

A session works in one directory by default. When one change spans two or more
code bases, add each directory to the project. Then the diff panel shows the
changes of every directory in one grouped view.

- Call `mcp__cmux__add_project_dir` for each directory the change touches.
- Call `mcp__cmux__remove_project_dir` when a directory is no longer part of the
  change, or `mcp__cmux__clear_project` to empty the whole project.
- Use a project for one change across many code bases, not for unrelated work.
