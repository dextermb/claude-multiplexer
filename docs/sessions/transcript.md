# The transcript

Every raw stdout line goes to `TranscriptPath` as JSON Lines, before the decoder
result reaches anyone. Memory holds only the counters and the last stderr lines.

There is one exception. A `stream_event` line is not written, because it repeats
text that the `assistant` line already holds. See [protocol.md](../protocol.md).
So the file is the complete history of the conversation, and not of the wire.

The counters are derived, and not stored. `Snapshot` reports the state, the
Claude session identifier, the accumulated cost, the number of turns, the
duration of the last turn, the accumulated tokens, and the queue length.

A turn is a turn of the agent loop, and not a prompt. Claude Code reports
`num_turns` for each prompt it answers, and one prompt that reads files and
calls tools reports many. So the count measures the work of the session, and it
is larger than the number of prompts you sent.

`Snapshot.Model` and `Snapshot.PermissionMode` report what the child confirms.
They start as the flag values, and the `init` event replaces them. So an empty
`--model` becomes the model Claude Code chose, and a mode the child changes is
the mode the interface shows.
