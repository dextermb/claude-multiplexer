package tui

import "testing"

// sectionize runs the sidebar pipeline the way refresh does, minus the manager,
// so a test can read the section dividers and the row order.
func sectionize(rows []row, sectioned bool) ([]row, []listLine) {
	for i := range rows {
		rows[i].section = sectionOf(rows[i])
	}
	sorted, groups := groupRows(rows, nil)
	return sorted, listLines(sorted, groups, sectioned)
}

func dividers(lines []listLine) []string {
	var out []string
	for _, line := range lines {
		if line.isDivider() {
			out = append(out, line.divider)
		}
	}
	return out
}

func TestSectionsSortLocalHostedStreamed(t *testing.T) {
	rows := []row{
		{name: "streamed-1", dir: "/peer/work", live: true, host: "workstation"},
		{name: "local-1", dir: "/home/me/app", live: true},
		{name: "hosted-1", dir: "/home/me/srv", live: true, hosted: true},
	}
	sorted, lines := sectionize(rows, true)

	// The rows come out local, then hosted, then streamed.
	order := []sectionKind{sectionLocal, sectionHosted, sectionStreamed}
	for i, item := range sorted {
		if item.section != order[i] {
			t.Fatalf("row %d (%s) in section %d, want %d", i, item.name, item.section, order[i])
		}
	}

	// The dividers name the local band and the remote band, with no sub-band
	// label: a hosted session shows muted instead. See docs/tui/sessions.md.
	want := []string{"local sessions", "remote sessions"}
	got := dividers(lines)
	if len(got) != len(want) {
		t.Fatalf("dividers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("divider %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSectionsHideWhenNotSectioned(t *testing.T) {
	rows := []row{{name: "local-1", dir: "/home/me/app", live: true}}
	_, lines := sectionize(rows, false)
	if got := dividers(lines); len(got) != 0 {
		t.Errorf("dividers = %v with sectioning off, want none", got)
	}
}

func TestRemoteParentShowsOnceWithoutHosted(t *testing.T) {
	rows := []row{
		{name: "local-1", dir: "/home/me/app", live: true},
		{name: "streamed-1", dir: "/peer/work", live: true, host: "workstation"},
	}
	_, lines := sectionize(rows, true)
	want := []string{"local sessions", "remote sessions"}
	got := dividers(lines)
	if len(got) != len(want) {
		t.Fatalf("dividers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("divider %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// groupize runs the grouping the way refresh does for remote rows: a streamed
// or hosted session keys on its peer, so a test reads the groups.
func groupize(rows []row) ([]row, []group) {
	for i := range rows {
		rows[i].section = sectionOf(rows[i])
		if remote := rows[i].remoteHost(); remote != "" {
			rows[i].group = hostPrefix + remote
		} else {
			rows[i].group = dirPrefix + rows[i].dir
		}
	}
	return groupRows(rows, nil)
}

func TestStreamedSessionsGroupByHost(t *testing.T) {
	rows := []row{
		{name: "s1", dir: "/tmp/cmux-session-1", live: true, host: "touchbar"},
		{name: "s2", dir: "/tmp/cmux-session-2", live: true, host: "touchbar"},
	}
	_, groups := groupize(rows)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1: both sessions belong to one host", len(groups))
	}
	if groups[0].label != "touchbar" {
		t.Errorf("group label = %q, want the host name touchbar", groups[0].label)
	}
	if groups[0].count != 2 {
		t.Errorf("group count = %d, want 2", groups[0].count)
	}
}

func TestHostedSessionsGroupByClient(t *testing.T) {
	rows := []row{
		{name: "h1", dir: "/srv/a", live: true, hosted: true, owner: "laptop"},
		{name: "h2", dir: "/srv/b", live: true, hosted: true, owner: "laptop"},
	}
	_, groups := groupize(rows)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1: both sessions belong to one client", len(groups))
	}
	if groups[0].label != "laptop" {
		t.Errorf("group label = %q, want the client name laptop", groups[0].label)
	}
}

// TestSectionsSplitTheSameDirectory checks a hosted and a local session in the
// same directory land in different sections, not one shared group.
func TestSectionsSplitTheSameDirectory(t *testing.T) {
	rows := []row{
		{name: "local-1", dir: "/home/me/app", live: true},
		{name: "hosted-1", dir: "/home/me/app", live: true, hosted: true},
	}
	sorted, _ := sectionize(rows, true)
	byName := map[string]sectionKind{}
	for _, item := range sorted {
		byName[item.name] = item.section
	}
	if byName["local-1"] != sectionLocal {
		t.Errorf("local-1 section = %d, want local", byName["local-1"])
	}
	if byName["hosted-1"] != sectionHosted {
		t.Errorf("hosted-1 section = %d, want hosted", byName["hosted-1"])
	}
}
