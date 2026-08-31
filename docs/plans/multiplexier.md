# multiplexier — what is still ahead

**Status:** the multiplexer works, and it is verified against Claude Code
2.1.176. What it does is described in [protocol.md](../protocol.md),
[sessions.md](../sessions.md), [manager.md](../manager.md),
[tui.md](../tui.md), [templates.md](../templates.md), and
[markdown.md](../markdown.md).

This file holds only the work that is not built. Delete it when nothing is left.

---

## Control

- **Interrupt.** A key to stop a turn that is running, without stopping the
  session. Two mechanisms exist. The first is a `control_request` line on
  stdin. The second is a `SIGINT` signal to the child. The control protocol is
  not a stable public interface, so this work must first prove what the
  installed version accepts, and fall back to `SIGINT`, and then to a restart
  with `--resume`.
- **A configuration file** at `~/.multiplexier/config.json`, to hold the default
  model, the default permission mode, and the state directory. Every one of
  those is a flag today, and a flag is the wrong place for a setting you never
  change.

---

## Optional, after a review

- **A permission prompt for each request.** `--permission-prompt-tool
  mcp__mux__approve` points at an MCP tool that the multiplexer serves. Each
  request then appears in the interface, and the human answers with a key
  press. It needs an MCP server inside the multiplexer, which is why it waits.
- **A task router.** A queue, and dispatch to the first free session. This is a
  second product rather than a feature, so it needs its own plan.

---

## Open question

**Session directories.** Every session takes a directory that already exists.
Should the multiplexer instead make a Git worktree for each session, so two
agents in one repository cannot tread on each other? That would tie the tool to
Git, and it would need a rule for removing a worktree when a session ends.
