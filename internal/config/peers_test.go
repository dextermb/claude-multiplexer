package config

import (
	"path/filepath"
	"testing"
)

func TestPeersRoundTripThroughTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	cfg := Config{Peers: &Peers{
		Listen:  "0.0.0.0:51900",
		Reserve: &Reserve{Window: Window5h, MinPercent: 20},
		Hosts: []PeerHost{{
			Name:         "workstation",
			URL:          "http://192.168.1.20:51900",
			ClientID:     "cid_1",
			ClientSecret: "secret_1",
		}},
	}}
	if err := Write(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Peers == nil {
		t.Fatal("Peers is nil after the round trip")
	}
	if got.Peers.Listen != "0.0.0.0:51900" {
		t.Errorf("Listen = %q, want 0.0.0.0:51900", got.Peers.Listen)
	}
	if got.Peers.Reserve == nil || got.Peers.Reserve.MinPercent != 20 {
		t.Errorf("Reserve = %+v, want min_percent 20", got.Peers.Reserve)
	}
	if len(got.Peers.Hosts) != 1 || got.Peers.Hosts[0].Name != "workstation" {
		t.Errorf("Hosts = %+v, want one host named workstation", got.Peers.Hosts)
	}
}

func TestValidWindowKnowsOnlyTheTwoWindows(t *testing.T) {
	for _, ok := range []string{Window5h, Window7d} {
		if !ValidWindow(ok) {
			t.Errorf("ValidWindow(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "1h", "30d", "weekly"} {
		if ValidWindow(bad) {
			t.Errorf("ValidWindow(%q) = true, want false", bad)
		}
	}
}
