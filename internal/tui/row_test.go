package tui

import "testing"

func TestDisplayNameFallsBackToTheName(t *testing.T) {
	if got := (row{name: "api"}).displayName(); got != "api" {
		t.Fatalf("displayName = %q, want the name", got)
	}
	if got := (row{name: "api", title: "Billing rewrite"}).displayName(); got != "Billing rewrite" {
		t.Fatalf("displayName = %q, want the title", got)
	}
}
