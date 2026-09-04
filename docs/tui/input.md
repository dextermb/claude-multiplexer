# Typing, dropping, and starting a session

The prompt box, a file dropped on the window, and the new session form.

## The size of the box

The box holds one row of text, and it grows to four as you type. It counts the
rows the terminal shows, so a line that wraps grows the box in the same way a
new line does. Above four rows the box stops, and the text scrolls inside it.
Remove the lines and the box shrinks again.

The output pane gives up the rows the box takes, and it takes them back. The
window height never changes.

## Naming a file with `@`

Start a word with `@` and the box suggests paths. The names that match appear
under the label, six at a time, and the count of the rest follows them.

The path is read against the directory of the selected session, because that is
where the prompt is going. `@/tmp` names an absolute path, `@notes.md` names a
file next to the session, and `@~` is your home directory. A name that starts
with a dot is offered only once you type the dot.

The `@` and the path stay in the prompt as you send it, because Claude Code
reads `@path` as a reference to a file.

- `Tab` grows the path. One match completes in full, and several grow the path
  as far as they agree. A directory gets a separator, so you can carry on into
  the next level. A file gets a space, because there is nothing more to
  complete.
- With nothing left to grow, `Tab` moves the focus, as it always did.
- `Shift+Tab` walks the names instead. It starts at the last name and steps
  backwards, it wraps around, and `Tab` steps the other way. The name you are
  on is marked under the box.
- Type anything to stop the walk.

Two limits:

- The completion runs only when the cursor sits at the end of the prompt.
- A name that holds a space goes in as it is, and the space breaks the word. A
  dropped path is quoted instead, because a drop is not one word.

## Running a bash command with `!`

Start a prompt with `!` to run a bash command. The command runs in the
directory of the selected session, because that is where the session works. The
session must run first, because the output goes into the session.

- `!command` runs the command and shows the output in the session. Claude does
  not see the output.
- `!!command` runs the command, then sends the command and its output to Claude
  as the next prompt. So the next turn can use the output.

The session shows the command on a header line, then one line for each line of
the output. If the command fails, the session adds the error on a last line.

The output of `!command` is a local record. A resume does not show it again,
because it is not part of Claude's transcript. The output of `!!command` reaches
Claude as the next prompt, so the normal prompt echo shows it once.

The command runs off the interface thread, so a slow command does not hold the
interface. The status bar shows `running command…` until the command stops.

## Dropping a file

Drag a file or a folder onto the window and its path goes into the prompt,
wherever the focus is. Drop it on the list or on the output, and the focus moves
to the prompt first. What you already typed is kept, and a space separates it
from the path.

The path is cleaned up as it goes in. A terminal escapes a space as `\ `, and
some send a `file://` URL. Both become a plain path, and a path that holds a
space is put in quotes:

```
/Users/dexter/my\ notes.md      ->   "/Users/dexter/my notes.md"
file:///Users/dexter/a%20b.md   ->   "/Users/dexter/a b.md"
```

Drop several files at once and each one is cleaned and separated by a space.

A drop is only rewritten when every part of it names a file or folder that
exists. Anything else is ordinary pasted text, and it goes in exactly as it is.

A drop onto the new session form replaces the directory field. Drop a file and
the field takes the folder that holds it.

### When a terminal does not mark the drop

A terminal normally wraps a drop as a bracketed paste, which arrives as one
piece. A long path is sometimes split, and then it arrives as ordinary key
presses instead. So a text fragment that holds a `/` also moves the focus to the
prompt, and the pieces gather there in order. A single `/`, and quick typing
with no `/` in it, stay as key presses for the list.

## Answering a question

A session asks the human a multiple-choice question with the `AskUserQuestion`
tool. When that tool arrives, the session interrupts its turn and waits, and its
sidebar row shows a `?`. See [sessions.md](../sessions.md) and
[protocol.md](../protocol.md).

The dialog belongs to the session that asked. It draws in the output pane of
that session only, in place of the output, and only when that session is
selected. So a question never moves the selection. You move between sessions
while a question waits, and each waiting session keeps its own dialog. The
dialog shows one question at a time, with its options and a text field.

The dialog takes keys only when its session is selected and the output pane has
the focus. So focus the output pane to answer.

- `↑` and `↓` move through the options and the text field.
- `Space` chooses the option under the cursor. A single-choice question keeps
  only the last option. A multi-choice question keeps every option you mark.
- Type in the text field to give a free answer next to the options, or in place
  of them.
- `Enter` sends the answer. With more than one question, it steps to the next
  one first.
- `Tab` and `Shift+Tab` move the focus out of the pane, so you can leave the
  dialog and switch sessions without an answer.
- `Esc` dismisses the dialog without an answer.

The answer goes back as the next prompt, one labelled line for each question,
because the child already closed the tool call. For example, a choice of `Blue`
with a note reads `Colour: Blue (a lighter shade)`. The answer moves the session
from `waiting` to `busy`, and the focus returns to the prompt box, so you type
the next prompt without a step through the panes.

The child answers the tool itself with an error, so that error and the model's
follow-up line both stay in the transcript. A second question that arrives while
the first one waits is not shown. The status bar notes it, and the human can ask
the session again.

## The new session form

The form asks for a directory, a name, a model, a permission mode, an effort
level, a control grant, and a first prompt. The directory comes first, because
it is the only required field. The first prompt is sent as soon as the session starts, and it
takes a `/preset` name. See [templates.md](../templates.md). It defaults to the
current directory. An empty name becomes the last element of the directory.

The control field takes `yes` to let the session prompt, stop, and archive the
other sessions. Anything else means no. See [mcp/grant.md](../mcp/grant.md)
before you set it, because such a session can stop the work you are reading.

The effort field is one of `low`, `medium`, `high`, `xhigh`, or `max`, or empty
for the Claude Code default. The form checks the level before it starts. You can
change the model, the mode, and the effort later with a key; see
[keys.md](keys.md).

The directory field completes as you type. The names that match appear under
it, and `Tab` grows the path:

- One match, and `Tab` finishes it and adds a separator, so you can carry on
  into the next level.
- Several matches, and `Tab` grows the path as far as they agree. Type one more
  letter and press it again.
- Nothing left to complete, and `Tab` moves to the next field as it always did.

`Shift+Tab` walks the list instead of completing it. It starts at the last name
and steps backwards, and it wraps around. While you walk, the name you are on
is marked under the field, and `Tab` steps the other way.

Type anything to stop walking. `Shift+Tab` then moves a field, as it does
everywhere else in the form.

`~` becomes your home directory. A directory whose name starts with a dot is
offered only once you type the dot.

The form checks that the directory exists before it starts anything, and it
shows the reason in place when the check fails.

The form opens by itself when the interface starts with nothing to show, because
that is the only useful first action. It opens once, at the start. Archiving
your last session does not bring it back.
