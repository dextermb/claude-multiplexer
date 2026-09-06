# Session diff

Status: awaiting go-ahead. All decisions are settled. No code is written yet.

This plan adds two things:

1. A diff count in the top bar of the selected session.
2. A new right side panel that lists the changed files of the session. Each file
   expands in place to show its diff, and collapses again. The panel stays open
   while the user works with the agent, and its content updates as the agent
   changes files.
3. A key that collapses the left sidebar, to give the output pane and the diff
   panel more width.

Items 1 and 2 read the git working tree of the session working directory
(`openDir`).

# Decisions

- **Diff base is the working tree against HEAD.** The count and the file list
  come from `git diff HEAD` in the session working directory. This shows the
  staged and the unstaged changes that are not yet committed.
  - Rejected: branch against master (`merge-base master...HEAD`). It fits a
    worktree session better, but it needs a base branch, and the base branch is
    not always known. Working tree against HEAD needs no configuration.
  - Rejected: two counts (committed and uncommitted). It costs top-bar width and
    more state for a second number that most sessions do not need.

- **A file expands as an accordion in the panel.** Each file row expands in place
  to show its coloured diff below the row, and collapses again. The diff renders
  inside the panel, not in the output pane. The panel scrolls, and page up and
  page down move through the list and the expanded diffs.
  - Rejected: an inline viewer in the output pane. It gives more width, but it
    shows one file at a time, and it does not match the "collapse files" and
    "page through files" interaction the user asked for.
  - Rejected: an external editor or `git difftool`. The launch pattern already
    exists (`open.go`), but an in-panel accordion keeps the diff in the interface
    and needs no external program.
  - The panel is narrow (`taskPanelInner`, about 30 columns). A long diff line
    wraps to the panel width.

- **The panel stays open while the user works.** The panel is not modal. The user
  opens it, keeps it open, and continues to send prompts to the agent. The panel
  refreshes as the agent changes files, so the file list and the open diffs
  update in place. An expanded file stays expanded across a refresh, keyed by its
  path.

- **The file list is a new right side panel.** The panel sits beside the output,
  the same place as the Jobs and Tasks panel (`sidePanelView`,
  `internal/tui/app.go:2207`). It reuses the `taskPanelWidth` width machinery, so
  the output pane narrows the same way it does today.
  - Rejected: a centred body dialog. A panel keeps the transcript in view next to
    the file list, which the inline viewer needs.

- **The panel key is `s d`.** `s d` opens the session diff panel. The editor
  action moves from `s d` to `s E` (`sequenceActions`, `internal/tui/keys.go:80`).
  `s E` does not collide with `s e` (effort), because the sequence match is
  case-sensitive.

- **The count and the panel need a git work tree.** Both compute nothing unless
  `openDir` is inside a git work tree. `repoRoot` (`internal/tui/group.go:40`) is
  not the test, because it returns the directory itself when there is no
  repository. The git helper reports the work-tree state (a `git rev-parse
  --is-inside-work-tree`, or an error from the diff command), and the model holds
  a `repo bool` per session. A non-repository shows no count, and the panel shows
  "not a git repository".

- **The count format is `+120 −30`.** The count shows the inserted lines and the
  deleted lines from `--numstat`, green for the insertions and red for the
  deletions. It is not a glyph.
  - Rejected: `Δ5` (files changed). A line count tells more, and it matches the
    "diff count" request better.

- **A key collapses the left sidebar.** The sidebar (`sidebarView`,
  `internal/tui/app.go:2121`, width `sidebarWidth` = 26) hides, and the reclaimed
  width goes to the output pane and the diff panel. Three keys control it: `l t`
  toggles the sidebar, `l c` collapses (hides) it, and `l e` expands (shows) it.
  The keys act on the whole sidebar, not on a group inside it. The existing group
  fold keys (`l f`, `l F`, `l u`) do not change.

# Data flow

The interface never runs git inside `View`. `View` stays pure. All git commands
run as a `tea.Cmd`, and the result returns as a message that `update` stores on
the model.

```
select session / turn ends / open panel
        │
        ▼
  diffCmd(name, dir)                 tea.Cmd, runs git
        │  git -C dir diff --numstat --name-status HEAD
        ▼
  diffMsg{name, stat, files, err}    message
        │
        ▼
  m.diffs[name] = diffState{...}     stored on the model
        │
        ├──▶ barView reads m.diffs[m.sel].stat        → top-bar count
        └──▶ sidePanelView reads m.diffs[m.sel].files → file list
```

Enter on a collapsed file fetches that file's diff on demand:

```
Enter on a collapsed file row
        │
        ▼
  fileDiffCmd(name, dir, path)       git -C dir diff HEAD -- path
        │
        ▼
  fileDiffMsg{name, path, text, err}
        │
        ▼
  m.fileDiffs[name][path] = lines    the accordion renders the lines under the row
```

- `m.diffs` is a cache keyed by session name. It holds the last known stat and
  file list. It is derived data, so it is never persisted.
- `m.fileDiffs` is a second cache, keyed by session name and then by path. It
  holds the rendered diff lines of an expanded file.
