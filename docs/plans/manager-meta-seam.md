# Candidate 01 — one `mutateMeta` seam for the manager

Status: **in progress**. The seam, the callsite list, and the stored-session
path are decided.

Effort: `internal/manager`. Worktree `manager-meta-seam`.

## Problem

Every method that changes a live session's record repeats the same four steps by
hand:

```
item := entry(name)          // lock m.mu, find entry, unlock
meta := item.metaCopy()      // lock metaMu, copy, unlock
mutate the copy
item.setMeta(meta)           // lock metaMu, set, unlock
writeMeta(item.path, meta)   // file write, no lock
```

Two faults follow from the released lock:

1. **The read-modify-write is not atomic.** Between `metaCopy` and `setMeta` the
   `metaMu` is free, so the pump's `rememberSession` can commit turn totals in
   the gap. One writer then overwrites the other. A lock or a project change can
   lose a turn's cost, or the reverse.
2. **The file write is not ordered with the memory commit.** Two writers can
   reach `writeMeta` in an order that disagrees with memory, so the file on disk
   ends behind the record in memory. A restart then reads the stale file.

`metaCopy`/`setMeta`/`writeMeta` appear at ~13 sites (see the grep in the effort
notes). The pattern earns its keep, but it is copied, and each copy re-derives
the lock discipline by hand.

## Decision — the seam

Add one method on `*entry` that does the whole read-modify-write and the persist
under `metaMu`, and returns the new record:

```go
// mutateMeta applies fn to the record under metaMu, persists it once, and
// returns the new copy. It holds metaMu across the write, so the change is
// atomic against the pump. See docs/manager.md.
func (e *entry) mutateMeta(fn func(*Meta) error) (Meta, error) {
    e.metaMu.Lock()
    defer e.metaMu.Unlock()
    next := e.meta
    if err := fn(&next); err != nil {
        return Meta{}, err
    }
    if err := writeMeta(e.path, next); err != nil {
        return Meta{}, err
    }
    e.meta = next
    return next, nil
}
```

Two properties, both new:

- **Write-then-commit.** The memory field takes the change only after the file
  write succeeds, so memory never leads the disk. Today a failed `writeMeta`
  leaves memory ahead of the file. The new order is strictly safer.
- **One lock for the field and the file.** `metaMu` is held across the file
  write, so the memory commit and the file write cannot reorder against another
  writer.

### Rejected: a second write mutex

A separate `writeMu` per entry would let the field commit under `metaMu` and the
file write serialise under `writeMu`, so IO never blocks a `metaCopy` read. It
reintroduces a lock-ordering rule to get right at every site, which is the fault
the seam removes. `writeMeta` writes a small JSON file once per change, not per
render, so the brief IO under `metaMu` is acceptable. One lock, held across both.

## Decision — validation stays outside the lock

`SetWorkingDir` and the project methods call `resolveDir`, which runs `os.Stat`.
`meta.Dir` is fixed once a session starts, so the path resolves outside the lock
from `item.metaCopy().Dir`, and only the pure list change runs inside `mutateMeta`:

```go
full, err := resolveDir(item.metaCopy().Dir, path)   // IO outside the lock
if err != nil { return nil, err }
_, err = item.mutateMeta(func(meta *Meta) error {
    meta.WorkingDirs = appendUnique(meta.WorkingDirs, full)
    return nil
})
```

The list read and the list write now sit in one critical section, so the
add/remove RMW is atomic too — a fault the current `metaCopy`-then-`writeProject`
shape also has.

## Callsites to convert (live entry)

| Method | File | Now |
|---|---|---|
| `SetWorkingDir` / `UnsetWorkingDir` | lifecycle.go:228–263 | metaCopy → mutate → setMeta → writeMeta |
| `SetProject` / `AddProjectDir` / `RemoveProjectDir` / `ClearProject` | lifecycle.go:295–367 | via `writeProject` helper |
| `SetLocks` / `AddLock` / `RemoveLock` / `ClearLocks` | locks.go:29–125 | via `writeLocks` helper |
| `mutateSessionLayout` (live branch) | layout.go:202–206 | metaCopy → mutate → setMeta → writeMeta |
| `rememberSession` | store.go:14–41 | commits under lock, then writes **outside** it |

`writeProject` and `writeLocks` collapse into the mutate function of their
callers. `rememberSession` keeps its "skip when unchanged" fast path, but the
`writeMeta` moves inside the locked section so it cannot reorder against a tool
writer.

## Decision — the stored-session path

`SetTitle` (lifecycle.go:398–404), `mutateSessionLayout`'s stored branch
(layout.go:208–214), and `Archive` (store.go:134–147) change the meta of a
session that is **not** live, by `ReadMeta(path)` → mutate → `writeMeta(path)`.
No pump runs for a stored session, so there is no concurrent writer and no race.

Add a parallel helper so the two paths read alike:

```go
// mutateStoredMeta reads the record at path, applies fn, and writes it back. A
// stored session has no entry and no pump, so no lock is needed.
func mutateStoredMeta(path string, fn func(*Meta) error) error {
    meta, err := ReadMeta(path)
    if err != nil {
        return err
    }
    if err := fn(&meta); err != nil {
        return err
    }
    return writeMeta(path, meta)
}
```

It removes three more copies of the ReadMeta → mutate → writeMeta shape. It
guards no concurrency, so it is a shape helper, not a seam. The two together give
every meta change one of two forms: `entry.mutateMeta` for a live session,
`mutateStoredMeta` for a stored one.

## Verification

- `go test ./internal/manager/...` stays green.
- Add a race test: spawn a session against fakeclaude, then run `AddLock` and a
  forced `rememberSession` (or `Send` that drives a turn) at the same time under
  `-race`, and assert neither the lock nor the turn total is lost.
- `go test -race ./internal/manager/...`.

## When it lands

Move the durable part into `docs/manager.md`, "Two mutexes, for two things": the
record now changes only through `mutateMeta`, which holds `metaMu` across the
read, the write to disk, and the memory commit, in that order. Delete this plan.
