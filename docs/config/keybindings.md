# Keybindings

Every key of the interface is rebindable from the settings file, except a small
reserved set. The multiplexer starts from the built-in defaults, then overlays
the bindings the settings file names. A rebind takes effect on the next reload,
without a restart. See [../config.md](../config.md).

The help overlay (`?`), the status bar, and [../tui/keys.md](../tui/keys.md) all
read the resolved bindings, so a rebind never makes the on-screen list wrong.

## The reserved set

Three keys stay fixed and cannot be rebound, because they are the guaranteed way
out of a dialog or a pane:

| Key | What it does |
|---|---|
| `?` | Open the help overlay |
| `esc` | Cancel a sequence, close a dialog, leave a pane, or stop a turn |
| `ctrl+c` | Clear the prompt, then quit |

A binding that names a reserved key is refused.

## The shape

The `keybindings` block maps an action to its keys. An action is named for what
it does, as `<context>.<action>`, and each field is a list of keys:

```json
{
  "keybindings": {
    "session": { "rename": ["N"], "stop": ["X"] },
    "global":  { "quit": ["q", "Q"] }
  }
}
```

An absent action keeps its default. A key is a bubbletea key name, such as `n`,
`ctrl+n`, `shift+tab`, `enter`, or `pgup`.

## A context is where a key is read

The same key means one action in one context, and a different action in another,
so `j` moves the list and also scrolls the output. A **target** is the first key
of a two-key sequence; the other contexts are the panes.

| Context | What it is |
|---|---|
| `targets` | The first key of a two-key sequence |
| `session` | The second key after the session target |
| `list` | The second key after the list target |
| `output` | The second key after the output target |
| `diff` | The second key after the diff target |
| `global` | The keys that work on their own |
| `outputPane` | The output pane, when it holds the focus |
| `sidebar` | The list, when it holds the focus |
| `prompt` | The prompt box |
| `task` | The task and job panel |
| `diffPane` | The diff panel, when it holds the focus |
| `review` | The code review screen |

## The actions and their defaults

`targets` — the first key of a sequence:

| Action | Default |
|---|---|
| `targets.session` | `s`, `ctrl+s` |
| `targets.list` | `l`, `ctrl+l` |
| `targets.output` | `o`, `ctrl+o` |
| `targets.diff` | `d` (only while the diff panel is open) |

`session` — the second key after `s`:

| Action | Default | Action | Default |
|---|---|---|---|
| `session.new` | `c` | `session.editor` | `E` |
| `session.presets` | `t` | `session.model` | `m` |
| `session.resume` | `r` | `session.effort` | `e` |
| `session.rename` | `n` | `session.mode` | `p` |
| `session.archive` | `a` | `session.control` | `C` |
| `session.archiveAll` | `A` | `session.clearHold` | `h` |
| `session.stop` | `x` | `session.diff` | `d` |
| `session.jobs` | `j` | `session.review` | `R` |
| `session.focusTasks` | `k` | | |
| `session.files` | `f` | | |

`list` — the second key after `l`:

| Action | Default | Action | Default |
|---|---|---|---|
| `list.fold` | `f` | `list.search` | `s` |
| `list.foldOthers` | `F` | `list.sidebar` | `t` |
| `list.unfold` | `u` | `list.collapse` | `c` |
| `list.archived` | `a` | `list.expand` | `e` |

`output` — the second key after `o`:

| Action | Default |
|---|---|
| `output.markdown` | `m` |
| `output.layouts` | `l` |
| `output.age` | `a` |

`diff` — the second key after `d`:

| Action | Default | Action | Default |
|---|---|---|---|
| `diff.wider` | `+` | `diff.numbers` | `n` |
| `diff.narrower` | `-` | `diff.pr` | `p` |
| `diff.half` | `/` | `diff.allPrs` | `P` |

`global` — the keys that work on their own:

| Action | Default | Action | Default |
|---|---|---|---|
| `global.newSession` | `n`, `ctrl+n` | `global.focusNext` | `tab` |
| `global.presets` | `t`, `ctrl+p` | `global.pageUp` | `pgup` |
| `global.toggleMouse` | `ctrl+t` | `global.pageDown` | `pgdown` |
| `global.quit` | `q` | | |

