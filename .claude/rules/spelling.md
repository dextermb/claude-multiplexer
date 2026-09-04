# Spelling

Prose uses British English. Identifiers keep the spelling they already have.

# British English in the prose

Write `colour`, `behaviour`, `licence` (the noun), `recognise`, and
`standardise`. This holds in every document, every commit message, and every
answer to the user.

American spelling stays in three places, because there it is a name and not a
word:

- An identifier in the code, and a file name. Do not rename `Color` to
  `Colour` to follow this rule.
- CSS, and a Web API.
- A quotation. Quote what the other text says, not what it should have said.

# The program is `multiplexer` in the prose

The word is **multiplexer**. Write "the multiplexer" wherever the program is the
subject of a sentence.

The command is `multiplexer`, and so is the source directory,
`cmd/multiplexer`. Four names keep the older spelling, because they are already
written on a disk:

| Where | The name |
|---|---|
| The state directory | `~/.multiplexier`, and `--root` moves it |
| The settings file | `$XDG_CONFIG_HOME/multiplexier/config.json` |
| A template directory | `.multiplexier/templates/` |
| The environment variable of the real tests | `MULTIPLEXIER_REAL` |

The settings file and the template directories read both spellings, and read
`multiplexer` first. See [../../docs/config.md](../../docs/config.md) and
[../../docs/templates.md](../../docs/templates.md).

So a sentence names the program with the word, and a code span names the file or
the command with the name:

- Good: The multiplexer writes the transcript under `~/.multiplexier`.
- Bad: The multiplexier writes the transcript under `~/.multiplexer`.

# One term for one thing

This rule is the spelling half of "one meaning per word, one word per meaning"
in [language.md](./language.md). A reader who meets two spellings of one name
has to work out whether they are two things. They are not.
