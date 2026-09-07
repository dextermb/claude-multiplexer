# The session diff

The interface reads the git changes of a session and shows them in two places:

- A **count** in the session bar, for example `+120 −30`.
- A **diff panel** on a side of the pane, that lists the changed files and
  expands each one to its diff. A layout sets the side: left, right (the
  default), top, or bottom. See [layouts.md](layouts.md).

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
`D` (deleted), or `R` (renamed). The selected file has a purple background.

`Enter` expands the selected file, and its coloured diff shows below the row.
`Enter` again collapses it. `Space` does the same as `Enter` here. The diff
colours are green for an inserted line, red for a deleted line, and blue for a
hunk header. The panel drops the git file
header, because a narrow panel has no room for it. A long line wraps to the panel
width.

## The position and the grid

A layout puts the diff panel on any of four sides. Left and right are vertical
sides, so the panel is a narrow column beside the output. Top and bottom are
horizontal sides, so the panel is a wide band above or below the output.

On a vertical side the files draw in one column, and an open file's diff shows
inline, right after its row. On a horizontal side the panel is short and wide,
so the files draw in a grid of two or more columns. An open file's diff then
draws in one region below the whole grid, at the full width. Two open files
stack in that region, in file order.

```
top or bottom (a grid, then the open diffs below all files)

 Changes · 5
 ▸ M app.go      +12 −3    ▸ A git.go   +40 −0
 ▾ M layout.go   +8 −1     ▸ D old.txt   +0 −8
 ▸ M diff.go     +5 −2

 layout.go
 @@ -1,4 +1,8 @@
 +package tui
```

The size a layout sets is columns on a vertical side, and rows on a horizontal
side. See [layouts.md](layouts.md).

`d n` shows or hides the line numbers. The number is the new-file line, in a
grey gutter on the left. A deleted line has no new-file number, so its gutter is
blank.

## Scrolling and the size

On a vertical side the panel scrolls through the list and the open diffs. `j` and
`k` move a cursor down and up. On a collapsed file, `j` and `k` move to the next
and the previous file. On an open file, `j` and `k` step through the diff one
line at a time. At the bottom of an open diff, `j` moves to the next file, so you
can expand it. `k` into an open file above starts at the bottom of its diff, then
steps up to the top.

On a horizontal side the selection moves in the grid. `h` and `l` move one file
left and right. `j` and `k` move one grid row down and up. The open diffs sit
below the grid, so the arrow keys and the page keys reach them by scrolling.

The arrow keys scroll the panel one line, `pgup` and `pgdown` move a page, and
the mouse wheel moves a few lines. See [keys.md](keys.md).

`g` and `G` go to the top and the bottom. Their scope depends on the panel. If a
file is open, `g` and `G` scroll the panel to the top and the bottom of the diff.
If no file is open, `g` and `G` move the selection to the first and the last file
in the list.

`}` and `{` (`shift+]` and `shift+[`) jump to the next and the previous empty
line of an open diff, like the paragraph motions of vim. An empty line is a diff
line with no code, once its marker and its line number are removed.

The panel marks the current line, the top line of the view, so you see where a
jump lands. When the line numbers are off, the current line is bold. When they
are on, the line number is bold.

`d +` grows the panel, and `d -` shrinks it. The keys change the width on a
vertical side, and the height on a horizontal side. The size has a minimum, and a
maximum that keeps the output above its own minimum. The panel remembers its
size, so a hide and a later show keep it.

`d /` shows the panel at half the screen, and `d /` again returns it to the set
size. `d +` or `d -` leaves the half mode and adjusts the set size. On a
horizontal side, the half mode is half the screen height.

The panel is not modal. While the panel is open, `Tab` adds it to the focus
cycle: sidebar, prompt, output, then the diff panel. So you tab to the prompt to
send the agent a message, and the diff stays on the screen. `s d` also moves the
focus to the panel, and `s d` again closes it. `Esc` closes it.

## The panel stays current

The panel refreshes as the agent changes files, so the list and the open diffs
follow the work while a turn runs. An expanded file stays expanded across a
refresh, keyed by its path.

While the panel is open, a tick re-reads the diff every 800 milliseconds, so the
changes of a running agent show as they arrive, not only at the end of the turn.
The tick stops when the panel closes.

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

The diff refreshes at these points, each through `diffCmd`:

- The selected session changes.
- A session event ends a turn (the state leaves busy).
- The diff panel opens.
- A tick, every 800 milliseconds, while the panel is open. This point shows the
  changes of a running agent as they arrive, not only at the end of the turn.

On a refresh, the update loop re-fetches the diff of each file still open, so the
open accordions follow the agent's changes.

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

The diff panel uses the same place as the jobs and tasks panel. When the diff
panel is open, it takes the slot, so the jobs and tasks panel does not show. The
jobs and tasks panel is always on the right. The diff panel takes the side and
the size a layout sets, and `d +` and `d -` change the size. The panel hides
when the terminal is too small for the output and the panel. See
[tasks.md](tasks.md) and [layouts.md](layouts.md).
