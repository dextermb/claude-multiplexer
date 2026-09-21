# The status bars

The interface draws two status bars. The `bars` block in the settings file
composes them: it reorders the built-in elements, removes one, and adds a custom
element that runs a script. A settings file with no `bars` block draws the
built-in defaults. See [../config.md](../config.md).

## The two bars

- The **session bar** sits above the output pane. It shows the selected session.
  The left side is the name and the session details. The right side is the
  state, the diff, the pull requests, the tokens, the cost, and more.
- The **status bar** is the bottom line. The left side counts the sessions and
  the total cost. The right side is the key hints. Both sides also accept any
  session element, drawn for the selected session.

Each bar has a left side and a right side. Each side is an ordered list of
elements. An element is a built-in id or a custom script.

## The built-in elements

A built-in element is a string, its id. Each side has its own set of ids.

| Bar | Side | Ids |
|---|---|---|
| session | left | `name`, `control`, `model`, `mode`, `effort`, `workItem` |
| session | right | `state`, `diff`, `pr`, `context`, `raw`, `scroll`, `jobs`, `queued`, `tokens`, `cache`, `cost` |
| status | left | `sessions`, `busy`, `cost`, `status` |
| status | right | `hints` |

An id outside its side is an error at load. A built-in element draws nothing
when it has no data. For example `diff` draws nothing with no change, and `pr`
draws one segment for each code base and none with no pull request.

The status bar also accepts every session element, on either side, drawn for the
selected session. So `cache` on the status bar right shows the cache hit rate of
the selected session, and `model` on the status bar left shows its model. The
element takes the status bar colours, and it draws nothing with no selection.

An id the status bar owns keeps its own meaning. `cost` on the status bar is the
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
  "left":  ["sessions", "busy", "cost", "status"],
  "right": ["hints"]
}
```

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

A narrow terminal cannot draw every element. The bar sheds the last element
first, so the order is also the priority: the first element stays the longest.

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

A status-bar element reads the totals:

```json
{
  "bar": "status",
  "totals": { "sessions": 3, "busy": 1, "cost": 0.0881, "costWindow": "1d" }
}
```

The `diff`, `prs`, and `workItem` fields are absent when the session has none.

### When a script runs

The interface runs a custom element only where it shows: the session bar of the
selected session, and the status bar. A script runs on its refresh interval, off
the main loop, and the interface caches the last line of output. A two-second
timeout stops a slow script.

A script that fails, times out, or prints an empty line keeps its last good
output, so a transient error does not make the element flicker. An element with
no good output yet draws nothing, and a first failure writes a one-line notice
to the status bar.

## Write it with set_config

Read `get_bar_defaults` first, then write a side with `set_config` on
`bars.session` or `bars.status`. The tool checks the ids, the script extensions,
and the refresh intervals before it writes, so a mistake fails and the file
stays as it was. The interface reads the file again after the write, so the
change takes effect at once. See [../config.md](../config.md).
