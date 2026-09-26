# TUI redesign in the Blackline design system — what is still ahead

**Status:** in progress, on branch `claude/tui-blackline-redesign-884ec0`, not
merged. Steps 1 to 3 are built (the tokens, the palette, the state words, the
grey markdown, and the frame of labelled rules). What they do is described in
[../tui/theme.md](../tui/theme.md), [../tui.md](../tui.md) and
[../markdown.md](../markdown.md). This file holds only the work that is not
built.

The design canvas: <https://claude.ai/artifact/B53CvRFbfHa1UnkisHtGjC>.

---

## Decisions

1. **Dialogs are black.** A dialog sets no background, so each cell takes the
   terminal background. Blackline puts dialogs on `color-elevated`; this design
   does not.
2. **The terminal background stays the user's.** The interface does not force
   pure black. The tokens pick the light or the dark set from the terminal.
3. **The sidebar is 30 columns by default**, to fit the state word.
4. **The busy timer replaces the spinner.** The interface records when a session
   becomes busy. The session reports no turn start time.
5. **Greys are written as xterm numbers.** Lip Gloss v1 maps `#3a3a3a` to 59,
   not 237, so each token carries its 256-colour and 16-colour value by hand.
6. **Steps 1 to 3 stay on v1.** The v2 move is a separate change, so the look
   can be judged first.

## Open questions

1. **Waiting state.** Info blue is the quietest state colour, but `waiting` is
   the state that needs the user. It is info for now.
2. **Light theme borders.** The light `color-border` (`#e4e4e4`) is 1.3:1 on
   white. It is kept for now. `color-border-strong` is the fallback.
3. **Job lines in the transcript.** `internal/render` still writes `⚙ started`
   and `⚙ done`. Change them to `■ started` and `✓ done`, or leave the
   transcript text alone?

## Still ahead

1. **The key list in three columns.** The canvas board "key list" shows it: a
   black dialog with three columns split by shared rules, the stop key in the
   danger colour, and a cursor row that runs its key on `enter`. Today the list
   is one column with a filter, and it runs nothing.
2. **Move to Charm v2.** `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`,
   `charm.land/bubbles/v2` and `charm.land/glamour/v2`: `tea.View`,
   `KeyPressMsg`, and `BackgroundColorMsg` in place of the warm-up call in
   `tui.Run`. Check whether v2 maps the hex greys exactly; if it does, the
   hand-written numbers in `theme.go` can go.
3. **Dialogs on layers, over a scrim.** With the Lip Gloss v2 compositor, draw a
   dialog over a faint copy of the screen, in place of the region it replaces
   now.
4. **huh for the forms.** The new-session form, the preset field form, the
   choice dialogs, rename, and the stop confirm become huh forms with one
   Blackline theme.
5. **Key hints from bubbles.** Replace the hand-built hint list with
   `bubbles/key` and `bubbles/help`.
6. **The stop confirm as a danger dialog.** The canvas shows `[ keep running ]`
   and a danger `stop session` button. Today it is text with `y stop`.

Verify each step with `just check`, and in a real terminal at 256 colours and at
truecolor.
