# Session lifecycle

This page covers the state of a session and the transitions between states:
the state machine, stop, rename, stop when idle, auto-archive, and interrupt.

## The state machine

```
                 spawn
   (none) ---------------> starting
                              |
                              |  first prompt, or system/init
                              v
        +------------------ idle <------------------+
        |                     |                     |
   send |                     | child exits         | result
        v                     v                     |
      busy ----------------> exited                 |
        |  \                                        |
        |   \ question: interrupt, then result      |
        |    v                                      |
        |  waiting ---------------------------------+
        |    ^  |
        |    +--+  send: the answer moves waiting to busy
        |
        | decode failure, write failure, or a non-zero exit code
        v
      failed
```

- `starting` — the process runs, and no prompt has gone yet.
- `idle` — the session accepts a prompt.
- `busy` — a turn is in progress. A further prompt waits in the queue.
- `waiting` — the model asked a question. The session interrupted the turn, and
  it waits for the human answer. It accepts a prompt, the same as `idle`. See
  [protocol.md](../protocol.md).
- `exited` — the child stopped in a clean way.
- `failed` — the child stopped with an error, or the stream broke.

`exited` and `failed` are terminal. A transition out of them is ignored, so the
first cause of a failure is the one that is reported.

The first prompt goes while the state is `starting`, because the `init` event
arrives only after the child reads input. Every later prompt waits for `idle`.

## Stop

`Stop` closes stdin and waits. A clean child sees the end of its input and
exits. If the given context ends first, `Stop` kills the process, cancels the
session context to release any pending event, and waits for the channel to
close. `DefaultStopGrace` is 5 seconds.

`Wait` blocks until the session ends, and returns the first error.

## Rename

A rename sets the display title of a session. The manager routes the rename by
the state of the session.

A running session takes the title in memory, and its pump persists the title to
the meta on the next event. A session whose child exited gets the title written
straight to the meta, because the pump is gone and the events channel is closed.
An exited session lingers as an entry until a resume or a remove, so this second
path holds for both a lingering entry and a stored session.

A rename of an exited session does not emit an event, because a send on the
closed events channel panics. The `emit` function also returns early when the
session context is done, as a second guard.

## Stop when idle

A session arms a deferred action on itself with the `stop_when_idle` tool. The
action is a stop, and an archive after the stop when the caller asks for it. A
scheduled run is the main user: a prompt does its work, then arms the action, so
a spawn-mode run leaves no exited session behind. See
[mcp/tools/sessions.md](../mcp/tools/sessions.md) and [scheduler.md](../scheduler.md).

The tool cannot stop the session in the same turn, because the stop kills the
process that runs the tool call. So the manager defers the action. The
per-session pump watches every state event, and it starts the action once three
conditions hold:

- the state is `idle`, so the turn is complete and the model did not ask a
  question,
- the queue is empty, so no prompt waits, and
- the session ran at least one turn, so a fresh session does not act before its
  first prompt.

A queued prompt keeps the session busy, so the arm stays set until every prompt
is complete. Then the session goes idle, and the action runs once. The manager
runs the stop in its own goroutine, not in the pump, because the pump must keep
draining events for the session to exit. The arm lives in memory on the live
session, so a restart does not carry it, the same as the session itself.

A reuse-mode schedule works the same way. The stop ends the session after the
turn, and the next fire resumes it from its Claude session id, so it keeps its
memory. An archive clears on the resume, so the reuse session comes back.

## Auto-archive stopped sessions

The `autoArchiveDays` setting archives a stopped session on its own, after it is
idle for that many days. A sweep runs every hour. It reads the setting each time,
so a change takes effect on the next sweep with no restart. A nil setting, or a
value below 1, archives nothing. See [config.md](../config.md).

The sweep archives a session only when three conditions hold:

- the session is stopped, so it is not live,
- the session is not archived yet, and
- `now - last_active_at` is more than the set days, from the last turn time in
  the meta.

A session sets the value with the `set_auto_archive` tool, and clears it with
`unset_auto_archive`. The `set_config` tool reaches the same field by the path
`autoArchiveDays`. See [mcp/tools/settings.md](../mcp/tools/settings.md).

A raised value does not bring a session back: an archived session stays
archived, the same as a manual archive. The human clears the archive by hand.

## Interrupt

`Interrupt` stops the running turn without stopping the session. It writes a
`control_request` with the subtype `interrupt` to stdin. See
[protocol.md](../protocol.md). Claude Code answers with a `control_response`,
ends the turn with a `result` event, and stays alive for the next prompt. This
was proven against Claude Code 2.1.176.

The `result` event moves the state from `busy` to `idle`, the same as a normal
end of turn. So the writer wakes, and it sends the next queued prompt at once.
`Interrupt` writes only while the state is `busy`, and does nothing otherwise.

The session also interrupts itself when the model asks a question. That
interrupt ends the turn in `waiting`, not `idle`. See
[protocol.md](../protocol.md).

`DiscardQueued` clears the prompt queue. Call it before `Interrupt` to stop the
turn and hold the session, because the writer waits for `idle` before it takes
an item, so the cleared queue leaves nothing to send. Call `Interrupt` alone to
end the turn and let the next queued prompt go at once.