- `m.expanded` (per session, a set of paths) records which files are open. A
  refresh keeps this set, so an open file stays open.
- A stale cache is acceptable for one frame. A refresh replaces it. A refresh
  re-fetches the diff of each file that is still open, so an open diff follows the
  agent's changes.

# Refresh points

The diff cache refreshes at these points, each through `diffCmd`:

- The selected session changes.
- A session event ends a turn (the state leaves `StateBusy`). The event handler
  is around `internal/tui/app.go:432`. This is the point that keeps the open
  panel current while the user works with the agent.
- The diff panel opens.

A refresh replaces `m.diffs[name]`, and re-fetches the diff of each path still in
`m.expanded[name]` through `fileDiffCmd`, so the open accordions follow the
agent's changes.

The plan does not add a fast periodic tick. The points above cover the moments
the diff changes because of the session. A manual refresh key can come later if a
hand edit outside the session must show without a turn.

# Workflows

**Read the count.**

1. Select a session whose working directory is a git repository.
2. The top bar shows `+120 −30` when the working tree differs from HEAD.
3. The count is absent when the directory is not a repository, or the tree is
   clean, or the diff command fails.

**Open the panel, page through files, and expand a file.**

1. Press `s d`. The panel opens and takes focus.
2. The panel shows `Changes · N`, then one row per changed file. Each row shows a
   status letter (`M`, `A`, `D`, `R`), the path, and the file's `+I −D`.
3. Page up and page down move through the list and through any expanded diffs.
   The arrow keys (or `j` and `k`) move the selected row.
4. Press Enter on a collapsed file. The file expands, and its coloured diff shows
   below the row.
5. Press Enter on an expanded file. The file collapses.
6. Keep the panel open, and send a prompt to the agent. As the agent changes
   files, the list and the open diffs update in place.
7. Press `s d` again, or Escape, to close the panel. The focus returns to the
   output.

**Collapse the sidebar for more width.**

1. Press `l c` to hide the left sidebar, or `l t` to toggle it.
2. The output pane and the diff panel widen by `sidebarWidth`.
3. Press `l e` to show the sidebar again, or `l t` to toggle it back.

**Guards and empty states.**

- No repository: the panel shows "not a git repository".
- Clean tree: the panel shows "no changes".
- The panel needs width, the same test as `showSidePanel`
  (`internal/tui/app.go:1762`). Below that width the panel does not draw.

# The build

**1. A git helper package: `internal/git`.**

- `func Diff(dir string) (Stat, []FileChange, error)` — runs
  `git -C dir diff --numstat --name-status HEAD`, or two commands, and parses the
  output. `Stat` holds `Insertions`, `Deletions`, `Files`. `FileChange` holds
  `Status` and `Path`.
- `func FileDiff(dir, path string) (string, error)` — runs
  `git -C dir diff HEAD -- path` and returns the raw diff text.
- `func IsRepo(dir string) bool` — runs `git -C dir rev-parse
  --is-inside-work-tree`. The count and the panel guard on this result.
- A non-repository or a git error returns an error. The caller treats the error
  as "no diff", and stores the error text for the empty state.
- Follow the test pattern of `open.go`: make the command runner a variable so a
  test records the arguments instead of running git.

**2. Model state (`internal/tui/app.go`, the `Model` struct near line 75).**

- `diffs map[string]diffState` — the cache. `diffState` holds `repo` (a git work
  tree yes or no), `stat`, `files`, and an `err`.
- `fileDiffs map[string]map[string][]string` — the rendered diff lines of an
  expanded file, keyed by session name then by path.
- `expanded map[string]map[string]bool` — the open files, keyed by session name
  then by path.
- `diffPanel bool` — the panel toggle.
- `diffSel int` — the selected file row in the panel.
- `diffScroll int` — the scroll offset of the panel, for page up and page down.
- `sidebarHidden bool` — the sidebar is collapsed.

The panel focus reuses the focus machinery. Add a `focusDiff` value to
`focusArea`, so `handleKey` routes the paging keys to the panel while it is open.

**3. The messages and commands (a new `internal/tui/diff.go`).**

- `diffMsg{name string, repo bool, stat git.Stat, files []git.FileChange, err error}`.
- `fileDiffMsg{name, path, text string, err error}`.
- `diffCmd(name, dir string) tea.Cmd` and `fileDiffCmd(name, dir, path string)
  tea.Cmd`, each wrapping the `internal/git` calls.
- `update` stores `diffMsg` in `m.diffs[name]`, and `fileDiffMsg` lines in
  `m.fileDiffs[name][path]`.
- On a `diffMsg`, `update` also batches a `fileDiffCmd` for each path still in
  `m.expanded[name]`, so the open accordions refresh with the new content.

**4. The top-bar count (`rightSegs`, `internal/tui/app.go:2338`).**

- Add one `barSeg` after the context segment, from `m.diffs[m.sel].stat`.
- Render `+I` in green and `−D` in red, inside `barBackground`.
- Add the segment only when the session is a git work tree (`m.diffs[m.sel].repo`)
  and the stat has changes. The `barRights` ladder keeps dropping the
  low-priority segments first, so the count survives a narrow bar because it sits
  early in the list.

