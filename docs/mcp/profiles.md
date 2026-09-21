# The tool profile of a session

A profile names the open tools a session carries. It is a cost control, and not
a permission. The control grant is a separate axis, and it adds the control
tools whatever the profile says. See [grant.md](grant.md).

## Why a profile exists

Every tool the multiplexer serves puts its name, its description, and its input
schema into the system prompt of the session. That text goes in every request
the session makes, for as long as the session lives. A session that never calls
a layout tool still pays for the layout schemas on every turn.

The position of the text makes this worse than the size alone. A prompt renders
in the order tools, then system, then messages, and the prompt cache matches a
prefix of that render. So the tool list sits at the front of every prefix, and a
change to it invalidates the whole cache entry.

A profile therefore does two things. It removes tools a session will never call,
and it keeps what remains the same from one session to the next.

Claude Code defers the schemas of an MCP tool until the model asks for one, so a
profile removes a deferred entry rather than a loaded schema. That makes the
saving smaller than the tool count suggests. See [../caching.md](../caching.md).

## The two profiles

| Profile | The open tools it carries |
|---|---|
| `minimal` | The session reads (`list_sessions`, `list_inactive_sessions`, `list_archived_sessions`, `get_messages`, `rename_session`, `list_jobs`, `get_config_path`, `get_bar_defaults`, `get_template_path`) and `get_api_docs` |
| `standard` | Every open tool: `minimal`, plus the config, layout, schedule, usage, and share tools |

`standard` is the default, so a session that names no profile behaves exactly as
it did before profiles existed.

Use `minimal` for a session that does one job and does not administer the
multiplexer — a scheduled poll, a worker a control session drives, a session
that only writes code. Use `standard` for a session you talk to, because it can
then change a setting or a layout when you ask.

## How a session takes its profile

Three sources, in order. The first one that names a profile wins:

1. The `profile` argument of `create_session`.
2. The `defaultToolProfile` setting. See [../config.md](../config.md).
3. The built-in default, `standard`.

A name that is not a profile fails `create_session` with an error, so a caller
learns of the mistake. The same name in the settings file falls back to
`standard` instead, because a mistyped setting must not stop every session from
starting.

## A profile is fixed for the life of a session

A session picks its profile once, when it starts. The profile is written into
the `--allowedTools` flag and into the tool set of the MCP server of that
session, and neither changes while the child runs.

This is deliberate. A tool set that changes mid-conversation invalidates the
prompt cache from the first byte, so a profile that could change would cost more
than it saves. To change the profile of a session, stop it and start it again.

## What a profile never does

A profile never widens what a session may do. `minimal` and `standard` both draw
from the open tools, and the control tools ride on the grant alone. So a control
session on the `minimal` profile keeps every control tool, and a session without
the grant gets no control tool on either profile.
