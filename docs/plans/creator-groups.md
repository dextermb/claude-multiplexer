# Plan: group the sessions a control session created

**Status:** awaiting go-ahead. The two questions the user answered are in
Decisions 1 and 2. Nothing is built yet.

The sidebar groups the sessions by repository today. See
[../tui/sessions.md](../tui/sessions.md). A control session that starts other
sessions with `create_session` makes a second kind of group, and the list does
not show it, because nothing records who created a session.

## Decisions

1. **The creator replaces the directory.** A session that a control session
   created is keyed on that creator, and not on its repository. The sidebar
   holds one flat list of groups: some named for a control session, the rest
   named for a directory.

   Rejected: a creator group inside a directory group. A control session often
   creates sessions in other repositories, so that creator would appear under
   two headers, and one group would no longer be one unit.

2. **The control session heads its own group.** The header is followed by the
   control session, then by every session it created. So the group reads as one
   unit, whatever repository each member works in.

3. **A control session with no children keeps its directory group.** The grant
   alone does not make a group. A group appears the moment the first
   `create_session` call returns.

4. **The group key is the name of the creator.** The name is unique, and it is
   the key on disk, so it survives a restart and a rename. The header shows the
   display name (the title when there is one) after a `⇄` mark, which already
   means "control" on a row. See [../tui/sessions.md](../tui/sessions.md).

5. **The keys of the two kinds of group are namespaced.** A directory group is
   `dir:<path>` and a creator group is `by:<name>`, so a name can never collide
   with a path.

6. **Inside a creator group the control session is always first.** Every other
   row keeps the order it has: live, then stored, then archived. Without this
   rule, a control session you stopped would sort below the children it made.

7. **A session records its creator once, and forever.** `Meta.parent` is
   written when the session is created, and a resume carries it, so a child
   still joins its group after a restart. A session that a human started has no
   creator, and there is nothing to backfill for the sessions on disk today.

## Open questions

None.

## The data flow

```
create_session (control session "boss")
        │  caller name
        ▼
bridge.Create(dir, name, by)                internal/manager/mcp.go
        │  Spec{Dir, Name, Parent: by}
        ▼
Manager.Spawn ──▶ entry.meta.Parent ──▶ <root>/sessions/<name>/meta.json
        │                                          │
        │  Parents() map[string]string             │  Stored()
        ▼                                          ▼
   live rows                                  stored rows
        └───────────────┬──────────────────────────┘
                        ▼
              row.parent  ──▶  refresh(): the group key
                        │
        parent is set    ├──▶ "by:<parent>"
        parent is empty  └──▶ "dir:" + repoRoot(dir)   (as today)
```

`Spawn` already writes `meta.json`, so one new field on `Meta` carries the
creator to disk with no new write path.

The interface reads the creator of a live row from a new
`Manager.Parents() map[string]string`, which mirrors `Grants()`. A stored row
reads `Meta.Parent` directly, as it reads `Meta.Control` today.

`refresh()` needs one pass before it keys the rows, to find which names have
children, because Decision 3 makes a creator group only when a child exists.

## The workflow

```
 ▾ ⇄ boss              3      a creator group: the mark, the name, the count
 ● boss                       the control session heads its own group
 ⠹ api
 ○ landing
 ▾ claude-multiplexer  2      a directory group, unchanged
 ● docs
 · old-notes
```

- Every key works as it does today. `z` folds a creator group, `Z` folds all
  but the one you are in, and a click on the header folds it. See
  [../tui/keys.md](../tui/keys.md).
- Stop the control session and the group stays, because its children stay.
- Archive a child and it leaves the list, as any row does.

## The build

**`internal/manager`:**

- `Spec.Parent string`, and `Meta.Parent string` with the JSON name
  `parent,omitempty`.
- `Spawn` copies `spec.Parent` into `entry.meta`.
- `Resume` and `ResumeWithEffort` carry `Parent`, so a resumed child keeps its
  group.
- `Parents() map[string]string`, next to `Grants()`.
- `bridge.Create` passes the caller as `Spec.Parent`.

**`internal/tui`:**

- `row.parent string`, filled by `rowFromSnapshot` (through `Parents()`) and by
  `rowFromMeta`.
- `refresh()` collects the names that have children, then keys each row:
  `by:<parent>` when the row has a creator, and `dir:<root>` otherwise. A
  control session with a child takes its own `by:<name>` key.
- `groupRows` sorts the control session first inside its own group.
- `groupHeader` shows `⇄` before the label of a creator group.
- `groupLabels` runs on the directory keys only, because a creator label is a
  session name.

**The documents, in the same change:**

- [../tui/sessions.md](../tui/sessions.md) — the two kinds of group, and the
  rules above.
- [../mcp.md](../mcp.md) — `create_session` records the caller as the creator.
- [../manager.md](../manager.md) — `Meta` holds the creator, and `Parents()`
  reports it for the live rows.

## How it is verified

| Test | What it holds |
|---|---|
| `manager` | `create_session` writes the caller into `meta.json` |
| `manager` | `Parents()` reports the creator of a live session |
| `manager` | A resume keeps the creator |
| `tui` | A created session sits in the group of its creator, not of its repository |
| `tui` | The control session is the first row of its own group |
| `tui` | A control session with no children stays in its directory group |
| `tui` | The header of a creator group shows the `⇄` mark and the title |
| `tui` | A creator group folds like any other |

The work is done in the `creator-groups` worktree, and `just check` is green
before it collapses.

## When it lands

Move the decisions into [../tui/sessions.md](../tui/sessions.md), and delete
this file. See [../../.claude/rules/plans.md](../../.claude/rules/plans.md).
