# Candidate 04 — lift each MCP tool body to a pure handler

Status: **in progress**. Scope C is chosen: the full sweep of all 86 handlers,
so the package holds one pattern.

Effort: `internal/mcp`. Worktree `mcp-pure-handlers`. Follows candidate 02, which
split `Sessions` into per-concept ports.

## Problem

Each tool's trim, validate, call, and format logic lives inside an
`sdk.AddTool` closure over `*Server`. So a unit test of that logic must stand up
the HTTP server, a token, a client, and a JSON-RPC round trip, then string-match
the result. The interface a test wants (input in, output out) is buried behind
five hops.

Every handler has the same shape, and the first return is always `nil`:

```go
sdk.AddTool(server, &sdk.Tool{Name: ToolAddLock, Description: "..."},
  func(_ context.Context, _ *sdk.CallToolRequest, in lockIn) (*sdk.CallToolResult, lockOut, error) {
    lock := strings.TrimSpace(in.Lock)
    if lock == "" { return nil, lockOut{}, ErrNoLock }
    locks, err := s.sessions.AddLock(lock, caller)
    ...
    return nil, lockOut{...}, nil
  })
```

## Decision — the seam

Lift each body to an unexported pure function that takes the narrow port it
needs (from candidate 02), the caller, and the input, and returns the output:

```go
func addLock(locks LockPort, caller string, in lockIn) (lockOut, error) {
    lock := strings.TrimSpace(in.Lock)
    if lock == "" { return lockOut{}, ErrNoLock }
    held, err := locks.AddLock(lock, caller)
    if err != nil { return lockOut{}, err }
    holders, err := otherHolders(locks, lock, caller)
    if err != nil { return lockOut{}, err }
    return lockOut{OK: true, Locks: held, Changed: true, Holders: holders,
        Message: addLockMessage(caller, lock, holders)}, nil
}
```

The `sdk.AddTool` closure shrinks to a one-line adapter that only unmarshals and
calls the pure function:

```go
sdk.AddTool(server, &sdk.Tool{Name: ToolAddLock, Description: "..."},
  func(_ context.Context, _ *sdk.CallToolRequest, in lockIn) (*sdk.CallToolResult, lockOut, error) {
    out, err := addLock(s.sessions, caller, in)
    return nil, out, err
  })
```

Helpers move with the logic: `otherHolders` becomes a free function that takes a
`LockPort`. A handler that needs the context (`Stop`) takes `ctx` as its first
parameter.

The pure function takes the **narrow port**, not `Sessions`. So a handler names
the slice of the manager it uses, and a test builds a fake of that slice. In
production, `s.sessions` (the composite) satisfies the narrow port unchanged.

## Decision — where the unit tests live

The pure functions are unexported, so their unit tests run in `package mcp`
(white-box internal test files, e.g. `tools_locks_pure_test.go`). They call the
pure function with a small fake of its narrow port, and assert on the returned
`out` and error. No server, no HTTP, no token.

The existing `package mcp_test` integration tests stay. They guard the behaviour
through the real server during the move, and they still cover tool registration
and the profile and grant gates, which the pure functions do not touch. So the
move keeps its safety net.

## Scope — the full sweep

The change is a pure move: the logic is identical, only its home changes. Every
handler across all 12 tool files becomes a pure function, so the package holds
one pattern: **locks** (6), **control** (5), **config** (18), **api** (11),
**client** (11), **peers** (8), **read** (8), **schedule** (7), **layout** (5),
**shares** (4), **usage** (2), **api_docs** (1).

Each pure function takes what it needs, not the whole surface:

- A handler over one session port takes that narrow port (from candidate 02),
  the caller, and the input. Most handlers are this shape.
- A handler that needs the context (`stop`) takes `ctx` first.
- A handler that reads server state rather than a session port (the client-facing
  API tools, the share admin that counts watchers, the API docs) takes the
  concrete dependency it needs, so it stays testable without the HTTP stack.

`shares` also carries a per-session control check, so its pure function returns
the refusal rather than the closure holding the guard.

New white-box unit tests prove the seam on **locks** and **control**, the two
groups with the most branching. The other groups keep their `package mcp_test`
integration tests, which still pass because the move changes no behaviour.

## Data flow

```
request → sdk closure: unmarshal in → pure handler(port, caller, in) → out → sdk marshals out
test    →                              pure handler(fakePort, "docs", in) → out (asserted)
```

Nothing else changes: the same port methods run, the same messages format, the
same errors return.

## Verification

- `go test ./internal/mcp/...` stays green (the integration tests are unchanged).
- New white-box tests call `addLock`, `send`, and the rest directly and assert
  the `out`. A lock test builds a `LockPort` fake of six methods; a control test
  builds a `ControlPort` fake of the methods it calls.
- `just check` from the worktree.

## When it lands

The pure-handler pattern is code, not a doc subject, so the durable note is one
line in `CONTEXT.md` under the ports: a tool body is a pure function of its port,
and its test calls it directly. The full sweep leaves nothing deferred, so delete
the plan. Repoint nothing; no code named this plan.
