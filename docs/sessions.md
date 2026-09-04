# Sessions

A session is one child `claude` process, its state, and its transcript. The
package `internal/session` holds it. For the wire format, see
[protocol.md](./protocol.md).

## Configuration

`session.Config` names the child. `Dir`, `Model`, `PermissionMode`,
`AllowedTools`, `DisallowedTools`, `ResumeID`, and `SessionID` become command
line flags. `ClaudePath` selects the binary, which the tests point at a fake.
`TranscriptPath` turns on the transcript. An empty path turns it off.

Two flags change what the child sends back. `ReplayPrompts` adds
`--replay-user-messages`, so your prompt returns on stdout and reaches the
transcript. `IncludePartial` adds `--include-partial-messages`, so the text
arrives while the model writes it. The manager turns both on for every session
it supervises. See [protocol.md](./protocol.md).

The defaults are the permission mode `auto`, the binary `claude`, an event
buffer of 256, and 20 remembered stderr lines.

`New` rejects a directory that does not exist, because the failure is otherwise
a child that dies with no clear reason.

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
  [protocol.md](./protocol.md).
- `exited` — the child stopped in a clean way.
- `failed` — the child stopped with an error, or the stream broke.

`exited` and `failed` are terminal. A transition out of them is ignored, so the
first cause of a failure is the one that is reported.

The first prompt goes while the state is `starting`, because the `init` event
arrives only after the child reads input. Every later prompt waits for `idle`.

## The prompt queue

`Send` puts the text on a queue and returns at once. It fails only when the
session is not live. One writer goroutine takes the queue in order. For each
item, it first waits for a state that accepts a prompt, then takes the item,
moves the state to `busy`, and writes one line to stdin.

The wait comes before the take, so a queued prompt stays on the queue until the
session is truly idle. `DiscardQueued` can therefore drop it in time — see
[Interrupt](#interrupt).

So the order of the prompts is the order of the calls to `Send`, and a caller
never blocks. `Snapshot().Queued` reports the queue length.

## Goroutines

`Start` launches four goroutines:

| Goroutine | Work |
|---|---|
| `readStdout` | Decode each line, write it to the transcript, update the state, and publish the event. |
| `readStderr` | Keep the last 20 lines, and publish each one. |
| `writeLoop` | Drain the prompt queue into stdin. |
| `supervise` | Join the readers, wait for the child, set the final state, join the writer, and close the event channel. |

`supervise` closes the event channel exactly once, after every other goroutine
stops. A consumer therefore ends its loop when the channel closes.

## Events

The consumer receives one `session.Event` for each thing that happens:

| Kind | Meaning |
|---|---|
| `KindProtocol` | A decoded Claude Code event. |
| `KindState` | A state change, with the previous state. |
| `KindStderr` | One line from the child stderr. |
| `KindError` | A line that is not JSON, or a failure. |

`emit` waits for room in the buffered channel, or for the session context. A
consumer that stops reading therefore does not wedge the child for ever, because
`Stop` cancels the context when the grace period ends.

## Stop

`Stop` closes stdin and waits. A clean child sees the end of its input and
exits. If the given context ends first, `Stop` kills the process, cancels the
session context to release any pending event, and waits for the channel to
close. `DefaultStopGrace` is 5 seconds.

`Wait` blocks until the session ends, and returns the first error.

## Interrupt

`Interrupt` stops the running turn without stopping the session. It writes a
`control_request` with the subtype `interrupt` to stdin. See
[protocol.md](./protocol.md). Claude Code answers with a `control_response`,
ends the turn with a `result` event, and stays alive for the next prompt. This
was proven against Claude Code 2.1.176.

The `result` event moves the state from `busy` to `idle`, the same as a normal
end of turn. So the writer wakes, and it sends the next queued prompt at once.
`Interrupt` writes only while the state is `busy`, and does nothing otherwise.

The session also interrupts itself when the model asks a question. That
interrupt ends the turn in `waiting`, not `idle`. See
[protocol.md](./protocol.md).

`DiscardQueued` clears the prompt queue. Call it before `Interrupt` to stop the
turn and hold the session, because the writer waits for `idle` before it takes
an item, so the cleared queue leaves nothing to send. Call `Interrupt` alone to
end the turn and let the next queued prompt go at once.

## The transcript

Every raw stdout line goes to `TranscriptPath` as JSON Lines, before the decoder
result reaches anyone. Memory holds only the counters and the last stderr lines.

There is one exception. A `stream_event` line is not written, because it repeats
text that the `assistant` line already holds. See [protocol.md](./protocol.md).
So the file is the complete history of the conversation, and not of the wire.

The counters are derived, and not stored. `Snapshot` reports the state, the
Claude session identifier, the accumulated cost, the number of turns, the
duration of the last turn, the accumulated tokens, and the queue length.

`Snapshot.Model` and `Snapshot.PermissionMode` report what the child confirms.
They start as the flag values, and the `init` event replaces them. So an empty
`--model` becomes the model Claude Code chose, and a mode the child changes is
the mode the interface shows.

## Background jobs

A session derives its background jobs from the stream, the same way it derives
the cost and the token counts. Claude Code pushes three task events for each
job; see [protocol.md](./protocol.md). `apply` turns them into `Job` records:

- `task_started` adds a `Job` with the status `running`.
- `task_updated` finds the `Job` by its id, and sets the status from the patch.
- `task_notification` finds the `Job` by its id, and stops it.

A `Job` holds the id, the description, the task type, the status, and the start
and end times. The status is one of `running`, `done`, `failed`, or `killed`.
The start event makes a job `running`. A later event moves it to one terminal
status:

| Wire value | Job status |
|---|---|
| `completed` | done |
| `failed` | failed |
| `killed`, `stopped` | killed |
| any other | running |

An unknown status stays `running`, so a new Claude Code value cannot lose a job.
A notification does not change a job that already reached a terminal status.

`Snapshot.Jobs` carries every job in start order, for the life of the session.
`Snapshot.RunningJobs` counts the jobs that still run. The interface shows both;
see [tui/sessions.md](./tui/sessions.md).

## Tests without a network

`internal/testutil/fakeclaude` is a small program that answers like Claude Code.
The tests build it, and they point `ClaudePath` at it. `FAKECLAUDE_MODE` selects
the behaviour:

| Mode | Behaviour |
|---|---|
| unset | Emit `init`, then echo each prompt with an `assistant` and a `result` event. |
| `lazyinit` | Emit `init` only after the first prompt, as the real binary does. |
| `crash` | Write to stderr, and exit 1. |
| `garbage` | Write a line that is not JSON, and then work as normal. |
| `exit-after-init` | Emit `init`, and exit 0. |
| `noinit` | Exit 0 at once. |
| `bigline` | Emit a very long line. |

`FAKECLAUDE_ARGS_FILE` records the working directory and the arguments, so a
test can check what the child received.
