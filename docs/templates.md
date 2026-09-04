# Preset prompts

A preset is a prompt you keep, with holes in it. You fill the holes and send it.
The package `internal/template` reads them, and the interface offers three ways
to reach one.

## Writing one

A template is a markdown file. The file name, without `.md`, is its name.

`~/.claude-multiplexer/templates/linear.md`:

```markdown
---
description: Work a Linear issue from end to end
---
Look up Linear issue {{issue}}.

Read the description and every comment, then plan the work.
Focus on {{focus=correctness}}. Do not push anything.
```

That gives you `/linear`, and it asks for two things.

- Every `{{name}}` is a field. The order is the order they first appear, and a
  name that appears twice is one field.
- `{{name=value}}` gives the field a default. The form starts with it filled in,
  so you can press Enter and move on.
- The front matter is optional. It holds `key: value` lines, and only
  `description` is read. There is no YAML parser and no dependency.
- With no front matter, the first line of the body becomes the description.

## Where they live

| Directory | For |
|---|---|
| `~/.claude-multiplexer/templates/` | Every session. `--root` moves it. |
| `~/.multiplexier/templates/` | The same, under the older state directory. |
| `~/.multiplexer/templates/` | The same, under the other older spelling. |
| `<session directory>/.multiplexier/templates/` | One repository. |
| `<session directory>/.multiplexer/templates/` | The same, with the other spelling. |

**Every spelling is read.** A template in any of these directories is found, so
the letter you do or do not type costs you nothing. `--root` moves the first
row, and a root it names is read on its own.

Sessions and transcripts are a different matter. They are written to one place
only, which is the state directory. See [manager.md](manager.md).

A project template wins when both hold the same name. So a repository can give
`/review` its own meaning without touching the one you keep at home. At home,
the state directory wins over the older spellings. In a repository,
`.multiplexer` wins over `.multiplexier`.

`multiplexer templates` lists what exists, with the fields each one takes:

```
/linear  issue focus=correctness       Work a Linear issue from end to end
/review                                Review the diff on this branch
```

## The three ways in

### The picker

Press `t` when the focus is the list or the output, or `ctrl+p` at any time. Type to narrow
the list, press Enter to choose, fill the fields, and press Enter again.

The finished prompt goes into the prompt box, and the focus goes with it.
Nothing is sent. Read it, change it, and press Enter to send.

### The slash name

Type `/lin` in the prompt box, and the names that match appear above it. Press
Tab to complete the name. Then:

```
/linear ENG-123 the retry path
```

Arguments fill the fields in order, and the last field takes the rest of the
line. So `issue` is `ENG-123` and `focus` is `the retry path`. Press Enter and
it goes at once.

Leave a field out and the form opens, with what you did give already filled.

### Naming a field instead

A field can be named, in any order:

```
/linear issue=ENG-123 focus="the retry path"
```

- A value that holds a space needs quotes, single or double.
- A named field can be mixed with positional ones. `/linear focus=tests ENG-9`
  names `focus`, and `ENG-9` then fills `issue`, which is the field still open.
- `name=` with nothing after it counts as missing, so the form opens.

### Spaces and quotes

Quotes work the same way for a named value and for a positional one:

```
/linear id="multiple spaces" "fill the next field"
/linear "multiple spaces" "fill the next field"
```

A quote only opens a quoted value at the start of an argument, or straight
after an `=`. Everywhere else it is an ordinary character, so `don't push` keeps
its apostrophe and `-run=Test\d+` keeps its backslash.

Inside a quoted value, `\"` gives a quote. Nothing else is escaped.

**Only a name that the template knows is read as a name.** A template with one
field `{{command}}` takes `/run go test -run=TestThing` as one whole command,
because `-run` is not one of its fields. So an argument that holds an `=` is
safe.

### A new session that starts working

The new session form has a **first prompt** field. It takes plain text, or a
slash name with its arguments. The session starts, and the prompt goes with it.

## Two slash syntaxes, one rule

Claude Code reads `/name` itself, for its own commands, skills, and plugins. So
does this feature. The rule:

1. The name matches one of your templates: the multiplexer expands it, and the
   child never sees a slash.
2. Anything else goes to the child unchanged.

So an unknown `/name` is not an error. It is a Claude Code command that we do
not know about, and it still works exactly as it did. See
[protocol.md](./protocol.md).

## What it does not do

A template cannot send a literal `{{`. Every one is read as a field. If you need
the braces in a prompt, type them in the prompt box instead.
