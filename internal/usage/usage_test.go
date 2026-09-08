package usage

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestParseReadsBothWindows(t *testing.T) {
	h := http.Header{}
	h.Set("anthropic-ratelimit-unified-5h-status", "allowed")
	h.Set("anthropic-ratelimit-unified-5h-remaining", "42")
	h.Set("anthropic-ratelimit-unified-5h-reset", "2026-09-08T12:00:00Z")
	h.Set("anthropic-ratelimit-unified-7d-remaining", "80%")

	got := Parse(h)
	if got.FiveHour.Status != "allowed" {
		t.Errorf("5h status = %q, want allowed", got.FiveHour.Status)
	}
	if got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 42 {
		t.Errorf("5h remaining = %v, want 42", got.FiveHour.Remaining)
	}
	if got.FiveHour.ResetAt == nil || got.FiveHour.ResetAt.Hour() != 12 {
		t.Errorf("5h reset = %v, want 12:00Z", got.FiveHour.ResetAt)
	}
	if got.Weekly.Remaining == nil || *got.Weekly.Remaining != 80 {
		t.Errorf("7d remaining = %v, want 80", got.Weekly.Remaining)
	}
}

func TestParseLeavesAMissingHeaderUnknown(t *testing.T) {
	got := Parse(http.Header{})
	if got.FiveHour.Known() {
		t.Errorf("5h known with no headers, want unknown")
	}
	if got.Weekly.Remaining != nil {
		t.Errorf("7d remaining = %v, want nil", got.Weekly.Remaining)
	}
	if !got.OK {
		t.Errorf("OK = false on a clean parse, want true")
	}
}

func TestParseReadsAUnixReset(t *testing.T) {
	h := http.Header{}
	h.Set("anthropic-ratelimit-unified-5h-reset", "1757332800")
	got := Parse(h)
	if got.FiveHour.ResetAt == nil || got.FiveHour.ResetAt.Unix() != 1757332800 {
		t.Errorf("reset = %v, want unix 1757332800", got.FiveHour.ResetAt)
	}
}

func TestRefreshKeepsLastKnownOnAFailedFetch(t *testing.T) {
	good := http.Header{}
	good.Set("anthropic-ratelimit-unified-5h-remaining", "50")
	fail := false
	p := NewPoller(func(context.Context) (http.Header, error) {
		if fail {
			return nil, errors.New("down")
		}
		return good, nil
	}, time.Minute)

	p.Refresh(context.Background())
	if got := p.Usage(); got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 50 {
		t.Fatalf("after good fetch remaining = %v, want 50", got.FiveHour.Remaining)
	}
	fail = true
	got := p.Refresh(context.Background())
	if got.OK {
		t.Errorf("OK = true after a failed fetch, want false")
	}
	if got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 50 {
		t.Errorf("remaining = %v after a failed fetch, want the last-known 50", got.FiveHour.Remaining)
	}
	if got.Error == "" {
		t.Errorf("Error is empty after a failed fetch, want the message")
	}
}
