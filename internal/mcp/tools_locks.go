package mcp

import (
	"context"
	"strconv"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) addLockTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolListLocks,
		Description: "The locks a session holds, in the order it took them. " +
			"A lock is a label that says the session works on something, so another session keeps away. " +
			"Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listLocksIn) (*sdk.CallToolResult, lockOut, error) {
		target, err := targetOrSelf(in.Session, caller)
		if err != nil {
			return nil, lockOut{}, err
		}
		locks, err := s.sessions.Locks(target)
		if err != nil {
			return nil, lockOut{}, err
		}
		return nil, lockOut{OK: true, Locks: locks, Message: lockListMessage(target, locks)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolAddLock,
		Description: "Take one lock on this session, to say it works on something and another session must keep away. " +
			"A lock is advisory: it always succeeds, and the result names the other sessions that hold the same label. " +
			"Search with " + ToolFindLocked + " first, and release the lock with " + ToolRemoveLock + " when the work is done.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in lockIn) (*sdk.CallToolResult, lockOut, error) {
		lock := strings.TrimSpace(in.Lock)
		if lock == "" {
			return nil, lockOut{}, ErrNoLock
		}
		locks, err := s.sessions.AddLock(lock, caller)
		if err != nil {
			return nil, lockOut{}, err
		}
		holders, err := s.otherHolders(lock, caller)
		if err != nil {
			return nil, lockOut{}, err
		}
		return nil, lockOut{OK: true, Locks: locks, Changed: true, Holders: holders,
			Message: addLockMessage(caller, lock, holders)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolRemoveLock,
		Description: "Release one lock this session holds, so another session can take it. " +
			"Call it when the work the lock covers is done.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in lockIn) (*sdk.CallToolResult, lockOut, error) {
		lock := strings.TrimSpace(in.Lock)
		if lock == "" {
			return nil, lockOut{}, ErrNoLock
		}
		locks, err := s.sessions.RemoveLock(lock, caller)
		if err != nil {
			return nil, lockOut{}, err
		}
		return nil, lockOut{OK: true, Locks: locks, Changed: true,
			Message: caller + " released " + lock + " and " + lockCount(locks)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetLocks,
		Description: "Replace the whole set of locks this session holds. " +
			"Give every label the session works under. An empty list releases every lock.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setLocksIn) (*sdk.CallToolResult, lockOut, error) {
		locks, err := s.sessions.SetLocks(in.Locks, caller)
		if err != nil {
			return nil, lockOut{}, err
		}
		return nil, lockOut{OK: true, Locks: locks, Changed: true,
			Message: caller + " " + lockCount(locks)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolClearLocks,
		Description: "Release every lock this session holds. " +
			"A lock is never released on its own, so call this before the session stops.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, lockOut, error) {
		changed, err := s.sessions.ClearLocks(caller)
		if err != nil {
			return nil, lockOut{}, err
		}
		message := caller + " held no lock"
		if changed {
			message = caller + " released every lock it held"
		}
		return nil, lockOut{OK: true, Changed: changed, Message: message}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolFindLocked,
		Description: "The sessions that hold every one of the named locks. " +
			"Call it before work another session could also do, to find out who holds the lock. " +
			"A stopped session keeps its locks, so set live true to search only the sessions that run now.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in findLockedIn) (*sdk.CallToolResult, listOut, error) {
		found, err := s.sessions.FindLocked(in.Locks, in.Live)
		if err != nil {
			return nil, listOut{}, err
		}
		return nil, listOut{Sessions: found}, nil
	})
}

// otherHolders names the sessions besides the caller that hold one lock.
func (s *Server) otherHolders(lock, caller string) ([]string, error) {
	found, err := s.sessions.FindLocked([]string{lock}, false)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, item := range found {
		if item.Name != caller {
			out = append(out, item.Name)
		}
	}
	return out, nil
}

func addLockMessage(caller, lock string, holders []string) string {
	if len(holders) == 0 {
		return caller + " holds " + lock + ", and no other session holds it"
	}
	return caller + " holds " + lock + ", and so do " + strings.Join(holders, ", ")
}

func lockListMessage(session string, locks []string) string {
	if len(locks) == 0 {
		return session + " holds no lock"
	}
	return session + " " + lockCount(locks)
}

func lockCount(locks []string) string {
	if len(locks) == 0 {
		return "holds no lock now"
	}
	if len(locks) == 1 {
		return "holds 1 lock now"
	}
	return "holds " + strconv.Itoa(len(locks)) + " locks now"
}
