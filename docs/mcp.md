# The tools a session can call

Each session is a Claude Code child, and the multiplexer serves it a small MCP
server named `mux`. Through it, a session names itself, reads its neighbours,
and — with a grant — drives them.

For the stream between the multiplexer and one child, see
[protocol.md](./protocol.md). For the manager the tools call, see
[manager.md](./manager.md).

## The tools

Claude Code names an MCP tool `mcp__<server>__<tool>`, so every tool below
reaches the model as `mcp__mux__…`.

| Tool | Arguments | What it does | Grant |
|---|---|---|---|
| `rename_session` | `title` | Sets the title of the calling session. An empty title clears it. | open |
| `list_sessions` | `live_only` | Every session, running and stored: name, title, directory, state, model, turns, cost, and the archive flag. | open |
| `get_messages` | `session`, `limit` | The recent messages of a session, oldest last. 20 by default, 200 at most. | open |
| `send_message` | `session`, `text` | Queues a prompt for another session, and returns the queue length. | control |
| `stop_session` | `session` | Ends another child in a clean way. Its transcript is kept. | control |
| `archive_session` | `session`, `restore` | Takes a stopped session out of the list, or with `restore` brings it back. | control |

`get_messages` reads the transcript on disk, so it answers for a stored session
as well as a live one. It returns one entry for each user prompt, each assistant
message, and each result. A tool call becomes the short line `[used Bash]`,
because the whole input of a tool call is large and it is rarely what a reader
wants.

## The grant

A session gets `rename_session`, `list_sessions`, and `get_messages` always.
It gets the three control tools **only** when it is started with control.

- In the new session form, set the `Control` field to `yes`.
- On the command line, `multiplexier --dir <path> --control`.

The grant is stored in `meta.json`, so a resumed session keeps what it had.

A session without the grant is never offered the control tools. They are absent
from its `tools/list` answer, so the model does not know they exist, and it
cannot call one by name.

**A session with control can stop the work you are reading.** The interface
marks such a row with `⇄` in the sidebar, and the session bar names `control`
next to the model. See [tui/sessions.md](./tui/sessions.md).

## What a tool refuses

- An unknown token. The request never reaches a tool.
- An unknown session name.
- `send_message` to the calling session itself. That prompt would run after the
  turn that sent it, which is a loop with no human in it.
- `stop_session` on the calling session itself. The child would die before the
  tool result returned.
- `archive_session` on a session that still runs. Stop it first.

## Two agents can talk in a circle

`send_message` returns as soon as the prompt is on the queue. It does not wait
for the answer, so no agent blocks on another and two agents cannot deadlock.

They can, however, trade prompts for as long as they have money. Nothing stops
session A prompting B while B prompts A. The multiplexer does not guard against
this. It makes the traffic visible instead: every prompt that came from another
session is marked in the pane of the session that received it.

```
← prompt from docs
› list the files in the repository
```

Watch for that line, and press `x` to interrupt a session that runs away. See
[tui/keys.md](./tui/keys.md).

## The transport

One HTTP server serves every session, on a free port on `127.0.0.1`. It starts
before the first session, and it stops with the manager.

Each session gets its own random token and its own tool set. At spawn the
manager writes the configuration file under the state directory:

```json
{
  "mcpServers": {
    "mux": {
      "type": "http",
      "url": "http://127.0.0.1:52413/mcp",
      "headers": { "Authorization": "Bearer 4f3a…" }
    }
  }
}
```

The child is then started with `--mcp-config <that file>`, and with the tool
names on `--allowedTools`, so a call needs no key press.

```
<root>/sessions/<name>/mcp.json     the configuration above
```

The token is the identity of the session. The server holds one map from token
to tool set, and the tool set carries the session name, so a handler always
knows who called it and never has to trust an argument. A request with no token,
or with a token the server does not hold, is answered `401` before it reaches
any tool.

The manager drops the token when the session ends.

`--mcp-config` **adds** to the MCP servers the user already configured. The
multiplexer does not pass `--strict-mcp-config`, so a session keeps the servers
its project and its settings give it.

## What the installed Claude Code does

Proven against Claude Code 2.1.176:

- It accepts `--mcp-config <file>` with an `http` entry and a `headers` object,
  and it sends those headers on every request.
- The `init` event lists the server: `{"name":"mux","status":"connected"}`. The
  decoder reads it as `protocol.Init.MCPServers`.
- It names the tool `mcp__mux__rename_session` on `--allowedTools`, and calling
  one needs no permission prompt.
- The tools arrive as **deferred** tools. The model loads the schema with
  `ToolSearch` before its first call, which costs one extra step.
- The child continues normally after a tool returns. The turn ends with a
  `result` event, and the session goes back to `idle`.

`just probe-mcp` runs that check again. It starts one real session and it costs
money, so it is not part of `just check`.

## How a change reaches the screen

A tool that changes a live session moves the session itself, so the interface
learns of it the way it learns of everything else: the session emits an event,
and the bus carries it.

Two changes have no session event behind them, because both are only a write to
`meta.json`: archiving a stored session, and renaming one. For these the manager
publishes an event of its own, carrying `Notice` and `Reload`:

```
archive_session          Manager.Archive writes meta.json
                              |
                              v
                         Bus.Publish{Session:"landing", Reload:true,
                                     Notice:"docs archived landing"}
                              |
                              v
                         the interface reads the stored list again,
                         and shows the notice in the status bar
```

So the screen is true within one event, and the interface polls nothing.

**A notice event carries no output and no streaming text.** The interface must
therefore keep it off the normal path, which treats empty streaming text as the
end of an answer and clears the pane. See `handleNotice` in
`internal/tui/app.go`.
