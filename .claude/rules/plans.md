# Plans

Non-trivial work gets a written plan **before** implementation — anything that
adds or reshapes database tables, spans several files/layers, changes a workflow,
or has decisions worth settling with the user first. Small, obvious changes do
not need one; when in doubt, a plan is cheap insurance against building the wrong
thing.

Plans live in `docs/plans/<topic>.md`, one file per effort, kebab-case name. See
`docs/plans/multiplexier.md` for the shape.

**Do not** start implementing your plan without confirming with the user.

# Why

A plan is where the thinking happens in the open: the user approves the approach,
the decisions and their reasons are recorded, and a later session (or a fresh
agent) can pick up the intent without re-deriving it. The plan is the source of
truth until the code lands; keep it current as decisions are made.

# What a plan contains

Not every section every time, but reach for these:

1. **Status** — one line at the top: `draft`, `awaiting go-ahead`, `in progress`,
   `done`. State what is decided vs. still open.

2. **Decisions** — what was chosen and, briefly, _why_. Record the rejected
   option when the choice was close; it stops the same debate reopening later.

3. **Open questions** — list them explicitly and resolve them before building.
   Move each into Decisions (or a "Resolved" list) once answered rather than
   deleting it, so the reasoning survives.

4. **ERD** — for any schema change, an ASCII entity–relationship diagram in a
   fenced block: every table with its columns, `PK`/`FK` markers, cardinality
   (`1:1` / `1:N`), and which columns/tables are new. Follow it with a plain
   relationship summary. Mark FK delete behaviour (e.g. `ON DELETE CASCADE`).

5. **Data flows** — how data moves through the change: the key read/write paths,
   what is stored vs. derived, and any ordering that matters (e.g. render →
   denormalize → confirm). A short sequence or bullet trace beats prose.

6. **Workflows** — the user-facing steps and the states/guards between them
   (e.g. a confirm gate, an empty state, a loading state), so the UX is agreed
   before it is built.

7. **The build** — schema/DDL sketch, server actions or API, app wiring,
   migration/backfill strategy, and how it will be verified.

# A plan is never the specification

`docs/*.md` is the specification. `docs/plans/*.md` is the thinking that came
before it. Links run one way only:

```
code   ──▶  docs/*.md          yes
docs   ──▶  docs/*.md          yes
plans  ──▶  docs/*.md          yes

docs   ──▶  docs/plans/*.md    never
code   ──▶  docs/plans/*.md    never
```

A plan holds rejected options, open questions, and a shape that may not be the
shape that shipped. A reader who follows a link into one cannot tell what was
decided from what was merely considered. So a specification that points at a
plan inherits the plan's uncertainty, and stops being a specification.

Never resolve a missing document by pointing at the plan that describes it.
Write the document.

# When the code lands

A plan describes work that does not exist yet. The moment it does exist, the
plan starts to lie, because the code is now the truth and the plan is a memory
of what was intended.

So when a plan ships, in the same change:

1. **Move what stays true into `docs/*.md`.** The dataflow, the workflow, the
   reason a shape was chosen — a reader needs these, and `docs/plans/` is not
   where they will look. See [documentation.md](./documentation.md).
2. **Repoint anything that named the plan** at the document that now holds the
   content.
3. **Leave in the plan only what is still ahead**, and say so in the status
   line.
4. **Delete the plan once nothing is ahead.** The git history keeps it.

A plan marked `in progress` for months is the failure this prevents. It holds
knowledge the specification needs, in a directory that is not the specification.

# Style

- Concrete over vague: real table and column names, real action signatures.
- Keep diagrams in fenced code blocks so they render monospace.
- Prefer diagrams and short traces to long paragraphs.
- When the plan and the code diverge after implementation, update the plan or
  delete it — a stale plan is worse than none.
