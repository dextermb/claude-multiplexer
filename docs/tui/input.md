# Typing, dropping, and starting a session

The prompt box, a file dropped on the window, and the new session form.

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
tool. When that tool arrives, a dialog opens over the interface and the session
that asked becomes the selected one. The dialog shows one question at a time,
with its options and a text field.

- `↑` and `↓` move through the options and the text field.
- `Space` chooses the option under the cursor. A single-choice question keeps
  only the last option. A multi-choice question keeps every option you mark.
- Type in the text field to give a free answer next to the options, or in place
  of them.
- `Enter` sends the answer. With more than one question, it steps to the next
  one first.
- `Esc` dismisses the dialog without an answer.

The answer goes back as the next prompt, one labelled line for each question,
because the child already closed the tool call. See
[protocol.md](../protocol.md). For example, a choice of `Blue` with a note reads
`Colour: Blue (a lighter shade)`.

The child answers the tool itself with an error, so that error and the model's
follow-up line both stay in the transcript. A second question that arrives while
the dialog is open is not shown. The status bar notes it, and the human can ask
the session again.

## The new session form

The form asks for a directory, a name, a model, a permission mode, and a first
prompt. The directory comes first, because it is the only required field. The
first prompt is sent as soon as the session starts, and it takes a `/preset`
name. See [templates.md](../templates.md). It defaults to the
current directory. An empty name becomes the last element of the directory.

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
