# Background jobs

Claude Code can run a shell command in the background. The multiplexer shows
each background job in four places. See [../../sessions.md](../../sessions.md) for
the job model, and [../../protocol/jobs.md](../../protocol/jobs.md) for the wire
events.

- The output pane marks each job in order. A start line reads `⚙ started ·
  <description>`. A stop line reads `⚙ done · <id>`, or `failed`, or `killed`.
- The sidebar row shows `⚙n` for `n` running jobs, next to the queue badge. The
  badge clears when the last job stops.
- The session bar shows a `⚙n` segment while jobs run, next to the queue segment.
- The side panel lists every job above the task list. See [../tasks.md](../tasks.md).

Press `s j` to open the jobs dialog for the selected session. The dialog draws in
the pane, so the sidebar stays on the screen. It has two levels. See
[../keys.md](../keys.md) and [../../tui.md](../../tui.md).

## The list

The first level lists every job, the running ones first, then the finished ones,
in start order. Each row shows a status glyph, the status word, and the job
description. A `▸` marks the row under the cursor. Press `enter` to open that
job, and `esc` to close the dialog.

## One job and its output

The second level shows one job. A head names the status, the command, the task
type, the start and end times, the summary, and the output file. The output
follows, under a rule.

The dialog is live. It takes the jobs of the session on every event, so a
running job grows while you read it, and the cursor holds its own job when a job
stops and moves down the list. The open job re-reads its output file twice a
second, and the view follows the new lines only when you already sit at the
bottom. That tick runs only while a job is open.

The body says why when there is no output:

| The body says | Because |
|---|---|
| `Waiting for the output file.` | The job started, and the launch result has not named its file yet. |
| `No output yet.` | The file is absent or empty, so the job has printed nothing. |
| `The output could not be read: …` | The path is not a job output file, or the read failed. |

Press `esc` to step back to the list, and `esc` again to close the dialog. See
[../../sessions.md](../../sessions.md) for where the path comes from, and for the
guards on the read.
