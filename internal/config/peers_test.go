package config

import (
	"path/filepath"
	"testing"
)

func TestPeersRoundTripThroughTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	cfg := Config{Peers: &Peers{
		Enabled: true,
		Port:    51900,
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
	if !got.Peers.Enabled || got.Peers.Port != 51900 {
		t.Errorf("Enabled/Port = %v/%d, want true/51900", got.Peers.Enabled, got.Peers.Port)
	}
	if got.Peers.ListenAddr() != "0.0.0.0:51900" {
		t.Errorf("ListenAddr = %q, want 0.0.0.0:51900", got.Peers.ListenAddr())
	}
	if got.Peers.Reserve == nil || got.Peers.Reserve.MinPercent != 20 {
		t.Errorf("Reserve = %+v, want min_percent 20", got.Peers.Reserve)
	}
	if len(got.Peers.Hosts) != 1 || got.Peers.Hosts[0].Name != "workstation" {
		t.Errorf("Hosts = %+v, want one host named workstation", got.Peers.Hosts)
	}
}

func TestListenAddrDefaultsAndDisables(t *testing.T) {
	cases := []struct {
		name  string
		peers *Peers
		want  string
	}{
		{"nil", nil, ""},
		{"disabled", &Peers{Enabled: false, Port: 51900}, ""},
		{"enabled default port", &Peers{Enabled: true}, "0.0.0.0:51900"},
		{"enabled custom port", &Peers{Enabled: true, Port: 52000}, "0.0.0.0:52000"},
	}
	for _, tc := range cases {
		if got := tc.peers.ListenAddr(); got != tc.want {
			t.Errorf("%s: ListenAddr = %q, want %q", tc.name, got, tc.want)
		}
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
