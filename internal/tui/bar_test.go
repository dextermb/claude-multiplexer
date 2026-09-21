package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/manager"
)

func segTexts(segs []barSeg) []string {
	out := make([]string, 0, len(segs))
	for _, seg := range segs {
		out = append(out, seg.text)
	}
	return out
}

func indexOfPrefix(texts []string, prefix string) int {
	for i, text := range texts {
		if strings.HasPrefix(text, prefix) {
			return i
		}
	}
	return -1
}

func busyRow() row {
	return row{
		label:     "idle",
		live:      true,
		input:     1000,
		cacheRead: 940,
		output:    50,
		cost:      0.25,
	}
}

func TestSessionBarShowsTheCacheHitRate(t *testing.T) {
	var m Model
	texts := segTexts(m.rightSegs(busyRow()))
	if indexOfPrefix(texts, "cache 94%") < 0 {
		t.Fatalf("segments = %v, want one reading cache 94%%", texts)
	}
}

func TestSessionBarShedsTheCacheRateAfterTheCostAndBeforeTheTokens(t *testing.T) {
	var m Model
	texts := segTexts(m.rightSegs(busyRow()))

	tokens := indexOfPrefix(texts, "1.0k in")
	cache := indexOfPrefix(texts, "cache ")
	cost := indexOfPrefix(texts, "$")
	if tokens < 0 || cache < 0 || cost < 0 {
		t.Fatalf("segments = %v, want the tokens, the cache rate, and the cost", texts)
	}
	if !(tokens < cache && cache < cost) {
		t.Fatalf("segments = %v, want the order tokens, cache, cost", texts)
	}
}

func TestSessionBarHidesTheCacheRateWithoutAPromptToken(t *testing.T) {
	var m Model
	texts := segTexts(m.rightSegs(row{label: "idle", live: true}))
	if indexOfPrefix(texts, "cache ") >= 0 {
		t.Fatalf("segments = %v, want no cache rate", texts)
	}
}

func TestSessionBarShowsThePullRequest(t *testing.T) {
	var m Model
	item := busyRow()
	item.prs = []manager.PRBadge{{Provider: "gitlab", Number: 1045, State: "open", Unresolved: 3}}
	texts := segTexts(m.rightSegs(item))
	if indexOfPrefix(texts, "!1045 (3)") < 0 {
		t.Fatalf("segments = %v, want !1045 (3)", texts)
	}

	item.prs = []manager.PRBadge{{Provider: "github", Number: 7, State: "draft"}}
	texts = segTexts(m.rightSegs(item))
	if indexOfPrefix(texts, "#7 draft") < 0 {
		t.Fatalf("segments = %v, want #7 draft", texts)
	}
}

func TestSessionBarShowsAPullRequestPerCodeBase(t *testing.T) {
	var m Model
	item := busyRow()
	item.prs = []manager.PRBadge{
		{Provider: "github", Number: 1045, State: "open", Unresolved: 3},
		{Provider: "gitlab", Number: 88, State: "open"},
	}
	texts := segTexts(m.rightSegs(item))
	if indexOfPrefix(texts, "#1045 (3)") < 0 || indexOfPrefix(texts, "!88") < 0 {
		t.Fatalf("segments = %v, want a segment per code base", texts)
	}
}

func TestSessionBarHidesThePullRequestWithoutOne(t *testing.T) {
	var m Model
	texts := segTexts(m.rightSegs(busyRow()))
	if indexOfPrefix(texts, "#") >= 0 || indexOfPrefix(texts, "!") >= 0 {
		t.Fatalf("segments = %v, want no PR segment", texts)
	}
}

func TestSessionBarReordersAndRemovesByConfig(t *testing.T) {
	m := Model{barSpecs: map[string]config.BarSpec{
		config.BarSession: {Right: []config.BarElement{{ID: "cost"}, {ID: "state"}}},
	}}
	texts := segTexts(m.rightSegs(busyRow()))
	if len(texts) != 2 {
		t.Fatalf("segments = %v, want only the two named elements", texts)
	}
	if indexOfPrefix(texts, "$") != 0 || texts[1] != "idle" {
		t.Fatalf("segments = %v, want the cost first and the state second", texts)
	}
}

func TestSessionBarDrawsACustomElementFromTheCache(t *testing.T) {
	item := busyRow()
	item.name = "api"
	m := Model{
		barSpecs: map[string]config.BarSpec{
			config.BarSession: {Right: []config.BarElement{{Script: "branch.sh", Label: "branch"}}},
		},
		barOutputs: map[string]string{
			barOutputKey(config.BarSession, "api", "branch"): "main",
		},
	}
	texts := segTexts(m.rightSegs(item))
	if len(texts) != 1 || texts[0] != "main" {
		t.Fatalf("segments = %v, want the custom output main", texts)
	}
}

func TestSessionBarHidesACustomElementWithNoOutput(t *testing.T) {
	m := Model{barSpecs: map[string]config.BarSpec{
		config.BarSession: {Right: []config.BarElement{{Script: "branch.sh", Label: "branch"}}},
	}}
	if texts := segTexts(m.rightSegs(busyRow())); len(texts) != 0 {
		t.Fatalf("segments = %v, want none without a cached output", texts)
	}
}

func TestSessionBarShedsOneSegmentAtATime(t *testing.T) {
	var m Model
	item := busyRow()
	want := len(m.rightSegs(item)) + 1
	if got := len(m.barRights(item)); got != want {
		t.Fatalf("barRights gave %d widths, want %d", got, want)
	}
}
