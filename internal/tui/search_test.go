package tui

import (
	"strings"
	"testing"
)

func TestRowMatches(t *testing.T) {
	item := row{name: "api-server", title: "Build the API", dir: "/work/backend"}
	cases := []struct {
		needle string
		want   bool
	}{
		{"", true},
		{"api", true},
		{"build", true},
		{"backend", true},
		{"frontend", false},
		{"missing", false},
	}
	for _, tc := range cases {
		if got := rowMatches(item, tc.needle); got != tc.want {
			t.Errorf("rowMatches(%q) = %v, want %v", tc.needle, got, tc.want)
		}
	}
}

// sidebarFocus closes the auto-opened form and puts the focus on the list, so a
// list sequence like "l s" resolves.
func sidebarFocus(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = step(t, m, key("esc"))
	m.focus = focusSidebar
	return m
}

func typeSearch(t *testing.T, m Model, text string) Model {
	t.Helper()
	m, _ = step(t, m, key("l"))
	m, _ = step(t, m, key("s"))
	if !m.searchOn {
		t.Fatal("l s must focus the search box")
	}
	for _, r := range text {
		m, _ = step(t, m, key(string(r)))
	}
	return m
}

func TestSearchNarrowsTheList(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m = spawn(t, m, mgr, "api-server", t.TempDir())
	m = spawn(t, m, mgr, "web-client", t.TempDir())
	m = spawn(t, m, mgr, "docs", t.TempDir())
	m = sidebarFocus(t, m)

	m = typeSearch(t, m, "web")
	if names := rowNames(m); len(names) != 1 || names[0] != "web-client" {
		t.Fatalf("search web narrows to %v, want [web-client]", rowNames(m))
	}
}

func TestSearchFiltersLiveAndStoredRows(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m = spawn(t, m, mgr, "api-live", t.TempDir())
	m = spawn(t, m, mgr, "web-live", t.TempDir())
	m = sidebarFocus(t, m)

	m = typeSearch(t, m, "live")
	if names := rowNames(m); len(names) != 2 {
		t.Fatalf("search live keeps %v, want both live rows", rowNames(m))
	}

	m, _ = step(t, m, key("esc"))
	m, _ = step(t, m, key("esc"))
	m = typeSearch(t, m, "api")
	if names := rowNames(m); len(names) != 1 || names[0] != "api-live" {
		t.Fatalf("search api narrows to %v, want [api-live]", rowNames(m))
	}
}

func TestSearchEscKeepsNeedleThenClears(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m = spawn(t, m, mgr, "api-server", t.TempDir())
	m = spawn(t, m, mgr, "web-client", t.TempDir())
	m = sidebarFocus(t, m)
	m = typeSearch(t, m, "web")

	m, _ = step(t, m, key("esc"))
	if m.searchOn {
		t.Fatal("the first esc must blur the search box")
	}
	if !m.searchActive() {
		t.Fatal("the first esc must keep the needle, so the list stays narrowed")
	}
	if names := rowNames(m); len(names) != 1 || names[0] != "web-client" {
		t.Fatalf("after the first esc the list holds %v, want [web-client]", rowNames(m))
	}

	m, _ = step(t, m, key("esc"))
	if m.searchActive() {
		t.Fatal("the second esc must clear the needle")
	}
	if names := rowNames(m); len(names) != 2 {
		t.Fatalf("after the second esc the list holds %v, want both rows", rowNames(m))
	}
}

func TestSearchBoxTakesOneSidebarLine(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m = spawn(t, m, mgr, "api-server", t.TempDir())
	m = sidebarFocus(t, m)

	before := m.visibleLines()
	m = typeSearch(t, m, "api")
	if got := m.visibleLines(); got != before-1 {
		t.Fatalf("the search box must take one list line: visibleLines %d, want %d", got, before-1)
	}
	if !strings.Contains(visible(m.View()), "api") {
		t.Fatalf("the sidebar must show the search box:\n%s", visible(m.View()))
	}
}