**5. The panel (`sidePanelView`, `internal/tui/app.go:2207`).**

- When `m.diffPanel` is on, render a `Changes · N` section. For each
  `FileChange`, render one row with the status letter (`M`, `A`, `D`, `R`), the
  path, and the file's `+I −D`. Highlight the row at `m.diffSel`.
- Below an expanded file (a path in `m.expanded[m.sel]`), render its diff lines
  from `m.fileDiffs[m.sel][path]`.
- The whole section is a list of lines, taller than the panel. Slice the lines by
  `m.diffScroll` and the panel height, so page up and page down scroll it. This
  matches the manual paging already used elsewhere, so it does not need a bubbles
  viewport.
- Reuse `taskHeaderStyle`, `truncate`, and `taskPanelInner` for the layout. The
  diff lines wrap to `taskPanelInner`.
- Make the panel reserve width: `showSidePanel` (or a sibling) returns true when
  `m.diffPanel` is on, so `outputWidth` subtracts `taskPanelWidth`.

**6. The diff line colours (`internal/tui/styles.go`).**

- Colour the diff lines: `+` green, `−` red, `@@` hunk headers cyan, the file
  header muted, the rest default.
- Add the styles next to the existing panel styles.

**7. Keys and focus (`internal/tui/keys.go`, `handleKey`).**

- Rebind `s d` in `sequenceActions` to the diff panel toggle, and move the editor
  action to `s E` (`internal/tui/keys.go:80`).
- Opening the panel sets `m.focus = focusDiff`. Closing it (a second `s d`, or
  Escape) returns the focus to the output.
- While `focusDiff` holds the focus:
  - The arrow keys (or `j` and `k`) move `m.diffSel`.
  - Page up and page down change `m.diffScroll`.
  - Enter toggles the selected path in `m.expanded[m.sel]`. On an expand of a path
    with no cached lines, it runs `fileDiffCmd`.
- The panel is not modal. A prompt key still goes to the prompt, so the user
  works with the agent while the panel is open. Route only the paging and the
  toggle keys to the panel, the same way `handleKey` routes by focus area
  (`internal/tui/app.go:615`).

**8. The sidebar collapse (`internal/tui/app.go`).**

- Add `l t`, `l c`, and `l e` to `sequenceActions` (`internal/tui/keys.go:71`).
  `l t` toggles `m.sidebarHidden`, `l c` sets it true, and `l e` sets it false.
- `baseOutputWidth` (`internal/tui/app.go:1744`): when `m.sidebarHidden`, reclaim
  the sidebar width — `width = m.width - gutterWidth`. `outputWidth` and
  `showSidePanel` both derive from `baseOutputWidth`, so the output and the diff
  panel gain the width without a further change.
- `View` (`internal/tui/app.go:2058`): when `m.sidebarHidden`, omit
  `m.sidebarView()` from the body `JoinHorizontal`.
- Focus: a collapse moves the focus off the sidebar. When `m.focus ==
  focusSidebar` and the sidebar hides, set `m.focus = focusOutput`. While the
  sidebar is hidden, the sidebar movement keys do nothing.
- Mouse: the click math assumes a sidebar column (`internal/tui/app.go:887`,
  `:925`, `:981`). While the sidebar is hidden, the pane starts at column 0, so
  the offset must drop `sidebarWidth`. Handle the hidden case in each of the three
  places.

# Verification

- Unit test `internal/git`: feed recorded git output to the parser, and check
  `Stat` and the `[]FileChange`. Check the non-repository path returns an error.
- Unit test the top-bar segment: a model with a known `m.diffs` renders the
  count, and a clean stat renders none.
- Unit test the panel: a known file list renders the rows and the header, and an
  empty list renders the empty state.
- Unit test the accordion: an expanded path renders its diff lines under the row,
  a collapsed path renders only the row, and page up and page down move the
  visible slice.
- Unit test the sidebar collapse: with `sidebarHidden` set, `baseOutputWidth`
  reclaims `sidebarWidth`, and the body omits the sidebar. A collapse from the
  sidebar focus moves the focus to the output.
- Manual check with `just run` (or the run skill): open a repository session,
  open the panel with `s d`, expand a file, then send a prompt that edits that
  file. Confirm the count, the file list, and the open diff update while the panel
  stays open. Press `l c` and confirm the panes widen.

# When it lands

Move the durable parts into the specification, and delete or trim this plan:

- The diff data flow and the refresh points → a new `docs/tui/diff.md`, named in
  the `docs/tui.md` index.
- The panel behaviour → `docs/tui/tasks.md`, or the new `docs/tui/diff.md` when
  the panel earns its own page.
- The keys (`s d` for the panel, `s E` for the editor, `l t`/`l c`/`l e` for the
  sidebar collapse) → `docs/tui/keys.md`, and the sidebar collapse also →
  `docs/tui/sessions.md`.
- The git helper → a short note in `docs/tui/diff.md`, because no other area owns
  it yet.

Do the work in a git worktree (`just worktree session-diff`), and collapse it
when the tests pass. See `.claude/rules/worktrees.md`.
