# The new session form defaults

The new session form has four select fields: the model, the permission mode, the
effort, and the control grant. Each field opens on a default option. The settings
file sets that option, so the form opens on the choice you make most. For the
form itself, see [../tui/input.md](../tui/input.md). For where the settings file
lives, see [../config.md](../config.md).

## The fields

| Field | Setting | Values | Built-in |
|---|---|---|---|
| Model | `defaultModel` | `default`, `opus`, `sonnet`, `haiku` | `default` |
| Permission mode | `defaultPermissionMode` | `acceptEdits`, `auto`, `bypassPermissions`, `default`, `dontAsk`, `plan` | `auto` |
| Effort | `defaultEffort` | `default`, `low`, `medium`, `high`, `xhigh`, `max` | `default` |
| Control | `defaultControl` | `true` or `false` | `false` |

```json
{
  "defaultModel": "sonnet",
  "defaultPermissionMode": "plan",
  "defaultEffort": "high",
  "defaultControl": false
}
```

The value of `defaultControl` is a boolean, because the control field takes only
`no` or `yes`. The other three take a string.

## What `default` means

The model and the effort have a `default` option. This option sends nothing to
Claude Code, so Claude Code takes the model or the effort from your project or
global configuration. To pin one instead, set the value.

The permission mode has no `default` option, because its own list holds a mode
named `default`. The form always sends a mode.

## The order that wins

For the model and the permission mode, three sources name the default, and the
first one that names one wins:

| Order | Source |
|---|---|
| 1 | `--model` or `--permission-mode` on the command line |
| 2 | `defaultModel` or `defaultPermissionMode` in the settings file |
| 3 | The built-in |

The effort and the control have no command-line flag, so the settings file wins
over the built-in.

A value that is not in the list of a field falls back to the first option of that
field. The interface reads the settings file again at each notice, so a change to
a default takes effect at the next form, without a restart.
