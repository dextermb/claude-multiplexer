# Session configuration

`session.Config` names the child. `Dir`, `Model`, `PermissionMode`,
`AllowedTools`, `DisallowedTools`, `ResumeID`, and `SessionID` become command
line flags. `ClaudePath` selects the binary, which the tests point at a fake.
`TranscriptPath` turns on the transcript. An empty path turns it off.

Two flags change what the child sends back. `ReplayPrompts` adds
`--replay-user-messages`, so your prompt returns on stdout and reaches the
transcript. `IncludePartial` adds `--include-partial-messages`, so the text
arrives while the model writes it. The manager turns both on for every session
it supervises. See [protocol.md](../protocol.md).

The defaults are the permission mode `auto`, the binary `claude`, an event
buffer of 256, and 20 remembered stderr lines.

`New` rejects a directory that does not exist, because the failure is otherwise
a child that dies with no clear reason.
