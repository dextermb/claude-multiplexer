# The prompt cache of a session

A session sends its whole prompt on every turn. The prompt cache makes the
repeated part cheap, so what a turn costs depends less on how long the prompt is
than on how much of it is unchanged.

The multiplexer does not set the cache. It starts `claude` child processes, and
Claude Code builds the request. So this page records what Claude Code does, and
what the multiplexer can and cannot change. The measurements are against Claude
Code 2.1.231.

## The three prices

A prompt splits into three parts, and each has its own price:

| Part | The price, against the base input rate |
|---|---|
| A cache read | About 0.1x |
| A cache write, at the five-minute lifetime | 1.25x |
| A cache write, at the one-hour lifetime | 2x |
| Fresh input | 1x |

So a five-minute entry pays for itself on the second request, and a one-hour
entry needs three. The session bar shows the share that comes from the cache.
See [tui/sessions/bars.md](tui/sessions/bars.md).

## The lifetime is one hour

Claude Code picks the lifetime from the source of the request, against an
allowlist of `repl_main_thread*`, `sdk`, `auto_mode`, and `memdir_relevance`. A
session of the multiplexer runs the main loop, so it takes the one hour.

Three things turn the choice back to five minutes:

- an API key in place of a subscription login,
- an account in usage overage,
- the environment variable `FORCE_PROMPT_CACHING_5M`.

The variable `ENABLE_PROMPT_CACHING_1H` forces the hour. There is no longer
lifetime, so a session that goes quiet for more than an hour writes its prefix
again on the next turn, whatever else is true.

## The cache does not belong to the session

The cache is server-side, and it keys on the bytes of the prefix. It does not
key on the session, the process, or the conversation. Two fresh sessions started
one after the other in one directory read 18465 and 22421 tokens from the cache
on their first turn.

So keeping a session process alive buys nothing the cache was not already
giving. A schedule that starts a fresh session each time still reads most of its
prefix, as long as that prefix is the same bytes as last time.

## What invalidates it

A prompt renders in the order tools, then system, then messages, and the cache
matches a prefix of that render. A change at any point invalidates everything
after it, so the earlier the change, the more it costs.

The tool list sits first. That makes the set of connected MCP servers the most
expensive thing to change, and it changes on its own: a remote server that
answers slowly joins one session and misses the next. Two sessions started
seconds apart reported 280 and 344 tools, because one server answered in time
for the second only, and the later session wrote 13798 tokens to the cache.

Nothing in the multiplexer can order a remote server to answer on time. Connect
fewer servers, or accept the write.

## Claude Code defers the tool schemas

Claude Code carries its own tool search, and it is on by default. It marks the
tools of every MCP server for deferred loading, so their schemas stay out of the
prompt until the model asks for one. A run on a machine with 14 MCP servers
reported:

```
Dynamic tool loading: 0/332 deferred tools included
```

Two things follow. The first is that a large tool count costs far less than it
looks, because the schemas are not in the prefix. The second is that the
multiplexer must not build a tool search of its own, because the layer above
already has one, and only that layer can mark a tool as deferred.

Tool search turns off for a HIPAA workspace, for the environment variable
`CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS`, and for a proxy that is not a
first-party Anthropic host. Read the decision from the debug log:

```
claude --debug --debug-file /tmp/ts.log -p "hello"
grep ToolSearch /tmp/ts.log
```

## What the multiplexer controls

| Lever | Where it is |
|---|---|
| The model of a session, and of a schedule | [scheduler.md](scheduler.md) |
| The open tools a session carries | [mcp/profiles.md](mcp/profiles.md) |
| The size of the context a turn re-reads | [sessions/context.md](sessions/context.md) |
| The visibility of the hit rate | [tui/sessions/bars.md](tui/sessions/bars.md) |

The tool profile is a smaller lever than it first appears, for the reason above:
it removes a deferred entry rather than a loaded schema. Measure the prefix of a
session the manager spawns before you size any further work on it.

> The `cmux run` command starts a bare child and attaches no MCP server, so a
> transcript from it holds no `mcp__cmux__` tool. Measure from a session the
> manager spawns.
