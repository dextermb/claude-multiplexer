# The theme

The interface follows Blackline, a black and grey design system for dense
tools. It is quiet at rest and loud on interaction: nearly everything is grey on
black, and the thing you point at inverts to white.

The tokens live in `internal/tui/theme.go`. Every style in the interface takes
its colour from a token. No other file names a colour.

## One hex value for each token

Each token is one truecolor hex value. Lip Gloss v2 always writes the full
colour, and Bubble Tea v2 converts it to the colours the terminal supports as it
draws the screen. There is one set of tokens, the dark set, because the ground
is always black (see below).

Each Blackline grey is an xterm grey, and the conversion finds it exactly, so a
256-colour terminal shows every grey with no loss. The xterm column below is
what a 256-colour terminal gets.

On a 16-colour terminal the conversion is coarse. The greys fall to three
levels, and the warning amber becomes bright red, so there the warning and the
danger colours look the same. The word beside each colour still tells them
apart.

| Token | Dark | xterm | Used for |
|---|---|---|---|
| `colGround` | `#000000` | 16 | the ground |
| `colSurface` | `#080808` | 232 | the status bar and the search box |
| `colCode` | `#262626` | 235 | inline code and code blocks |
| `colFg` | `#ffffff` | 231 | keys, strong text, the focused value |
| `colHeading` | `#eeeeee` | 255 | your prompts, dialog titles, headings |
| `colSecondary` | `#c6c6c6` | 251 | assistant text, session names, tool input |
| `colMuted` | `#8a8a8a` | 245 | labels, hints, tool names and results |
| `colDimmed` | `#6c6c6c` | 242 | glyphs, separators, meta lines |
| `colFaint` | `#3a3a3a` | 237 | archived rows |
| `colBorder` | `#3a3a3a` | 237 | every rule and border |
| `colSubtle` | `#1d1d1d` | 234 | rules inside a pane |
| `colScrim` | `#262626` | 235 | the faint copy under a dialog |
| `colAccent` | `#ffffff` | 231 | the inverted fill |
| `colAccentFg` | `#000000` | 16 | text on the inverted fill |
| `colSubdued` | `#2e2e2e` | 236 | the selected row when its pane has no focus |

Blackline writes some tokens with alpha, such as the subdued tint. A terminal
cell has no alpha, so each tint is composed over the ground one time, in the
token.

## Colour is state, and always has a word

Only four tokens have a hue. Each one marks a state, and a word always stands
beside it, so the colour is never the only mark.

| Token | Dark | xterm | Marks |
|---|---|---|---|
| `colPositive` | `#4ade80` | 78 | idle, done, inserted lines |
| `colWarning` | `#fbbf24` | 214 | busy, running, stderr, the update notice |
| `colDanger` | `#ff6666` | 203 | failed, errors, deleted lines |
| `colInfo` | `#60a5fa` | 75 | waiting for an answer |

The waiting state needs you, but it takes the quietest hue, info blue. The word
`waiting` and the count in the top band call you, so the colour does not have
to. Amber would give busy and waiting one colour, and inversion means the focus.

Three tints (`colPositiveBg`, `colWarningBg`, `colDangerBg`) sit behind a notice
row, and the positive and danger tints sit behind an added and a removed row of
a diff, in the diff panel and on the review screen. They are for truecolor. A 256-colour terminal draws each one as
xterm 232, a near-black that hardly shows, so there the green and red text of
a diff row carries the change alone.

## Inversion

The inverted style (`invertStyle`) is white fill with black text. It marks the
thing that has the focus or the cursor:

- the label of the focused pane, set in its rule,
- the current screen in the band,
- the target of a key sequence, in the status bar,
- the selected row, when its pane has the focus,
- the row under the cursor in a dialog,
- the block marker under the output cursor,
- the prompt label, when the prompt has the focus,
- the header of the focused pane on the review screen.

A selected row whose pane has no focus takes the subdued style instead: white
text on `colSubdued`.

## Type

The terminal sets the font and the size, and Blackline uses one weight. So the
interface sets no bold and no italic. Hierarchy comes from case, colour, and
rules. Labels, titles, and group names are uppercase through
`lipgloss.Style.Transform`, so the text in the code stays in lowercase.

## Key hints

A key hint reads `[n] new`: the brackets dimmed, the key white, and the word
muted. Two spaces separate one hint from the next. The status bar, the question
dialog, and the key list use this form, from `styleHints` and `paneHints` in
`internal/tui/help.go`. A hint list that does not fit sheds whole hints from its
end and ends with `…`.

## Forms

The huh forms take the theme `blacklineForm` in `internal/tui/huhdialog.go`. The
focused field has a white rule on its left, and a field at rest has none. The
option under the cursor inverts, and the current value of a setting carries a
filled mark (`(●)`). The focused button inverts. Titles are uppercase labels.

The stop confirmation is the one dialog with a coloured button: `stop session`
fills with the danger colour, because the stop is destructive.

A dialog sets no background, so it shows the black ground. Blackline puts a
dialog on `color-elevated`, but here the scrim under the dialog already sets it
apart from the screen.

The new session form is not a huh form. Its directory field completes a path on
`tab` and walks the matches on `shift+tab`, and its peer mode depends on the
host. huh has no field for either, so the form is custom, drawn in the same
look.

## Inputs

The prompt, the search box, and every input in a dialog are Bubbles text areas
and text inputs, made by `newTextArea` and `newTextInput` in
`internal/tui/inputs.go`. They take the Blackline greys: your text in the heading
grey, the placeholder and the prompt mark dimmed, and no band on the cursor line.

## The ground is black on every terminal

The interface paints every cell with `colGround`, so the screen looks the same
whatever background the terminal has. `paintGround` in `internal/tui/ground.go`
does it as the last step of `View`:

1. Each line starts with the ground as its background.
2. A reset (`ESC[0m`, `ESC[m`, or `ESC[49m`) clears the background with the other
   attributes, so the ground is set again after each one.
3. Each line is padded to the full width, and the view to the full height, so no
   cell is left to the terminal.

A style that sets its own background, such as the inverted style, still wins,
because its code comes after the ground. A style with no background shows the
ground under it.

The interface does not ask the terminal to change its own background (OSC 11).
Not every terminal honours that request, and the terminal keeps the new colour
if the program stops without a restore. Painting the cells works on every
terminal that shows colour. For the same reason the interface does not set
`tea.View.BackgroundColor`, which sends that request.

On a 16-colour terminal the ground is ANSI colour 0. Many terminal themes set
colour 0 to a dark grey, not to black, so the ground follows the theme there. On
a terminal with no colour the interface paints nothing.
