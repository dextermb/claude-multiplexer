# Candidate 02 — split the `Sessions` seam into per-concept ports

Status: **in progress**. Scope A is chosen: split the interface and prove each
port with a narrow test fake, no handler rewrite.

Effort: `internal/mcp`. Worktree `mcp-sessions-ports`.

## Problem

The seam between the tools and the manager is one interface with ~80 methods
(`mcp.go:391–473`). Every caller and every test learns the whole surface to use
any slice of it:

- `NewServer(sessions Sessions)` takes the full interface.
- The test fake `fakeSessions` (server_test.go, ~615 lines) implements all ~80
  methods, even a test that exercises six lock tools.

The interface is as wide as the manager it fronts. The methods already cluster
by concept — read, control, config, locks, layouts, schedules, api, peers,
shares — but no name marks the clusters, so a reader finds them only by reading
every line.

## Decision — embedded per-concept ports

Define one interface per concept, and make `Sessions` a composite that embeds
them. The manager satisfies `Sessions` exactly as today, so no production wiring
changes.

```go
type SessionReader interface {
    List() []Session
    Messages(name string, limit int) ([]Message, error)
    Jobs(name string) ([]Job, error)
    // ...
}

type LockPort interface {
    Locks(session string) ([]string, error)
    SetLocks(labels []string, by string) ([]string, error)
    AddLock(label, by string) ([]string, error)
    RemoveLock(label, by string) ([]string, error)
    ClearLocks(by string) (bool, error)
    FindLocked(labels []string, live bool) ([]Session, error)
}

// SchedulePort, ConfigPort, ControlPort, APIPort, PeerPort, SharePort, ...

type Sessions interface {
    SessionReader
    ControlPort
    ConfigPort
    LockPort
    LayoutPort
    SchedulePort
    APIPort
    PeerPort
    SharePort
}
```

The port names are the concept names. They belong in `CONTEXT.md`, which does
not exist yet — this effort creates it and records the ports as the domain terms
for the manager slice each tool group needs.

## Decision — prove each port with a narrow fake

A split interface with one adapter is a hypothetical seam. The second adapter is
the test fake. So each tool group's test builds a **narrow** fake that embeds its
port and implements only its slice. An unimplemented method of the embedded
interface panics, which a focused test never calls:

```go
type fakeLocks struct {
    LockPort            // embedded, nil; only the lock methods are set
    locks map[string][]string
}
func (f *fakeLocks) AddLock(label, by string) ([]string, error) { /* ... */ }
// the other ~74 methods do not exist here
```

Two adapters make the port a real seam: the manager in production, the narrow
fake in tests. A lock test now fakes six methods, not eighty.

## Scope

In: define the ports, redefine `Sessions` as their composite, create
`CONTEXT.md` with the port glossary, and rewrite the lock-tool test (and the
other tool-group tests that already stand alone) onto narrow fakes.

Out: changing `NewServer` or the `addXxxTools` signatures to hold narrow ports
instead of `s.sessions`. The tool groups keep reaching through `s.sessions`
(the composite) for now. Narrowing the handler signatures is candidate 04's
work, and this effort sets it up.

## Open questions

1. **Port boundaries.** A first cut: `SessionReader`, `ControlPort`,
   `ConfigPort`, `LockPort`, `LayoutPort`, `SchedulePort`, `APIPort`, `PeerPort`,
   `SharePort`. `ConfigPath`/`TemplatePath`/`SchedulePath`/`APIEndpoint`/
   `PeerEndpoint` are path reads — fold each into the port of its concept.

## Verification

- `go test ./internal/mcp/...` stays green. The composite is structurally the
  same interface, so the manager still satisfies it with no change.
- The rewritten lock-tool test compiles against `LockPort` only, which proves
  the narrow fake.
- `go build ./...` from the main worktree at the merge.

## When it lands

`CONTEXT.md` holds the port glossary as the durable output. Move the "what each
port covers" note there, repoint nothing (no code named this plan), and delete
this plan. If scope B is deferred, the plan's follow-up note moves to candidate
04's card, not to a doc.
