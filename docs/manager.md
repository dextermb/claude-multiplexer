# The manager

The manager owns every live session, and it is the only thing the interface
talks to. The package `internal/manager` holds it. For one session, see
[sessions.md](./sessions.md).

## What it holds

For each session the manager keeps three things:

- the `session.Session` itself,
- a ring buffer of rendered output lines, 5000 by default,
- a `Meta` record on disk.

A pump goroutine reads the event channel of each session. For each event it
renders the lines, appends them to the ring buffer, updates `meta.json` when the
Claude session identifier changes, and publishes one `manager.Event` on the bus.

So the manager renders once, and every subscriber gets the same lines. A late
subscriber reads the buffer with `Lines(name)` and misses nothing.

A rendered line carries its class, so the interface can colour it. See
[tui.md](./tui.md) for the classes and their colours.

## What is kept between runs

A session that finishes at least one turn is written to
`<root>/sessions/<name>/meta.json`. A session that finishes none leaves nothing:
its directory is removed when it ends. So a mistake, or a start that failed,
never reaches the list you see tomorrow.

`Meta` holds the name, the title, the directory, the model, the permission mode,
the Claude session identifier, the lifetime turn count and cost, the tokens, the
control grant, and the archive flag. The counts are lifetime totals: a resumed
session adds to them, and does not restart them.

| Method | What it does |
|---|---|
| `Stored()` | Every session on disk that the manager does not currently hold, newest first. |
| `Replay(name)` | The transcript of a stored session, rendered, capped to the line limit. |
| `Resume(ctx, meta)` | Start a child with `--resume`, under the same name and the same transcript. |
| `Archive(name, on)` | Set or clear the archive flag. It refuses a session that still runs. |
| `SetTitle(name, title)` | Rename a session. A live session takes it at once; a stored one gets it written to its meta. |
| `Meta(name)` | The record for one session. |
| `Messages(name, limit)` | The recent conversation, read from the transcript. It answers for a stored session too. |
| `List()` | Every session, live and stored, in the shape the tools return. |
| `Grants()` | Which live sessions may drive their neighbours. |

A resumed session starts with its past output already in the line buffer,
followed by a `— resumed —` marker. So the pane reads as one conversation, and
not as a fresh window on an old subject.

**A new session never takes a stored name.** `Spawn` checks the live list and
the directory on disk, and appends `-2` when either holds the name. Only a
resume keeps the exact name, because a resume means to write to that transcript.

## Text that has not settled

While the model writes, the child sends pieces of the answer. The manager adds
each piece to a buffer for that session, and every event it publishes carries
the **whole** buffer so far in `Event.Partial`.

Cumulative text, and not the piece, is what makes a drop harmless. A subscriber
that misses ten events still receives the newest whole text in the next one.

The buffer clears when the `assistant` message or the `result` arrives, because
the settled text then reaches the line buffer as a normal line. A subscriber
therefore sees the text grow, and then sees it replaced once by the finished
message.

## Names and titles

A session name must be unique, because it is the key everywhere. `Spawn` takes
the wanted name, or the last element of the directory when the name is empty.
When that name is taken, it appends `-2`, then `-3`, and so on.

A title is separate from the name. A title is display text, set by `SetTitle`,
and it does not need to be unique. The interface shows the title when it is set,
and the name when it is not. The name stays fixed, so a rename moves no files and
rekeys nothing.

The state of each session lives under `<root>/sessions/<name>/`, which holds
`transcript.jsonl` and `meta.json`. The root is `~/.multiplexier` by default.

## The bus, and what a drop means

Each subscriber gets a buffered channel of 256 events. When that channel is
full, the bus drops the **oldest** event for that subscriber, counts the drop,
and delivers the new one. A slow reader therefore falls behind, but it never
blocks a session, and it always holds the newest state.

Every event carries a sequence number from one counter. The numbers are
contiguous for a subscriber that keeps up. A gap means the bus dropped
something, so the reader must rebuild its view from `Lines(name)` instead of
appending. The terminal interface does exactly this.

A drop costs screen updates only. The transcript on disk stays complete, and the
ring buffer in memory stays complete up to its 5000 lines.

## The snapshot stays with the lines

`Snapshot(name)` reports the state of a session. The turn count, the cost, the
duration, and the token totals come from the stream. The pump adds each of these
only after it appends that event's lines. So the count never leads the buffer:
when `Snapshot` shows five turns, `Lines` already holds those five turns.

Each session event carries the snapshot as of that event, because the session
reads the stream on one goroutine and the pump appends the lines on another. The
session pins the snapshot when it sends the event, and the pump caches that
snapshot when it appends the lines. So the two clocks agree.

The title, the queue length, and the state come live from the session, because a
direct action changes them with no stream event behind it. So a new title or a
queued prompt shows at once, and does not wait for the next turn.

## The tools a session can call

The manager runs one MCP server for every session, and `StartMCP` starts it.
Each session gets a token, a configuration file at
`<root>/sessions/<name>/mcp.json`, and the tool names on `--allowedTools`. The
manager holds the map from token to session, and it drops the token when the
session ends. The full tool set is in [mcp.md](./mcp.md).

`Spec.Control` decides whether a session may drive its neighbours. It is stored
in `Meta`, so a resume keeps it, and `Grants()` reports it for the live rows.

`SendFrom(target, from, text)` is `Send` with provenance. It marks the pane of
the target with `← prompt from <from>` before it queues the prompt, so the human
sees which agent asked. The keys use `Send`, which adds no mark.

A change that a tool makes to `meta.json` has no session event behind it. The
manager therefore publishes an event carrying `Notice` and `Reload`, and the
interface acts on that instead of a timer. See [mcp.md](./mcp.md).

## Control

`Send(name, text)` puts a prompt on the session queue. `Stop(ctx, name)` ends a
session. `Interrupt(name, discardQueued)` stops the running turn without
stopping the session. When `discardQueued` is true, it clears the prompt queue
first, so the session stops until the next prompt. When false, it ends the turn
and lets the next queued prompt go at once. See
[Interrupt](./sessions.md#interrupt).

## Shutdown

A session that ends stays in the list, so you can still read it. It becomes a
stored row on the next run, or as soon as you archive it.

`Shutdown` stops every session at the same time, each with the same context, and
then waits for every pump to finish. The interface calls it before it quits, so
no child process is left behind.

## Events

```go
type Event struct {
    Seq      uint64            // the position in the one global order
    Session  string            // the session name
    Kind     session.EventKind // protocol, state, stderr, or error
    Lines    []render.Line     // the rendered output, with a class for each line
    Snapshot session.Snapshot  // the state of the session after the event
    Closed   bool              // the session ended, and no more events follow
    Notice   string            // a change made outside the session stream
    Reload   bool              // the stored list changed, so read it again
}
```
