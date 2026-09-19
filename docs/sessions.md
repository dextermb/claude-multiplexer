# Sessions

A session is one child `claude` process, its state, and its transcript. The
package `internal/session` holds it. For the wire format, see
[protocol.md](./protocol.md).

This page is a map. Each subject has its own page:

- [config.md](sessions/config.md) — the `session.Config` flags, the defaults,
  and the checks `New` makes. Read this to set up a child.
- [lifecycle.md](sessions/lifecycle.md) — the state machine and the transitions:
  stop, rename, stop when idle, auto-archive, and interrupt. Read this to see
  how a session starts, stops, and changes state.
- [runtime.md](sessions/runtime.md) — the prompt queue, the four goroutines, and
  the events they publish. Read this for the moving parts of a live session.
- [transcript.md](sessions/transcript.md) — the transcript file, and the derived
  counters `Snapshot` reports. Read this for what a session records.
- [context.md](sessions/context.md) — the context governor, which raises a
  notice as the context fills and holds the session on request.
- [jobs.md](sessions/jobs.md) — the background jobs Claude Code starts, which the
  session tracks and the interface lists.
- [testing.md](sessions/testing.md) — `fakeclaude`, the program the tests run in
  place of Claude Code, and its modes.
