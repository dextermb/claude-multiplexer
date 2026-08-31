# Documentation

Documentation lives in `docs/*.md`. It is the long form: the workflow, the
dataflow, the reason a shape was chosen. Code carries the short form, and points
here when a sentence is not enough. See [no-comments.md](./no-comments.md).

# Why

The code says what happens now. It cannot say what was tried, what a provider
refuses to do, or which of two correct-looking orders is the one that works.
That knowledge has no other home, and the next session starts with none of it.

A document is also the only thing a reader can find without knowing where to
look. A comment answers a question you already had. A document answers a
question you did not know to ask.

# One document per area, named for the reader

The file name is what someone types when they want an answer. Name it for the
subject, not for the module:

- `docs/database.md`, `docs/feed.md`, `docs/storage.md` — good.
- `docs/lib-storage-r2.md` — bad. It names the code, so it dies at the next
  rename.

Use kebab-case, and one file per area. Two files on the same subject drift
apart, and a reader believes whichever one they find first.

# Progressive disclosure

Start with one file. Split it when it gets long or covers subjects a reader
would not arrive with at the same time.

**300 lines is the cap.** No document may pass it. A plan is exempt, because a
plan is thinking rather than a specification.

**The signals to split, before the cap is reached:**

- The file passes roughly 250 lines.
- Two readers with different problems both have to scroll past the other's
  half.
- A new section is about to add a third subject.

250 is a judgement and 300 is a gate. Reach the first and consider a split.
Reach the second and the build fails.

**How to split.** Keep the original path as the entry point and move the detail
beneath it:

```
docs/storage.md                 index: what each page covers, and when to read it
  |
  +-- docs/storage/r2.md
  +-- docs/storage/providers.md
  +-- docs/storage/files.md
```

The entry point keeps its name. Every pointer that already names
`docs/storage.md` stays correct, and the reader lands on a map rather than a
wall. **Do not** move it to `docs/storage/index.md` — that reads more neatly and
breaks every pointer in the code.

The index is short. It says what each page covers and why you would open it. It
does not summarize the pages, because a summary is a second copy that goes
stale.

**The index names every page in its folder.** A page it does not name is a page
nobody finds, and no link breaks to tell you. Add a new page to its index in the
same change.

**A moved section takes its links with it.** A page one folder deeper reaches
`docs/testing.md` as `../testing.md`, and a link into a section that has moved
must name the new file.

# A pointer and its document land together

A comment may name a document only if that document exists in the same change.
This is not a style preference. Under `no-comments.md` a pointer **replaces**
the explanation, so a pointer to a file that is not there leaves the reader with
a file name and nothing else.

If the document is too large to write now, write the explanation in the code
instead, or write a smaller document. Do not write the pointer and mean to
follow it up.

The same applies in reverse. When a document moves or is split, every pointer
that names it must still resolve.

# Keep it current or delete it

A document that describes code which no longer exists is worse than no document,
because a reader trusts it. When a change makes a section wrong, fix the section
in that change.

A section nobody can make true any more gets deleted. The git history holds it.

# Never point at a plan

`docs/plans/*.md` holds work that is not built yet — the decisions, the open
questions, the shape being agreed. `docs/*.md` holds what is true today, and is
what to follow.

```
code   ──▶  docs/*.md          yes
docs   ──▶  docs/*.md          yes

docs   ──▶  docs/plans/*.md    never
code   ──▶  docs/plans/*.md    never
```

A plan records options that were rejected and questions that were open. A reader
who follows a link into one cannot tell those apart from what shipped. A
specification that points at a plan is no longer a specification.

So a missing document is never resolved by pointing at its plan. Write the
document. When a plan lands, its durable parts move here and anything that named
the plan is repointed. See [plans.md](./plans.md).
