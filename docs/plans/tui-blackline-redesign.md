# TUI redesign in the Blackline design system — what is still ahead

**Status:** in progress, on branch `claude/tui-blackline-redesign-884ec0`, not
merged. Built: the tokens, the palette, the state words, the grey markdown, the
frame of labelled rules, the painted black ground, the move to Charm v2, huh
forms for the settings and rename dialogs, the danger stop dialog, the scrim
under every dialog, the key list in columns, the restyled new-session form, the
inline question dialog, the review frame, the prompt label in its rule, and the
tool name as a label in the transcript. What they do is described in
[../tui/theme.md](../tui/theme.md), [../tui.md](../tui.md) and
[../markdown.md](../markdown.md). This file holds only the work that is not
built.

The design canvas: <https://claude.ai/artifact/B53CvRFbfHa1UnkisHtGjC>.

---

## Decisions

1. **Dialogs are black.** A dialog sets no background, so it shows the painted
   black ground. Blackline puts dialogs on `color-elevated`; this design does
   not.
2. **The ground is black on every terminal.** The interface paints every cell,
   so no terminal background shows. So there is one set of tokens, the dark set,
   and no light theme. Rejected: asking the terminal to change its background
   (OSC 11), because not every terminal honours it.
3. **The sidebar is 30 columns by default**, to fit the state word.
4. **The busy timer replaces the spinner.** The interface records when a session
   becomes busy. The session reports no turn start time.
5. **Charm v2.** The interface is on `charm.land/bubbletea/v2`,
   `charm.land/lipgloss/v2`, `charm.land/bubbles/v2` and `charm.land/glamour/v2`.
   Bubble Tea v2 maps each hex grey to its exact xterm grey, so the tokens are
   plain hex. It can also set the terminal background from
   `tea.View.BackgroundColor`; the painted ground stays, because it works on
   every terminal.
6. **The new-session form stays custom.** Its directory field completes paths on
   `tab` and walks them on `shift+tab`, and its peer mode depends on the host;
   huh has no field for either. It is restyled to the canvas instead.
7. **Key hints stay hand-built.** The status bar hints and the key list read the
   one key table in `internal/tui/help.go`. `bubbles/help` would draw the same
   text from a second copy of the bindings, so it is not used.

## Open questions

1. **Waiting state.** Info blue is the quietest state colour, but `waiting` is
   the state that needs the user. It is info for now.
3. **The Windows paste rule.** Bubble Tea v2 marks a paste on Windows, so the
   timing rule in `internal/tui/burst_windows.go` may be unnecessary. Test a
   multi-line paste on Windows before removing it.

## Still ahead

1. **Remove the light theme board** from the canvas, or change it: the app has
   no light theme now.

Verify each step with `just check`, and in a real terminal at 256 colours and at
truecolor.