`outputPane`, `sidebar`, and `prompt`:

| Action | Default | Action | Default |
|---|---|---|---|
| `outputPane.openBlock` | `enter` | `outputPane.halfUp` | `u`, `ctrl+u` |
| `outputPane.toggleBlock` | `space` | `outputPane.halfDown` | `d`, `ctrl+d` |
| `outputPane.blockNext` | `]` | `outputPane.top` | `g`, `home` |
| `outputPane.blockPrev` | `[` | `outputPane.bottom` | `G`, `end` |
| `outputPane.toPrompt` | `i` | `sidebar.up` | `up`, `k` |
| `outputPane.up` | `up`, `k` | `sidebar.down` | `down`, `j` |
| `outputPane.down` | `down`, `j` | `sidebar.enter` | `enter`, `i` |
| `prompt.send` | `enter` | `prompt.newline` | `ctrl+j` |
| `prompt.unqueueLast` | `backspace` | | |

`task` — the task and job panel:

| Action | Default | Action | Default |
|---|---|---|---|
| `task.up` | `up`, `k` | `task.pageUp` | `pgup` |
| `task.down` | `down`, `j` | `task.pageDown` | `pgdown` |
| `task.halfUp` | `u`, `ctrl+u` | `task.top` | `g`, `home` |
| `task.halfDown` | `d`, `ctrl+d` | `task.bottom` | `G`, `end` |
| `task.focusNext` | `tab` | | |

`diffPane` — the diff panel:

| Action | Default | Action | Default |
|---|---|---|---|
| `diffPane.up` | `k` | `diffPane.pageDown` | `pgdown` |
| `diffPane.down` | `j` | `diffPane.top` | `g` |
| `diffPane.left` | `h` | `diffPane.bottom` | `G` |
| `diffPane.right` | `l` | `diffPane.jumpDown` | `}`, `shift+]` |
| `diffPane.lineUp` | `up` | `diffPane.jumpUp` | `{`, `shift+[` |
| `diffPane.lineDown` | `down` | `diffPane.toggle` | `enter`, `space` |
| `diffPane.pageUp` | `pgup` | `diffPane.focusNext` | `tab` |

`review` — the code review screen:

| Action | Default | Action | Default |
|---|---|---|---|
| `review.hunkNext` | `j`, `down` | `review.pageUp` | `pgup` |
| `review.hunkPrev` | `k`, `up` | `review.pageDown` | `pgdown` |
| `review.fileNext` | `}`, `shift+]` | `review.numbers` | `n` |
| `review.filePrev` | `{`, `shift+[` | `review.explainHunk` | `e` |
| `review.top` | `g`, `home` | `review.explainFile` | `E` |
| `review.bottom` | `G`, `end` | `review.focusNext` | `tab` |

A few text-editing keys of the prompt are not rebindable, because they are part
of the text box: the arrow keys recall the older and newer prompts, and
`shift+tab` walks the paths that match an `@` word.

## Precedence and conflicts

A **key pattern** is the whole trigger of an action: a standalone key (`n`), a
control key (`ctrl+n`), or a two-key sequence (`o a`). One pattern maps to one
action inside its context.

When a default and a user binding both claim a pattern:

1. A **user** binding wins over a **default** binding.
2. Two **user** bindings on one pattern in one context have no tiebreak, so the
   file is refused and the previous keymap stays. The status bar shows the
   reason.

When a user binding takes a pattern a default action still holds, the user
binding wins and that default action loses the pattern. If the default action
has no other pattern, it becomes unbound. The multiplexer warns which action it
displaced, so you can give that action a new key.

Example. The default binds `list.archived` to `l a` and `output.age` to `o a`. A
user binds `session.archive` to `o a`. Now `session.archive` runs on `o a`,
`output.age` loses `o a` and is unbound, and the warning names `output.age`.

## The tools

`get_keybindings` lists the resolved bindings — every action, the keys it
answers to now, and whether the settings changed it. `set_keybinding` and
`reset_keybinding` write one action each, and validate the whole keymap before
they write. See [../mcp/tools/settings.md](../mcp/tools/settings.md).
`set_config` reaches the same fields by the path
`keybindings.<context>.<action>`, and `list_config_keys` lists every path.
