package manager

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/usage"
)

func percent(n int) *int { return &n }

func TestReserveGatePausesHostedSessions(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(m.opts.Root, config.FileName)}
	if _, err := m.SetReserve(config.Window5h, 20); err != nil {
		t.Fatal(err)
	}
	if err := m.StartMCP(); err != nil {
		t.Fatal(err)
	}

	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Hosted: true})
	if err != nil {
		t.Fatal(err)
	}

	// Below the floor, the gate trips and the hosted session pauses.
	m.evaluateReserve(usage.Usage{OK: true, FiveHour: usage.Window{Remaining: percent(10)}})
	if !m.HostingPaused() {
		t.Fatal("the gate did not trip below the floor")
	}
	item, err := m.entry(name)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, func() bool { return item.sess.Paused() })

	// Above the floor, the gate clears and the hosted session resumes.
	m.evaluateReserve(usage.Usage{OK: true, FiveHour: usage.Window{Remaining: percent(50)}})
	if m.HostingPaused() {
		t.Fatal("the gate did not clear above the floor")
	}
	waitFor(t, 2*time.Second, func() bool { return !item.sess.Paused() })
}

func TestReserveGateIgnoresUnknownAndNoReserve(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(m.opts.Root, config.FileName)}

	// No reserve: the gate never trips, whatever the usage.
	if m.reserveTripped(usage.Usage{OK: true, FiveHour: usage.Window{Remaining: percent(1)}}) {
		t.Error("the gate tripped with no reserve configured")
	}

	if _, err := m.SetReserve(config.Window5h, 20); err != nil {
		t.Fatal(err)
	}
	// An unknown percent reads as unknown, never as below the floor.
	if m.reserveTripped(usage.Usage{OK: true, FiveHour: usage.Window{}}) {
		t.Error("the gate tripped on an unknown percent")
	}
	// The reserve guards its own window: a low 5h does not trip a 7d reserve.
	if _, err := m.SetReserve(config.Window7d, 20); err != nil {
		t.Fatal(err)
	}
	if m.reserveTripped(usage.Usage{OK: true, FiveHour: usage.Window{Remaining: percent(1)}, Weekly: usage.Window{Remaining: percent(80)}}) {
		t.Error("a low 5h window tripped a 7d reserve")
	}
}
