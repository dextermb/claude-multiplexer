# TUI redesign in the Blackline design system — what is still ahead

**Status:** in progress, on branch `claude/tui-blackline-redesign-884ec0`, not
merged. Built: the tokens, the palette, the state words, the grey markdown, the
frame of labelled rules, the painted black ground, the move to Charm v2, huh
forms for the settings and rename dialogs, the danger stop dialog, and the scrim
under every dialog. What they do is described in
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

## Open questions

1. **Waiting state.** Info blue is the quietest state colour, but `waiting` is
   the state that needs the user. It is info for now.
2. **Job lines in the transcript.** `internal/render` still writes `⚙ started`
   and `⚙ done`. Change them to `■ started` and `✓ done`, or leave the
   transcript text alone?

## Still ahead

1. **The key list in three columns.** The canvas board "key list" shows it: a
   black dialog with three columns split by shared rules, the stop key in the
   danger colour, and a cursor row that runs its key on `enter`. Today the list
   is one column with a filter, and it runs nothing.
2. **The new-session form in huh, or not.** The settings and rename dialogs are
   huh forms. The new-session form stays custom: its directory field completes
   paths on `tab` and walks them on `shift+tab`, and its peer mode depends on the
   host. huh has no field for either. Restyle it to the canvas board instead:
   segmented selects and `[ ]` input brackets.
3. **Key hints from bubbles.** Replace the hand-built hint list with
   `bubbles/key` and `bubbles/help`.
Verify each step with `just check`, and in a real terminal at 256 colours and at
truecolor.
