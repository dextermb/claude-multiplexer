# The frame of the canvas: band, bars and prompt hint

**Status:** awaiting go-ahead. Decisions 1 to 5 are proposals. The open
questions need an answer first.

The canvas (<https://claude.ai/artifact/B53CvRFbfHa1UnkisHtGjC>, board
"workspace") and the app differ in four places. The user chose to follow the
canvas in all four.

```
CANVAS
 MULTIPLEXER  WORKSPACE REVIEW KEYS   sessions (8 · 5 live) · 2 busy · 1 waiting · $4.21 / 1d
── SESSIONS (8) ─────────────┬── API ────────────────────────────────────────────────────
 ...                         │ opus · accept-edits · high      ■ busy 0:42 · +48 −12 · $0.84
                             │ › write the summary
─────────────────────────────┴─ PROMPT → API ────────────────────────────────────────────
 › type a prompt
                                   enter sends · ctrl+j newline · @ adds a file · ! runs a command
 [n] new  [t] preset  [s] session  [l] list  [o] output  [?] keys  [q] quit      synced 2s ago

APP TODAY
 MULTIPLEXER  WORKSPACE REVIEW KEYS                                              ■ 1 waiting
── SESSIONS (8) ─────────────┬── API ─ opus · auto ──── ■ busy 0:42 · queued (2) · $0.0212 ─
 ...                         │ › write the summary
─────────────────────────────┴─ PROMPT → API ────────────────────────────────────────────
 enter sends · ctrl+j new line · @ adds a file · ! runs a command
 › type a prompt
 3 sessions · 1 busy · $0.0881        n new  t preset  s session  l list  o output  ? keys  q quit
```

---

## Decisions (proposed)

1. **A third bar, `band`.** The band right side becomes a bar that the `bars`
   block of the settings file composes, the same as the other two. Its ids are
   `sessions`, `busy`, `waiting`, `failed`, and `cost`, plus custom elements and
   the session elements. Its default is all five, in that order. `sessions`
   reads `sessions (8 · 5 live)`: every row in the sidebar, then the live ones.
   `waiting` and `failed` keep their state colours; the other parts are muted.
   Rejected: a fixed band, because the rest of the chrome is configurable.
2. **The status bar holds the keys.** Its default becomes left `["hints"]`,
   right `["status"]`. `hints` and `status` become valid on both sides. The ids
   `sessions`, `busy` and `cost` stay valid on the status bar, so a settings
   file that names them still loads and still draws them.
3. **Bracketed hints.** Every hint in the status bar reads `[n] new`: the
   brackets dimmed, the key white, the word muted. The hints of a pending key
   sequence (` s → `) take the same form.
4. **The session bar is two rows.** The rule holds only the session name, as a
   label that inverts with the focus. The row under it holds the details: the
   left elements after `name`, and the right elements. The output pane is one
   row shorter. `name` in the settings stays valid, and the rule always shows
   the name.
5. **The prompt hint goes under the prompt text**, right-aligned, dimmed. The
   prompt area keeps its height.

## Open questions

1. **`synced 2s ago`.** The canvas shows it at the right of the status bar. The
   app has no sync clock. Proposal: the right side shows the status message
   (`status`, such as `started api`), and nothing when there is none. Or did
   the canvas mean a real clock, such as the last poll of the work items?
2. **The band on a narrow screen.** The band summary drops from the right when
   it does not fit: first `cost`, then `sessions`, then `busy`. `waiting` and
   `failed` stay last, because they call you. Is that order right?

## The build

1. `internal/config`: the `band` bar, its ids and default, `hints` and `status`
   on both sides of the status bar; `get_bar_defaults` returns three bars.
   Tests for the load and the validation.
2. `internal/tui/chrome.go`: the band draws the `band` bar.
3. `internal/tui/bar.go`: the status bar reads `hints` and `status` on either
   side; the bracketed hints; the session bar as the name rule and the details
   row. `barHeight` becomes 2.
4. `internal/tui`: the prompt hint row under the prompt text.
5. Tests: the band summary, the status bar default, the two-row session bar,
   and a settings file with the old status bar ids.
6. Docs: `docs/tui.md` (the layout diagram), `docs/config/bars.md` (the third
   bar, the new defaults), `docs/tui/input.md` (the hint row).
7. Check in tmux at 160 and at 80 columns.
