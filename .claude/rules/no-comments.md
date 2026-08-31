# No Comments

Do not add comments to code unless requested. If you feel like something _should_ have code comments then mention it to the user, then they can decide if you should add them.

The default is **no comment**. Readable code is the explanation; a comment is an
admission that the code could not carry the meaning on its own.

# The bar

A comment has to earn its place. Before writing one, ask which of these it is:

- **Restates the code.** Delete it. If a reader who knows the language and the
  libraries can get there from the code, the comment is noise.
- **Names something that should have been named.** Delete it and rename the
  variable, extract the function, or split the expression instead.
- **Explains _why_, where the reason is genuinely not visible.** This is the only
  comment worth keeping — a constraint from elsewhere in the system, a
  workaround for a library's behaviour, a deliberate choice that looks wrong
  until you know the reason.

"Someone might not know how this library works" is not a reason. "This looks
like a bug but isn't, because X" is.

# Rules

When given the all clear to write comments refer to the rules below:

1. Keep it concise

- Code comments should **not** be any longer than one sentence.

2. Current state only

- Code comments should **not** describe the past state of code, only write comments about the _current_ state.

3. Complex explanations belong in `docs/*.md`

- If it takes more than one sentence, it is documentation, not a comment.
- Write it up in `docs/*.md` — expand on workflows/dataflows for later sessions
  — and leave at most a one-line pointer at the code, e.g.
  `// Full refresh every 8 partial updates, to clear ghosting; see docs/hardware/panel.md.`
- Do not repeat in the code what the doc already says. The pointer replaces the
  explanation, it does not summarise it.

4. Do not narrate the diff

- Comments are for whoever reads the file next, not for reviewing the change.
- Never write a comment whose purpose is to point out what you just did or why
  you chose it over an alternative. Put that in the response to the user.

5. Do not name a ticket

- The identifier is already on the commit, so the git history answers *why this
  line changed* better than a comment can.
- A comment that names an issue identifier sends the reader to a tracker
  instead of to the reason.

# Pointers

The pointer is one sentence ending in the path, exactly as the rest of the
codebase writes it:

```c
/* The single entry point every panel refresh goes through; see docs/hardware/panel.md. */
```

Not a bare path on its own line. One form, used everywhere, so it can be found
by search.

**The document must exist in the same change as the pointer.** A pointer
replaces the explanation, so one that names a missing file leaves the reader
with a file name and nothing else. If the document is too big to write now,
write a smaller one, or keep the explanation in the code.

The same holds when a document is renamed or split: every pointer that named it
must still resolve. See [documentation.md](./documentation.md).
