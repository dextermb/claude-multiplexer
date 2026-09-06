# Plan: split the long internal files

Status: draft, awaiting go-ahead.

The task started as one ask: split `internal/mcp/tools.go` into smaller files.
This plan answers that ask, and it lists the other files that grew long, in the
order they need the work.

## The files, by size

A count of the non-test Go files (lines):

| Lines | File | Note |
|---|---|---|
| 2664 | `internal/tui/app.go` | The standout. Five times the size of `tools.go`. |
| 795 | `internal/session/session.go` | The process, its I/O, its state machine, all in one file. |
| 783 | `internal/manager/manager.go` | The registry, the lifecycle, the store, the queries. |
| 700 | `internal/tui/diff.go` | Already a split-out feature file. Borderline. |
| 671 | `internal/manager/mcp.go` | The `bridge` type plus the Manager MCP methods. |
| 520 | `internal/mcp/tools.go` | The original ask. |

Go sets no line cap (the 300-line cap in `documentation.md` is for docs). So the
guide here is one file, one subject. A reader who wants the key map must not
scroll past the layout code to reach it.

## The order of work

Each file is one effort, so each file gets its own worktree (see
`worktrees.md`). Do them in this order, largest pain first:

1. `internal/tui/app.go`
2. `internal/mcp/tools.go` (the ask)
3. `internal/session/session.go`
4. `internal/manager/manager.go`
5. `internal/manager/mcp.go`
6. `internal/tui/diff.go` (optional)

Every split moves functions between files in the same package. No name changes,
no export changes, no behaviour change. So each effort is mechanical and low
risk, and `go test ./...` in the worktree is the check that it stayed correct.

## Open question — the package boundary for the MCP tools

The ask names `internal/mcp/tools/*.go`. A subdirectory is a new package
(`tools`). The `build` method reads `Server` internals (`s.sessions`), the
package error values (`ErrNoEditor`, `ErrBadCap`, and the rest), and the message
helpers. A new package needs all of these exported, so a subdirectory turns a
mechanical move into an API change.

**Recommendation: keep the package `mcp`, and split `tools.go` into more files in
the same directory.** Go puts no weight on the file count in a package, and the
split gets the same result — one subject per file — with no exports. The reader
opens `internal/mcp/` and sees `tools_session.go`, `tools_config.go`, and the
rest.

Resolve this before the `tools.go` effort starts. The file plan below assumes the
recommendation.

---

## 1. `internal/tui/app.go` (2664 → ~7 files)

All one package `tui`, so this is a pure move. The `Model` struct stays in
`app.go`, because every file reads it. Group the methods by subject:

| New file | Holds | Functions (by prefix) |
|---|---|---|
| `app.go` | The core. | `Model`, `Options`, msg types, `New`, `Run`, `Init`, `Update`, `update`, `resize`, the ticks, the `*Cmd` helpers. |
| `events.go` | The message handlers. | `handleEvent`, `handleNotice`, `handleStored`, `handleSettings`, `handleSpawned`, `handleSpin`, `maybeAskQuestion`, `dropQueued`. |
| `input.go` | The key handlers. | `handleKey`, `handlePaste`, `promptKey`, `historyKey`, `recordHistory`, `jobsModalKey`, `outputKey`, `sidebarKey`, `choiceKey`, `renameKey`, `pickerKey`, `fieldsKey`, `looksDropped`. |
| `mouse.go` | The pointer and the text selection. | `handleMouse`, `handleLeftMouse`, `blockAtRow`, `outputPos`, `clearSelection`, `copySelection`, `highlight`. |
| `list.go` | The session list, the groups, the folds. | `buildLines`, `selLine`, `selGroup`, `selectFirst`, `selectNear`, `setFold`, `applyFolds`, `move`, `toggleFold`, `foldOthers`, `unfoldAll`, `setAllFolds`, `rowGroup`, `groupKey`, `selectedRow`, `selIndex`. |
| `blocks.go` | The output blocks and the block cursor. | `rebuildOutput`, `appendOutput`, `clearBlocks`, `redrawBlocks`, `drawBlocks`, `setBlockCursor`, `resetBlockCursor`, `moveBlockCursor`, `showBlock`, `toggleBlock`, `setPartial`, `setContent`, `rowCount`. |
| `layout.go` | The view and the geometry. | `View`, `paneView`, `sidebarView`, `outputView`, `sidePanelView`, `barView`, `statusView`, `promptView`, `confirmView`, and the width/height helpers (`leftWidth`, `outputWidth`, `bodyHeight`, and the rest). |

The action methods that are neither an event nor a key (for example `send`,
`stopBusy`, `dispatch`, `interrupt`, `startQuit`, `openNewForm`, `resumeSelected`)
go into an `actions.go`, or they stay in `app.go` if the file stays small enough.
Settle the last few by size at merge time.

