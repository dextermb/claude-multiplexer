# The lock tools

A lock is a label a session holds, to say it works on something and another
session must keep away. The locks live in the metadata of a session, so they
survive a restart. See [../../manager.md](../../manager.md).

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `list_locks` | `session` | The locks a session holds, in the order it took them. An empty session means the caller. | open |
| `add_lock` | `lock` | Takes one lock on the caller, and names the other sessions that hold the same label. | open |
| `remove_lock` | `lock` | Releases one lock of the caller. | open |
| `set_locks` | `locks` | Replaces the whole set of locks of the caller. An empty list releases every lock. | open |
| `clear_locks` | — | Releases every lock of the caller. | open |
| `find_locked_sessions` | `locks`, `live` | The sessions that hold every named lock. | open |

### What a lock is

A lock is a plain string. The multiplexer does not read it, so the sessions
agree on the labels themselves. These are typical labels:

- `repo:claude-multiplexer` — the whole code base.
- `file:internal/manager/meta.go` — one file.
- `deploy` — a shared action that must run alone.

The label keeps its case, because a path is case-sensitive. The multiplexer cuts
only the outer space. An empty label is an error.

The locks of a session are a set, in the order the session took them. A repeat
`add_lock` changes nothing, and it is not an error.

### A lock is advisory

`add_lock` always succeeds. It never fails because another session holds the
same label. The result carries `holders`, which names the other sessions that
hold it, and the message says so in words.

The multiplexer does not stop the work, because it cannot see what the work is.
The model decides. So the pattern is a search, then the lock:

1. Call `find_locked_sessions` with the label.
2. If the answer is empty, call `add_lock` and start the work.
3. If the answer names a session, do not start. Report the holder to the human.
4. Call `remove_lock` when the work is complete.

### A lock is never released on its own

A lock stays until `remove_lock` or `clear_locks` takes it off. A session that
stops keeps every lock it held, and a search still finds it.

This is deliberate. A lock that dies with the session cannot protect work that
crosses a restart. The cost is a stale lock, so call `clear_locks` before a
session ends.

`find_locked_sessions` takes `live` for this. With `live` true it returns only
the sessions the multiplexer runs now, so a stale lock of a stopped session is
left out. With `live` false, the default, it returns every session that holds
the label.

### The search

`find_locked_sessions` takes a list, and a session matches only when it holds
**every** label in it. So one label is a broad search, and several labels are a
narrow one.

It answers with the same rows as `list_sessions`, and each row carries its
`locks`. A search with no label is an error, because it would return every
session.

A session that runs on a peer carries no metadata on this host, so it never
matches. The locks are local to one host. See [../../peers.md](../../peers.md).
