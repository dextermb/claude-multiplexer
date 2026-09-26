# The bars

The interface draws three bars. The `bars` block in the settings file composes
them: it reorders the built-in elements, removes one, and adds a custom element
that runs a script. A settings file with no `bars` block draws the built-in
defaults. See [../config.md](../config.md).

## The three bars

- The **session bar** heads the output pane. It shows the selected session. The
  rule holds the name, and the row under it holds the details: the left side is
  the session details, and the right side is the state, the diff, the pull
  requests, the tokens, the cost, and more.
- The **status bar** is the bottom line. The left side is the key hints, and the
  right side is the last message.
- The **band** is the top line. Its right side counts the sessions, the busy,
  waiting, and failed ones, and the total cost. Its left side holds the screens,
  so a settings file cannot compose it.

The status bar and the band also accept any session element, drawn for the
selected session. Each side is an ordered list of elements. An element is a
built-in id or a custom script. For what each bar shows, see
[../tui/sessions/bars.md](../tui/sessions/bars.md).

## The built-in elements

A built-in element is a string, its id. Each side has its own set of ids.

| Bar | Side | Ids |
|---|---|---|
| session | left | `name`, `control`, `model`, `mode`, `effort`, `workItem` |
| session | right | `state`, `diff`, `pr`, `context`, `raw`, `scroll`, `jobs`, `queued`, `tokens`, `cache`, `cost` |
| status | left or right | `hints`, `status`, `sessions`, `busy`, `cost` |
| band | right | `sessions`, `busy`, `waiting`, `failed`, `cost` |

An id outside its side is an error at load, and so is any element on the left
side of the band. The session bar always shows the name in its rule, so `name`
on the session bar draws nothing more. A built-in element draws nothing
when it has no data. For example `diff` draws nothing with no change, and `pr`
draws one segment for each code base and none with no pull request.

The status bar and the band also accept every session element, drawn for the
selected session. So `cache` on the status bar right shows the cache hit rate of
the selected session, and `model` on the band shows its model. The element takes
the colours of its bar, and it draws nothing with no selection.

An id the status bar or the band owns keeps its own meaning. `cost` there is the
total across the sessions, not the cost of the selected session.

## The defaults

Two files in the program hold the default composition. The tool
`get_bar_defaults` returns them, so an agent reads the default, then writes a
copy it edits. The default of the session bar is:

```json
{
  "left":  ["name", "control", "model", "mode", "effort", "workItem"],
  "right": ["state", "diff", "pr", "context", "raw", "scroll", "jobs", "queued", "tokens", "cache", "cost"]
}
```

The default of the status bar is:

```json
{
  "left":  ["hints"],
  "right": ["status"]
}
```

The default of the band is:

```json
{
  "right": ["sessions", "busy", "waiting", "failed", "cost"]
}
```

A settings file written for an older default, with the counts on the status bar
left and `hints` on its right, still loads and still draws them.

## Reorder and remove

A side the settings name replaces the default of that side. A side the settings
do not name keeps its default. So the list order is the display order, and an id
you leave out is removed.

```json
{
  "bars": {
    "session": {
      "right": ["cost", "state", "tokens"]
    }
  }
}
```

This example draws the cost first, then the state, then the tokens, and it
removes the other right elements. The session bar left side and the whole status
bar keep their defaults, because the block does not name them.

An empty list draws an empty side. A `null` value, or an absent side, keeps the
default.

## Shed to fit

A narrow terminal cannot draw every element. The session bar sheds the last
element first, so the order is also the priority: the first element stays the
longest.

The status bar sheds whole hints from the end of `hints` first, down to `…`, so
the other elements keep their text. Then it drops the other elements, the right
side first, each from its end.

The band drops a custom or session element first, then `cost`, then `sessions`,
then `busy`. `waiting` and `failed` go last, because they ask for you.

## Custom elements

A custom element is an object that names a script. The script runs, and the
first line of its output is the element text. For a worked example, see
[../example-scripts/bars/hello-world.sh](../example-scripts/bars/hello-world.sh),
which rotates between two words on each refresh.

```json
{
  "bars": {
    "session": {
      "right": ["state", "cost",
                { "script": "bars/branch.sh", "label": "branch", "refresh": "5s", "style": "muted" }]
    }
  }
}
```

| Field | What it holds |
|---|---|
| `script` | The path of the script. This field makes the element custom. |
| `label` | A name for the element, used in the payload and the notices. |
| `refresh` | How often the script runs, such as `2s` or `1m`. The default is 3 seconds, and the floor is 500 milliseconds. |
| `style` | The colour: `muted` (default), `strong`, `cost`, or `warn`. |

### The script and its runner

The interface picks the runner from the file extension:

| Extension | Runner |
|---|---|
| `.sh` | `bash` |
| `.py` | `python3` |
| `.go` | `go run` |

An extension outside this set is an error at load. A `.go` script compiles on
each run, so it is slower than the other two. The refresh interval and the cache
keep the cost low.

The path may be absolute, start with `~` for the home directory, or be relative
to the settings directory.

### The payload

The script reads a JSON object on its stdin. A session-bar element reads the
selected session:

```json
{
  "bar": "session",
  "session": {
    "name": "api", "title": "", "dir": "/repo", "workDir": "",
    "model": "claude-opus-4-8", "mode": "auto", "effort": "",
    "state": "busy", "live": true, "control": false,
    "cost": 0.0212,
    "tokens": { "input": 4200, "output": 300, "cacheRead": 940, "cacheWrite": 0, "context": 18000 },
    "diff": { "added": 120, "removed": 30 },
    "jobs": 2, "queued": 1,
    "prs": [{ "provider": "github", "number": 7, "state": "open", "unresolved": 3 }],
    "workItem": { "provider": "linear", "key": "ENG-1", "status": "In Progress" }
  }
}
```

A status-bar element and a band element read the totals. The `bar` field names
the bar, `status` or `band`:

```json
{
  "bar": "status",
  "totals": { "sessions": 3, "busy": 1, "cost": 0.0881, "costWindow": "1d" }
}
```

The `diff`, `prs`, and `workItem` fields are absent when the session has none.

### When a script runs

The interface runs a custom element only where it shows: the session bar of the
selected session, the status bar, and the band. A script runs on its refresh interval, off
the main loop, and the interface caches the last line of output. A two-second
timeout stops a slow script.

A script that fails, times out, or prints an empty line keeps its last good
output, so a transient error does not make the element flicker. An element with
no good output yet draws nothing, and a first failure writes a one-line notice
to the status bar.

## Write it with set_config

Read `get_bar_defaults` first, then write a side with `set_config` on
`bars.session`, `bars.status`, or `bars.band`. The tool checks the ids, the script extensions,
and the refresh intervals before it writes, so a mistake fails and the file
stays as it was. The interface reads the file again after the write, so the
change takes effect at once. See [../config.md](../config.md).
