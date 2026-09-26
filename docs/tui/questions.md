# Answering a question

A session asks the human a multiple-choice question with the `AskUserQuestion`
tool. When that tool arrives, the session interrupts its turn and waits. Its
sidebar row shows `waiting` in the info colour, and the band at the top counts
it. See [sessions.md](../sessions.md) and
[protocol.md](../protocol.md).

The dialog belongs to the session that asked. It draws in the output pane of
that session only, in place of the output, and only when that session is
selected. It draws at the top of the pane, not in a box: a rule that holds its
label (`A QUESTION FOR YOU`, or `QUESTION 1 OF 2`), the question, a line that
says where the answer goes, the options, an `ANSWER` field in `[ ]` brackets, and
the keys in brackets (`[space] choose`). A short option shows its description on
the same row, after the label; a long one wraps it under the label. The option
under the cursor inverts, with its description. So a question never moves the selection. You move between sessions
while a question waits, and each waiting session keeps its own dialog. The
dialog shows one question at a time, with its options and a text field.

The dialog takes keys only when its session is selected and the output pane has
the focus. So focus the output pane to answer.

- `↑` and `↓` move through the options and the text field.
- `Space` chooses the option under the cursor. A single-choice question keeps
  only the last option. A multi-choice question keeps every option you mark.
- Type in the text field to give a free answer next to the options, or in place
  of them. A paste into the text field goes into the field, not the prompt.
- `Enter` sends the answer. With more than one question, it steps to the next
  one first.
- `Tab` and `Shift+Tab` move the focus out of the pane, so you can leave the
  dialog and switch sessions without an answer.
- `Esc` dismisses the dialog without an answer.

A long option label or a long option description wraps across lines. The dialog
caps each at a small number of lines and draws a marker for the rest. The option
under the cursor draws in full, so you read the rest by moving to it. The
`question_option` and `question_description` caps set the line count, and default
to 2. See [../config.md](../config.md).

The answer goes back as the next prompt, one labelled line for each question,
because the child already closed the tool call. For example, a choice of `Blue`
with a note reads `Colour: Blue (a lighter shade)`. The answer moves the session
from `waiting` to `busy`, and the focus returns to the prompt box, so you type
the next prompt without a step through the panes.

The child answers the tool itself with an error, so that error and the model's
follow-up line both stay in the transcript. A second question that arrives while
the first one waits is not shown. The status bar notes it, and the human can ask
the session again.