## 2. `internal/mcp/tools.go` (520 → ~6 files, package `mcp`)

Split `build` into four register helpers, one per tool group, each in its own
file. `build` becomes the assembler that calls them:

```
func (s *Server) build(caller string, control bool) *sdk.Server {
    server := sdk.NewServer(...)
    s.addReadTools(server, caller)
    s.addConfigTools(server, caller)
    s.addJobTools(server, caller)
    if control {
        s.addControlTools(server, caller)
    }
    return server
}
```

| New file | Holds |
|---|---|
| `tools.go` | `build`, and the shared helper `targetOrSelf`. |
| `tools_types.go` | Every `...In` and `...Out` struct. |
| `tools_read.go` | `addReadTools`: `rename`, `list`, `messages`. |
| `tools_config.go` | `addConfigTools`: the editor, block-cap, working-directory, config-path, and template-path tools, plus the message helpers (`editorMessage`, `blockCapMessage`, `blockCapClearedMessage`, `clearedMessage`). |
| `tools_jobs.go` | `addJobTools`: `list_jobs`, `stop_job`. |
| `tools_control.go` | `addControlTools`: `send`, `stop`, `archive`, `create` (the `control` group). |

## 3. `internal/session/session.go` (795 → ~5 files, package `session`)

| New file | Holds |
|---|---|
| `session.go` | `Config`, `Session`, `Snapshot`, `Event`, `New`, `Start`, `Wait`, `Stop`. |
| `io.go` | `readStdout`, `readStderr`, `writeLoop`, `closeStdin`, `waitIdle`. |
| `apply.go` | `apply`, `applyJobBlocks`, `applyLaunchResult`, `applyTask`, `setJobStatus`. |
| `state.go` | `setState`, `setStateIf`, `afterState`, `markSent`, `fail`, `supervise`, `emit`. |
| `control.go` | `Send`, `Interrupt`, `DiscardQueued`, `SetModel`, `SetPermissionMode`, `SetTitle`, `controlID`. |

`jobs.go`, `joboutput.go`, and `queue.go` already sit apart, so leave them.

## 4. `internal/manager/manager.go` (783 → ~4 files, package `manager`)

| New file | Holds |
|---|---|
| `manager.go` | `Options`, `Spec`, `Event`, `Manager`, `New`, `pump`, `trackPartial`, `trackTodos`. |
| `entry.go` | The `entry` type and its methods (`metaCopy`, `setMeta`, `partialText`, `setSnapshot`, `view`, `todoList`), and `totals`. |
| `lifecycle.go` | `Spawn`, `Resume`, `ResumeWithEffort`, `Stop`, `Remove`, `Shutdown`, `Send`, `Interrupt`, and the setters (`SetModel`, `SetPermissionMode`, `SetWorkingDir`, `UnsetWorkingDir`, `SetTitle`). |
| `store.go` | `Stored`, `Replay`, `Archive`, `Meta`, `rememberSession`, `tasksFromTranscript`, and the query methods (`Names`, `Snapshots`, `Snapshot`, `Lines`, `AppendLines`, `Stderr`, `Todos`, `TotalCost`). |

`bus.go` and `meta.go` already sit apart, so leave them.

## 5. `internal/manager/mcp.go` (671 → 2 files, package `manager`)

The file holds two subjects: the Manager MCP wiring, and the `bridge` type that
carries out the tool calls.

| New file | Holds |
|---|---|
| `mcp.go` | `StartMCP`, `MCPURL`, `equipTools`, `releaseTools`, and the Manager methods the bridge calls (`SendFrom`, `Jobs`, `StopJobFrom`, `List`, `Messages`, `SetEditor`, `SetBlockCap`, and the rest), plus `findJob`, `killPrompt`, `messageOf`, `assistantText`. |
| `bridge.go` | The `bridge` type and every method on it, plus the notice helpers (`blockCapNotice`, `blockCapClearedNotice`, `clearedNotice`, `editorNotice`). |

## 6. `internal/tui/diff.go` (700 → 2 files, optional, package `tui`)

This file is already a split-out feature, so it is the lowest priority. Split it
only if it keeps growing:

| New file | Holds |
|---|---|
| `diff.go` | The state, the commands, the key handling, the navigation (`diffKey`, `diffCursorUp`/`Down`, `diffJumpUp`/`Down`, the toggles, the resize). |
| `diffview.go` | The rendering (`diffPanelView`, `diffPanelLines`, `diffFileRow`, `diffFileBody`, `renderDiffBody`, `wrapHard`, `padLeft`, `isDiffHeader`, `hunkNewStart`). |

---

## Verification

For each effort, in its worktree:

1. `go build ./...` compiles.
2. `go test ./...` stays green.
3. `git diff --stat` shows only moved lines, no changed logic.

No document points at these files by path, so no pointer needs a repoint. When
every effort lands, delete this plan (see `plans.md`).
