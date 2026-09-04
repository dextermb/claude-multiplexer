# The transport

One HTTP server serves every session, on a free port on `127.0.0.1`. It starts
before the first session, and it stops with the manager.

Each session gets its own random token and its own tool set. At spawn the
manager writes the configuration file under the state directory:

```json
{
  "mcpServers": {
    "cmux": {
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
- The `init` event lists the server: `{"name":"cmux","status":"connected"}`. The
  decoder reads it as `protocol.Init.MCPServers`.
- It names the tool `mcp__cmux__rename_session` on `--allowedTools`, and calling
  one needs no permission prompt.
- The tools arrive as **deferred** tools. The model loads the schema with
  `ToolSearch` before its first call, which costs one extra step.
- The child continues normally after a tool returns. The turn ends with a
  `result` event, and the session goes back to `idle`.

`just probe-mcp` runs that check again. It starts one real session and it costs
money, so it is not part of `just check`.
