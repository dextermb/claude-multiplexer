# The context governor

A session pays for its whole context on every turn. The context grows with each
turn, so the cost of a turn climbs while the work of a turn does not. The
governor watches that growth, and it tells you before the bill does.

The governor is off until you turn it on. See [../config.md](../config.md).

## What it watches

The session bar already shows the context fill, as `ctx 12.2k/200k (6%)`. The
governor reads the same number: the context tokens of the session over the
context window of its model.

A model with an unknown window has no percent, so the governor does nothing for
it. See [../tui/sessions/bars.md](../tui/sessions/bars.md).

## The two thresholds

```
   fill crosses contextWarnPercent
              |
              v
     a notice, and the session runs on
              |
   fill crosses contextActPercent
              |
              v
     contextAction decides:
       "notify"  ->  a notice only
       "hold"    ->  the session takes no more prompts
```

| Setting | What it holds |
|---|---|
| `contextWarnPercent` | The percent at which the governor raises a notice |
| `contextActPercent` | The percent at which the governor takes the action |
| `contextAction` | `notify` or `hold`. An empty value is `notify` |

Each threshold fires once. A session that passes the warn percent raises one
notice, and it does not raise it again as the context keeps growing. A session
that reaches the act percent skips the warn notice, because the later state is
the one you need to read.

Set one threshold or both. A nil percent, or a percent below 1, turns that
threshold off, and the governor sleeps when both are off.

## The hold

A hold is a guard, and not a repair. A held session keeps its whole context and
its place in the work. It refuses a new prompt, and it says why:

```
manager: the session is held, because its context is full: alpha
```

The sidebar marks a held session with `!`, and `s h` clears the hold on the
selected session. A cleared hold does not reset the thresholds, so the governor
does not raise the same notice twice.

A hold refuses a prompt from a human and a prompt from another session alike, so
a control session cannot drive past the guard without a human.

## Why it never compacts

A compaction throws away context to buy room. It is often the right answer, but
only a human knows which part of a long session still matters. So the governor
raises a notice, or it holds, and it leaves the choice with you. A silent loss
of context is worse than a known cost.

## The cost of the check

The governor reads the settings file only when the fill of a session changes,
and only while a threshold is still uncrossed. A session that crossed both
thresholds does no further work, and a session with the governor off does one
comparison per change.
