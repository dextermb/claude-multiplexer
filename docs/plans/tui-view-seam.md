# The sidebar view seam

Status: **in progress** — the first step landed. Only the `Update` post-sync
step is ahead. For how the list is built today, see
[../tui/sessions.md](../tui/sessions.md).

## What landed

A pure fold, `deriveSidebar(sidebarInputs) sidebarView`, derives the rows, the
groups, and the lines with no manager, and `refresh` became read → derive →
apply. The durable description of this lives in `docs/tui/sessions.md`.

## Deferred: the `Update` post-sync

`Update()` in `app.go` still post-processes the Model after every message to
enforce invariants, so the invariants are re-synced after the fact rather than
derived:

- if `sel` changed, refresh the diff (`diffFor`) and reset the task scroll
  (`taskFor`);
- if focus is `focusTask` but the diff panel is open or there is no side panel,
  force focus back to `focusOutput`.

### The step

Make `diffFor`, `taskFor`, and the `focusTask` guard derived from `sel` and the
panel state, not re-synced after every message. A caller sets `sel`; the derived
values follow, so no caller has to remember to re-sync them.

### Why it waits

- It touches the central Model and the diff, task, and focus invariants.
- A sibling effort (`tui-modal-seam`) edits `app.go` at the same time. This step
  waits until that effort merges, so the two do not collide on the Model.

A smaller complete step beats a half-done big one, so the first step landed on
its own.
