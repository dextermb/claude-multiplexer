package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestSessionRowShowsHoistBadge(t *testing.T) {
	m := Model{}
	got := m.sessionRow(row{name: "app", live: true, state: session.StateBusy, lender: "studio"})
	if !strings.Contains(got, hoistMark) {
		t.Fatalf("a hoisted row must show the hoist badge:\n%q", got)
	}
}

func TestSessionRowWithoutLenderHasNoHoistBadge(t *testing.T) {
	m := Model{}
	got := m.sessionRow(row{name: "app", live: true, state: session.StateBusy})
	if strings.Contains(got, hoistMark) {
		t.Fatalf("a local row must not show the hoist badge:\n%q", got)
	}
}

func TestSessionRowCombinesFlags(t *testing.T) {
	m := Model{}
	got := m.sessionRow(row{name: "app", live: true, state: session.StateBusy, lender: "studio", control: true})
	if !strings.Contains(got, hoistMark+controlMark) {
		t.Fatalf("a hoisted control row must combine the flags as %q:\n%q", hoistMark+controlMark, got)
	}
}
