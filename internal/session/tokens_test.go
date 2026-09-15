package session

import (
	"context"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func resultWithUsage(u *protocol.Usage) protocol.Event {
	return protocol.Event{
		Type: protocol.TypeResult,
		Result: &protocol.Result{
			Subtype:  "success",
			NumTurns: 1,
			Usage:    u,
		},
	}
}

func newAppliedSession(t *testing.T, events ...protocol.Event) *Session {
	t.Helper()
	s, err := New(Config{Name: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	t.Cleanup(s.cancel)
	for _, ev := range events {
		s.apply(ev)
	}
	return s
}

func TestSessionSplitsThePromptTokensByCacheState(t *testing.T) {
	s := newAppliedSession(t, resultWithUsage(&protocol.Usage{
		InputTokens:              100,
		CacheReadInputTokens:     800,
		CacheCreationInputTokens: 100,
		OutputTokens:             50,
	}))

	snap := s.Snapshot()
	if snap.InputTokens != 1000 {
		t.Fatalf("InputTokens = %d, want the sum of the three parts", snap.InputTokens)
	}
	if snap.CacheReadTokens != 800 {
		t.Fatalf("CacheReadTokens = %d, want 800", snap.CacheReadTokens)
	}
	if snap.CacheWriteTokens != 100 {
		t.Fatalf("CacheWriteTokens = %d, want 100", snap.CacheWriteTokens)
	}
	if snap.OutputTokens != 50 {
		t.Fatalf("OutputTokens = %d, want 50", snap.OutputTokens)
	}
}

func TestSessionAddsUpTheCacheCountsAcrossTurns(t *testing.T) {
	usage := &protocol.Usage{
		InputTokens:              10,
		CacheReadInputTokens:     80,
		CacheCreationInputTokens: 10,
		OutputTokens:             5,
	}
	s := newAppliedSession(t, resultWithUsage(usage), resultWithUsage(usage))

	snap := s.Snapshot()
	if snap.CacheReadTokens != 160 || snap.CacheWriteTokens != 20 {
		t.Fatalf("read = %d, write = %d, want 160 and 20",
			snap.CacheReadTokens, snap.CacheWriteTokens)
	}
	if snap.InputTokens != 200 {
		t.Fatalf("InputTokens = %d, want 200", snap.InputTokens)
	}
}

func TestCacheHitRateIsUnknownUntilAPromptTokenIsCounted(t *testing.T) {
	s := newAppliedSession(t)
	if _, ok := s.Snapshot().CacheHitRate(); ok {
		t.Fatal("CacheHitRate reported a rate before any turn")
	}

	s.apply(resultWithUsage(&protocol.Usage{
		InputTokens:          6,
		CacheReadInputTokens: 94,
	}))
	rate, ok := s.Snapshot().CacheHitRate()
	if !ok {
		t.Fatal("CacheHitRate reported no rate after a turn")
	}
	if rate != 94 {
		t.Fatalf("CacheHitRate = %d, want 94", rate)
	}
}

func TestCacheHitRateHandlesAZeroDenominator(t *testing.T) {
	if _, ok := CacheHitRate(0, 0); ok {
		t.Fatal("CacheHitRate reported a rate for no prompt token")
	}
	if rate, ok := CacheHitRate(200, 0); !ok || rate != 0 {
		t.Fatalf("CacheHitRate = %d, %v, want 0 and true", rate, ok)
	}
}
