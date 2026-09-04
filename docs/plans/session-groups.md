# Plan: group the sidebar by base directory

**Status:** awaiting go-ahead. Every decision below is settled with the user.
Nothing is built yet.

The sidebar is one flat list today. Two sessions in one repository sit next to
two sessions in another, and only the name tells them apart. This plan puts a
header above each directory, and lets you fold a directory away.

## Decisions

1. **The group key is the Git repository root.** The multiplexer walks up from
   the session directory to the nearest `.git`. A session in the repository, a
   session in a subdirectory, and a session in `.worktrees/<name>` all land in
   one group. When there is no `.git` above the directory, the key is the
   directory itself.

   Rejected: the exact working directory. It is simpler, but it splits a
   worktree away from its repository, and the worktree flow in
   [../../.claude/rules/worktrees.md](../../.claude/rules/worktrees.md) makes
   that split common here.

2. **A `.git` file is read, not only a `.git` directory.** A Git worktree and a
   submodule both hold a `.git` **file** that names a gitdir, for example
   `gitdir: /repo/.git/worktrees/api`. The walk reads that file and cuts the
   path at the `/.git/` element, so the key is `/repo`. Without this step the
   walk stops at the worktree and decision 1 fails.

3. **The label is the base name of the root.** `/Users/x/projects/go/mux`
   becomes `mux`. When two groups have the same base name, both labels grow one
   path element to the left, and they grow again until they differ or a path
   runs out. So `~/a/api` and `~/b/api` read `a/api` and `b/api`.

4. **A group that holds a live session comes first.** Each group takes the rank
   of its best row: live 0, stored 1, archived 2. Groups sort by that rank, and
   a tie keeps the order the rows have today. Inside a group the order is the
   order today: live rows, then stored rows, then archived rows.

5. **Every group shows a header, even a group of one.** The list then reads the
   same however many directories are open.

6. **A group folds and unfolds.** `z` folds the group of the selected row, and
   `Z` folds or unfolds every group at once. This matches the `a` and `A` pair
   already in the key list.

7. **A folded group shows the state of what it hides.** The header of a folded
   group carries the glyph of its most urgent row, in this order: waiting,
   busy, failed, starting, idle. Without it, a fold hides a session that waits
   for your answer.

8. **The fold state is memory only.** It lives in the model, keyed by the
   group key, and it is lost when the program stops. Nothing is written to
   disk.

9. **The selection is always a visible row.** Two rules hold this true:
   - Fold the group that holds the selection, and the selection moves to the
     first row of the next unfolded group. When there is none, it moves back to
     the previous one.
   - Start a session in a folded group, and the group unfolds, because the new
     session becomes the selection.

## Open questions

None. Ask again after the feature is in use, about two points:

- Whether the fold state must survive a restart (decision 8).
- Whether a blank line between groups reads better than the header alone.

## The data flow

`refresh()` builds the list today. It will build two things instead of one:

```
manager.Snapshots()  ─┐
m.stored (Meta)      ─┴─▶  []row  (as today, one per session)
                             │
                             ├─▶ groupKey(row.dir)      cached: dir ─▶ root
                             │
                             ▼
                        m.groups []group{key,label,folded,rank}
                        m.rows   sorted by (group rank, group, row rank)
                             │
                             ▼
                        m.lines  []listLine{group int, row int}
                                 row = -1 marks a header
```

`m.lines` is the display list, one entry per line on the screen. A folded group
contributes its header and nothing else. Everything that counts lines on the
screen reads `m.lines`:

- `sidebarView` renders `m.lines[m.listOffset:]`.
- `clampOffset` finds the display line of the selected row.
- The mouse maps a `Y` inside the sidebar to `m.lines[m.listOffset+Y]`.
- `move(delta)` steps over the entries of `m.lines` that are rows, so a header
  never takes the selection and a folded row is never reached.

`m.rows` keeps its meaning, so `selIndex`, `selectedRow`, and every caller of
them stay as they are.

**The group key is cached.** The walk touches the disk, and `refresh()` runs on
every event. A session directory never changes, so a `map[string]string` from
directory to root is exact, and each directory is read once.

## The workflow

```
 ▾ claude-multiplexer   3     a header: mark, label, and the row count
 ● api                        the rows of the group, indented by one column
 ⠹ docs             ⇢2
 ○ invoices
 ▸ landing-page  ⠹  2         a folded group: the mark, the count, and the
 ▾ scratch          1         glyph of the most urgent row it hides
 · old-notes
```

- `z` on any row folds its group. The header keeps its place, and the rows go.
- `z` again unfolds it, and the selection stays where it was when it can.
- `Z` folds every group. Press it again and every group unfolds.
- A click on a header folds or unfolds that group.
- A click on a row selects it, as it does today.
- The wheel over the sidebar moves the selection, as it does today.

## The build

**New file `internal/tui/group.go`:**

- `repoRoot(dir string) string` — walks up, reads a `.git` file when it finds
  one, and returns the directory itself when it finds no `.git`.
- `groupLabels(roots []string) map[string]string` — the base name, grown left
  while two labels collide.
- `type group struct { key, label string; folded bool; rank int; glyph ... }`.
- `type listLine struct { group, row int }`.

**Changed in `internal/tui/app.go`:**

- `Model` gains `groups []group`, `lines []listLine`, `folded map[string]bool`,
  and `roots map[string]string` (the cache).
- `refresh()` groups and sorts, then builds `m.lines`.
- `move`, `clampOffset`, `visibleRows` (renamed `visibleLines`), the sidebar
  mouse branch, and `sidebarView` read `m.lines`.
- `sidebarKey` gains `z` and `Z`.
- `handleSpawned` unfolds the group of the new session.

**Changed in `internal/tui/styles.go`:** a header style, and the fold marks
`▾` and `▸`.

**Changed in `internal/tui/help.go`:** two entries in `bindings`, in the
`Sessions` group, so the key list and the key page cannot drift.

**The documents, in the same change:**

- [../tui/sessions.md](../tui/sessions.md) — a section on the groups: the key,
  the label, the order, the header, and the fold.
- [../tui/keys.md](../tui/keys.md) — `z` and `Z` in the table, and the header
  click under the mouse.
- [../tui.md](../tui.md) — the layout diagram shows headers.

`sessions.md` is 149 lines, so the new section fits under the 300-line cap in
[../../.claude/rules/documentation.md](../../.claude/rules/documentation.md).

## How it is verified

New tests in `internal/tui`:

| Test | What it holds |
|---|---|
| `group_test.go` | `repoRoot` finds a `.git` directory, reads a `.git` file for a worktree and a submodule, and falls back to the directory |
| `group_test.go` | `groupLabels` gives base names, and grows them when two collide |
| `groups_test.go` | Rows sort by group rank, and live groups come first |
| `groups_test.go` | `m.lines` holds a header for each group, and a folded group hides its rows |
| `groups_test.go` | `move` steps over a header, and never lands in a folded group |
| `groups_test.go` | Folding the group of the selection moves the selection out |
| `groups_test.go` | A click on a header line folds the group |
| `groups_test.go` | `listOffset` counts the header lines when the list scrolls |
| `groups_test.go` | The header of a folded group shows the most urgent glyph |

The work is done in a worktree, and `just check` is green before it collapses.
See [../../.claude/rules/worktrees.md](../../.claude/rules/worktrees.md).

## When it lands

Move decisions 1 to 9 into `docs/tui/sessions.md` as the description of the
groups, repoint anything that names this plan, and delete this file. See
[../../.claude/rules/plans.md](../../.claude/rules/plans.md).
