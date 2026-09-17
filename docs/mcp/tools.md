# Every tool

Each tool reaches the model as `mcp__cmux__<tool>`. A tool marked `open` goes to
every session, and a tool marked `control` goes only to a session that holds the
grant. See [grant.md](grant.md).

Each page below carries the table of its own tools, with the arguments, what the
tool does, and the grant it needs.

| Page | Read it for |
|---|---|
| [tools/sessions.md](tools/sessions.md) | Naming, listing, and reading a session, and the tools that start, stop, and prompt one |
| [tools/settings.md](tools/settings.md) | The settings file: the editor, the block cap, auto-archive, a key by a dot path, and the paths a session reads |
| [tools/directories.md](tools/directories.md) | The working directory of a session, and the project that spans several code bases |
| [tools/locks.md](tools/locks.md) | The locks a session holds, so two sessions do not work on the same thing |
| [tools/layouts.md](tools/layouts.md) | The named interface dimensions, and the scope a layout is active in |
| [tools/schedules.md](tools/schedules.md) | The durable, recurring tasks, and the one field that needs the control grant |

For the API a program outside the multiplexer reaches, see [api.md](api.md). For
the tools that report cost, and for the peer and share tools, see
[../peers.md](../peers.md) and [../cost.md](../cost.md).
