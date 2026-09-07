package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestSessionRowShowsScheduleBadge(t *testing.T) {
	m := Model{}
	got := m.sessionRow(row{name: "poll", live: true, state: session.StateBusy, scheduled: "poll"})
	if !strings.Contains(got, scheduleMark) {
		t.Fatalf("row must show the schedule badge:\n%q", got)
	}
}

func TestSessionRowWithoutScheduleHasNoBadge(t *testing.T) {
	m := Model{}
	got := m.sessionRow(row{name: "api", live: true, state: session.StateBusy})
	if strings.Contains(got, scheduleMark) {
		t.Fatalf("row must not show the schedule badge:\n%q", got)
	}
}
