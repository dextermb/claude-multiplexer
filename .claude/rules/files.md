# Keep files short and single purpose

One file, one purpose. A file holds code that changes for one reason, and a
reader opens it for one question. When a file takes on a second purpose, split
it — where the split is clean.

# The signal to split

- The file passes roughly 250 lines.
- Two readers with different questions both scroll past the other's half.
- A new section adds a third subject.
- The name no longer covers everything inside (an `mcp.go` that also writes
  settings and reads transcripts).

Length alone is not the trigger. A single cohesive type, one struct and its
methods, may run long and stay one purpose. Do not split it to hit a number.

# How to split

- Split by purpose, not by size. Move each concern to its own file.
- Name each file for the concern a reader looks for, not for the module:
  `messages.go`, `settings.go`, `jobs.go`.
- Follow the pattern the package already uses. A tool set splits into
  `tools_*.go`. An adapter splits into `bridge_*.go`. Match it.
- Keep the entry point. The original file keeps its name and its core, and the
  parts move beside it.
- A pure move changes no behaviour. Split in its own commit, and change the code
  in another.

# Why

The name of a file is the first thing a reader searches. A file that does one
thing answers by its name alone. A file that does five hides each one, so the
reader reads all five to find the one. See [documentation.md](./documentation.md)
for the same rule applied to `docs/*.md`.
