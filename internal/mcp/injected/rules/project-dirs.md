# Work on many code bases in one project

A session works in one directory by default. When one change spans two or more
code bases, add each code base to the project. Then the diff panel shows the
changes of every code base in one grouped view.

- Add the root directory of a code base, not a subdirectory that the change
  touches. For a change in `repo/docs`, add `repo`, so the diff shows the whole
  code base.
- Call `mcp__cmux__add_project_dir` once for each code base the change spans.
- Call `mcp__cmux__remove_project_dir` when a code base is no longer part of the
  change, or `mcp__cmux__clear_project` to empty the whole project.
- Use a project for one change across many code bases, not for unrelated work.
