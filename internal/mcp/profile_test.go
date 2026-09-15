package mcp_test

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func TestParseProfileTakesTheDefaultForAnEmptyName(t *testing.T) {
	got, err := mcp.ParseProfile("")
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}
	if got != mcp.DefaultProfile {
		t.Fatalf("ParseProfile(\"\") = %q, want the default", got)
	}
}

func TestParseProfileRejectsAnUnknownName(t *testing.T) {
	if _, err := mcp.ParseProfile("everything"); err == nil {
		t.Fatal("ParseProfile took an unknown name")
	}
}

func TestTheMinimalProfileCarriesFewerToolsThanTheStandard(t *testing.T) {
	minimal := mcp.OpenToolsFor(mcp.ProfileMinimal)
	standard := mcp.OpenToolsFor(mcp.ProfileStandard)
	if len(minimal) >= len(standard) {
		t.Fatalf("minimal = %d tools, standard = %d, want fewer", len(minimal), len(standard))
	}
	open := make(map[string]bool, len(standard))
	for _, name := range standard {
		open[name] = true
	}
	for _, name := range minimal {
		if !open[name] {
			t.Fatalf("the minimal profile holds %q, which is not an open tool", name)
		}
	}
}

func TestTheProfileNeverChangesTheControlGrant(t *testing.T) {
	for _, profile := range []mcp.Profile{mcp.ProfileMinimal, mcp.ProfileStandard} {
		granted := make(map[string]bool)
		for _, name := range mcp.AllowedTools(profile, true) {
			granted[name] = true
		}
		for _, name := range mcp.ControlTools {
			if !granted[mcp.Qualify(name)] {
				t.Fatalf("profile %q dropped the control tool %q", profile, name)
			}
		}
		for _, name := range mcp.AllowedTools(profile, false) {
			for _, control := range mcp.ControlTools {
				if name == mcp.Qualify(control) {
					t.Fatalf("profile %q gave the control tool %q without the grant", profile, control)
				}
			}
		}
	}
}

func TestTheMinimalProfileStillCarriesTheSessionReads(t *testing.T) {
	carried := make(map[string]bool)
	for _, name := range mcp.AllowedTools(mcp.ProfileMinimal, false) {
		carried[name] = true
	}
	for _, name := range []string{mcp.ToolList, mcp.ToolMessages, mcp.ToolRename, mcp.ToolListJobs} {
		if !carried[mcp.Qualify(name)] {
			t.Fatalf("the minimal profile dropped %q", name)
		}
	}
}
