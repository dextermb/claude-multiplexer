# The tools a session can call

Each session is a Claude Code child, and the multiplexer serves it a small MCP
server named `cmux`. Through it, a session names itself, reads its neighbours,
and — with a grant — drives them.

Claude Code names an MCP tool `mcp__<server>__<tool>`, so every tool reaches the
model as `mcp__cmux__…`. The server carries the name `just install-as` gives the
binary, so one word names the program in a prompt and on the command line. A
session that starts writes its own `mcp.json`, so a resumed session takes the
name without a migration.

## The pages

| Page | Read it for |
|---|---|
| [mcp/tools.md](mcp/tools.md) | Every tool: its arguments, what it does, and the three it takes to change a setting |
| [mcp/grant.md](mcp/grant.md) | Which tools need the control grant, what a tool refuses, and two agents in a circle |
| [mcp/transport.md](mcp/transport.md) | The HTTP server, the token of a session, and what the installed Claude Code was proven to do |
| [mcp/notices.md](mcp/notices.md) | How a change with no session event behind it still reaches the screen |

For the stream between the multiplexer and one child, see
[protocol.md](./protocol.md). For the manager the tools call, see
[manager.md](./manager.md).
