package manager

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func TestPeerConfigToolsRoundTrip(t *testing.T) {
	m := newTestManager(t)
	path := filepath.Join(t.TempDir(), config.FileName)
	m.opts.ConfigPaths = []string{path}

	if _, err := m.EnablePeering(51900); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "b", URL: "http://host:51900", ClientID: "id", ClientSecret: "sec"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.SetReserve(config.Window5h, 20); err != nil {
		t.Fatal(err)
	}

	view := m.Peers()
	if !view.Enabled || view.Port != 51900 {
		t.Errorf("enabled/port = %v/%d, want true/51900", view.Enabled, view.Port)
	}
	if view.Reserve == nil || view.Reserve.MinPercent != 20 {
		t.Errorf("reserve = %+v, want min_percent 20", view.Reserve)
	}
	if len(view.Hosts) != 1 || view.Hosts[0].Name != "b" {
		t.Fatalf("hosts = %+v, want one named b", view.Hosts)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Peers == nil || cfg.Peers.Hosts[0].ClientSecret != "sec" {
		t.Errorf("the secret was not written to the file")
	}

	if _, had, err := m.RemovePeer("b"); err != nil || !had {
		t.Fatalf("RemovePeer = (%v, %v), want (true, nil)", had, err)
	}
	if _, had, err := m.UnsetReserve(); err != nil || !had {
		t.Fatalf("UnsetReserve = (%v, %v), want (true, nil)", had, err)
	}
	if _, had, err := m.DisablePeering(); err != nil || !had {
		t.Fatalf("DisablePeering = (%v, %v), want (true, nil)", had, err)
	}

	cfg, err = config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Peers != nil {
		t.Errorf("Peers = %+v after clearing everything, want nil", cfg.Peers)
	}
}

func TestSetReserveRejectsABadWindow(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(t.TempDir(), config.FileName)}
	if _, err := m.SetReserve("30d", 20); err == nil {
		t.Fatal("SetReserve took a bad window")
	}
	if _, err := m.SetReserve(config.Window5h, 200); err == nil {
		t.Fatal("SetReserve took a bad percent")
	}
}

func TestListPeersNeverHoldsASecret(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(t.TempDir(), config.FileName)}
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "b", URL: "http://host", ClientID: "id", ClientSecret: "sec"}); err != nil {
		t.Fatal(err)
	}
	// PeerHostView has no secret field, so a secret cannot leak through the view.
	view := m.Peers()
	if len(view.Hosts) != 1 || view.Hosts[0].ClientID != "id" {
		t.Fatalf("view = %+v", view)
	}
}

func TestPeerUsageReportsAnUnreachablePeer(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(t.TempDir(), config.FileName)}
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "down", URL: "http://127.0.0.1:1", ClientID: "id", ClientSecret: "sec"}); err != nil {
		t.Fatal(err)
	}
	reports := m.PeerUsage(context.Background())
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	if reports[0].Reachable {
		t.Errorf("an unreachable peer reported reachable")
	}
	if reports[0].Error == "" {
		t.Errorf("an unreachable peer had no error")
	}
}
