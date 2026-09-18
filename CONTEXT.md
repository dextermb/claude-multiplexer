# Domain model

This file names the concepts of the multiplexer, so the code and the docs use
one word for one thing. See `docs/` for how each concept works.

## Session ports

The `internal/mcp` package reaches the manager through `Sessions`, a composite
interface. `Sessions` embeds one **port** per concept, so a tool group and its
test depend on the port they need, not on the whole session surface. The manager
satisfies every port. See `internal/mcp/sessions.go`.

| Port | What it covers |
|---|---|
| `SessionReader` | Read the shape of a session: the list, the messages, the jobs. |
| `ControlPort` | Drive a session: send, stop, archive, create, stop a job. A control grant gates it. |
| `ConfigPort` | Read and change the settings and the directories of a session. |
| `LockPort` | Read and change the advisory locks a session holds. See [docs/mcp/tools/locks.md](docs/mcp/tools/locks.md). |
| `LayoutPort` | Read and change the saved screen layouts. See [docs/tui/layouts.md](docs/tui/layouts.md). |
| `SchedulePort` | Read and change the scheduled runs. See [docs/scheduler.md](docs/scheduler.md). |
| `UsagePort` | Read the token and cost usage, local and on the peers. See [docs/cost.md](docs/cost.md). |
| `APIPort` | Mint and revoke the credentials of the session API. See [docs/mcp/api.md](docs/mcp/api.md). |
| `PeerPort` | Read and change the peers and the reserve gate. See [docs/peers.md](docs/peers.md). |
| `SharePort` | Mint, list, revoke, and watch the read-only shares. See [docs/peers.md](docs/peers.md). |

A tool test builds a fake of one port, and embeds `Sessions` for the rest, so it
fills the six methods of its port rather than the whole surface. The lock tests
show the pattern: see `lockPortFake` in `internal/mcp/tools_locks_test.go`.
