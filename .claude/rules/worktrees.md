# Worktrees

Do each piece of work in its own git worktree. Start the worktree when the work
starts, and collapse it back into the main worktree when the work is complete.
One effort, one worktree.

# Why

Two efforts in one worktree share one set of files, so they share one test
binary. When both efforts add a test, or a type, or a package-level name, the two
sets collide, and the shared build stops compiling. Neither effort is wrong, but
neither can run its tests, because the other effort's half-finished code sits in
the same package.

A worktree gives each effort its own copy of the files on its own branch. The
tests of one effort cannot collide with the tests of another, because the two
never share a build. So each effort runs green on its own, and the collision only
has to be settled once, at the merge.

# When to start a worktree

Start a worktree for any effort that adds or changes code and runs tests:

- a feature,
- a fix that spans more than one file,
- any work that lands next to other work in progress.

A one-line edit that you commit at once does not need a worktree. When in doubt,
start one. A worktree is cheap, and a broken shared build is not.

# The shape of the work

1. **Branch.** Create a worktree on a new branch, off the main branch, named for
   the effort.
2. **Build.** Do all of the work in that worktree. Write the code, write the
   tests, and update the docs, the same as any change.
3. **Green.** Run the tests in the worktree. The feature is complete only when
   the tests pass and the docs are current. See [plans.md](./plans.md) and
   [documentation.md](./documentation.md).
4. **Merge.** Bring the branch back into the main worktree. Settle any collision
   with other work here, at the merge, and run the tests once more in the main
   worktree to confirm the merged result is green.
5. **Collapse.** Remove the worktree and delete its branch once the merge is in.
   Leave no empty worktree behind.

# Rules

1. **Do not run another effort's tests to judge your own.** A red test from
   half-finished work in the same worktree is not your failure. If the shared
   build will not compile because of work that is not yours, that is the signal
   to move your work into its own worktree, not to fix the other work.
2. **Merge only green.** Do not merge a worktree whose tests fail. The main
   worktree is the shared truth, and a red merge breaks it for everyone.
3. **Collapse when complete.** A worktree that outlives its work is a second copy
   of the repository that drifts from the main worktree. Remove it in the same
   step as the merge.
4. **One effort per worktree.** Do not do a second, unrelated effort in a
   worktree that already holds one. That rebuilds the collision the worktree
   exists to prevent.
