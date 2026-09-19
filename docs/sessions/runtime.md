# How a session runs

This page covers the moving parts of a live session: the prompt queue, the
goroutines, and the events they publish.

## The prompt queue

`Send` puts the text on a queue and returns at once. It fails only when the
session is not live. One writer goroutine takes the queue in order. For each
item, it first waits for a state that accepts a prompt, then takes the item,
moves the state to `busy`, and writes one line to stdin.

The wait comes before the take, so a queued prompt stays on the queue until the
session is truly idle. `DiscardQueued` can therefore drop it in time — see
[lifecycle.md](./lifecycle.md).

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

Every event carries the snapshot of the session as of that event, taken when
`emit` sends it. The manager caches this snapshot beside the lines, so the turn
count never leads the buffer. See [manager.md](../manager.md).
