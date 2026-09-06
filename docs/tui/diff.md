# The session diff

The interface reads the git changes of a session and shows them in two places:

- A **count** in the session bar, for example `+120 −30`.
- A **diff panel** on the right of the pane, that lists the changed files and
  expands each one to its diff.

Both read the working tree of the session working directory against
`origin/HEAD`, the default branch of the remote. So the diff is the whole branch,
the committed work and the uncommitted work together. The base falls back to
`HEAD` when `origin/HEAD` does not resolve, for a repository with no remote.

The working directory is `row.openDir`: the directory a tool of the session set,
or the directory the session started in. See [sessions.md](sessions.md).

## The count in the bar

The session bar shows the inserted lines in green and the deleted lines in red,
for example `+120 −30`. The count is the total of the changes against
`origin/HEAD`.

The count shows only when the working directory is a git work tree, and the tree
differs from `origin/HEAD`. A session outside a repository, or a tree with no
change, shows no count. The bar drops the low-priority parts first when it is short, so the count
stays while the token counts and the scroll mark go.

## The diff panel

`s d` opens the diff panel and gives it the focus. The panel lists the changed
files, one to a row:

```
┌──────────────────────────────┐
│ Changes · 3                  │
│                              │
│ ▸ M internal/tui/app.go  +12 −3 │
│ ▾ A internal/git/git.go  +40 −0 │
│   @@ -0,0 +1,40 @@            │
│   +package git               │
│   +...                       │
│ ▸ D old.txt              +0 −8 │
└──────────────────────────────┘
```

Each row shows a fold mark, a status letter, the path, and the file's own
`+I −D`. The status letter is the git letter: `M` (modified), `A` (added),
`D` (deleted), or `R` (renamed).

`Enter` expands the selected file, and its coloured diff shows below the row.
`Enter` again collapses it. The diff colours are green for an inserted line, red
for a deleted line, and blue for a hunk header. The panel drops the git file
header, because a narrow panel has no room for it. A long line wraps to the panel
width.

The panel is narrow (about 30 columns), so the diff reads best for a small file.
`pgup` and `pgdown` scroll the panel, through the list and the open diffs. The
arrow keys, or `j` and `k`, move the selected file. See [keys.md](keys.md).

The panel is not modal. While the panel is open, `Tab` adds it to the focus
cycle: sidebar, prompt, output, then the diff panel. So you tab to the prompt to
send the agent a message, and the diff stays on the screen. `s d` also moves the
focus to the panel, and `s d` again closes it. `Esc` closes it.

## The panel stays current

The panel refreshes as the agent changes files, so the list and the open diffs
follow the work. An expanded file stays expanded across a refresh, keyed by its
path.

The interface never runs git inside the view. The view stays pure. Git runs as a
command, and the result returns as a message that the update loop stores.

```
select a session / a turn ends / open the panel
        │
        ▼
  diffCmd(name, dir)                 runs git in the working directory
        │  git diff --name-status origin/HEAD
        │  git diff --numstat origin/HEAD
        ▼
  diffMsg{name, repo, stat, files}   stored in m.diffs[name]
        │
        ├──▶ the bar reads m.diffs[sel].stat
        └──▶ the panel reads m.diffs[sel].files
```

An expanded file fetches its own diff:

```
Enter on a collapsed file
        │
        ▼
  fileDiffCmd(name, dir, path)       git diff origin/HEAD -- path
        │
        ▼
  fileDiffMsg{name, path, text}      rendered lines stored in m.fileDiffs[name][path]
```

The diff refreshes at three points, each through `diffCmd`:

- The selected session changes.
- A session event ends a turn (the state leaves busy). This point keeps the open
  panel current while you work with the agent.
- The diff panel opens.

On a refresh, the update loop re-fetches the diff of each file still open, so the
open accordions follow the agent's changes. The interface adds no fast tick, so a
hand edit outside the session shows only at the next turn.

## Where it reads git

`internal/git` runs git and parses the output:

- `Diff(dir)` reads the file list and the total stat from `git diff
  --name-status origin/HEAD` and `git diff --numstat origin/HEAD`.
- `FileDiff(dir, path)` reads one file's diff from `git diff origin/HEAD --
  path`.
- `IsRepo(dir)` reports whether the directory is a git work tree.
- `baseRef(dir)` is `origin/HEAD`, or `HEAD` when the remote default does not
  resolve.

The command runner is a variable, so a test records the git arguments instead of
running git.

## The panel and the jobs panel share the slot

The diff panel uses the same place and the same width as the jobs and tasks
panel. When the diff panel is open, it takes the slot, so the jobs and tasks
panel does not show. The panel needs the same width as the jobs and tasks panel,
and it hides on a narrow terminal. See [tasks.md](tasks.md).
