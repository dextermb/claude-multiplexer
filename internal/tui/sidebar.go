package tui

import (
	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// sidebarInputs is everything the sidebar fold reads: the manager snapshots and
// maps, and the Model-owned bits that shape the list. It carries plain data, so
// deriveSidebar needs no manager. See docs/tui/sessions.md.
type sidebarInputs struct {
	snapshots       []session.Snapshot
	remoteSnapshots []session.Snapshot
	stored          []manager.Meta
	grants          map[string]bool
	parents         map[string]string
	schedules       map[string]string
	workDirs        map[string]string
	projects        map[string][]string
	layouts         map[string]string
	hosted          map[string]bool
	lenders         map[string]string
	owners          map[string]string
	watched         map[string]bool
	held            map[string]bool
	hosts           map[string]string
	readOnlys       map[string]bool
	clientNames     map[string]string
	roots           map[string]string
	folded          map[string]bool
	needle          string
	showArchived    bool
	peering         bool
}

type sidebarView struct {
	rows   []row
	groups []group
	lines  []listLine
}

// gatherSidebarInputs reads the manager once and packs the result as plain data.
// It is the single adapter between the manager and the sidebar fold.
func (m Model) gatherSidebarInputs() sidebarInputs {
	return sidebarInputs{
		snapshots:       m.mgr.Snapshots(),
		remoteSnapshots: m.mgr.RemoteSnapshots(),
		stored:          m.stored,
		grants:          m.mgr.Grants(),
		parents:         m.mgr.Parents(),
		schedules:       m.mgr.Schedules(),
		workDirs:        m.mgr.WorkingDirs(),
		projects:        m.mgr.Projects(),
		layouts:         m.mgr.SessionLayouts(),
		hosted:          m.mgr.Hosted(),
		lenders:         m.mgr.Lenders(),
		owners:          m.mgr.Owners(),
		watched:         m.mgr.Watched(),
		held:            m.mgr.HeldSessions(),
		hosts:           m.mgr.Hosts(),
		readOnlys:       m.mgr.ReadOnly(),
		clientNames:     clientNames(m.mgr.ListAPIClients()),
		roots:           m.roots,
		folded:          m.folded,
		needle:          m.searchNeedle(),
		showArchived:    m.showArchived,
		peering:         m.peering,
	}
}

// deriveSidebar folds the inputs into the rows, the groups, and the lines the
// sidebar draws. It is pure: given the same inputs it gives the same view, and
// it touches no manager. The roots map is a repo-root cache it fills on a miss.
func deriveSidebar(in sidebarInputs) sidebarView {
	rows := make([]row, 0, len(in.stored)+4)
	for _, snap := range in.snapshots {
		item := rowFromSnapshot(snap)
		item.control = in.grants[snap.Name]
		item.parent = in.parents[snap.Name]
		item.scheduled = in.schedules[snap.Name]
		item.workDir = in.workDirs[snap.Name]
		item.projectDirs = in.projects[snap.Name]
		item.layout = in.layouts[snap.Name]
		item.hosted = in.hosted[snap.Name]
		item.lender = in.lenders[snap.Name]
		item.owner = in.owners[snap.Name]
		item.watched = in.watched[snap.Name]
		item.held = in.held[snap.Name]
		rows = append(rows, item)
	}
	for _, snap := range in.remoteSnapshots {
		item := rowFromSnapshot(snap)
		item.host = in.hosts[snap.Name]
		item.readOnly = in.readOnlys[snap.Name]
		rows = append(rows, item)
	}
	for _, meta := range in.stored {
		if meta.Archived && !in.showArchived {
			continue
		}
		rows = append(rows, rowFromMeta(meta))
	}
	children := make(map[string]bool, len(rows))
	for _, item := range rows {
		if item.parent != "" {
			children[item.parent] = true
		}
	}
	for i := range rows {
		if name := in.clientNames[rows[i].owner]; name != "" {
			rows[i].owner = name
		}
		rows[i].group = rowGroup(rows[i], children, in.roots)
		rows[i].section = sectionOf(rows[i])
	}
	if in.needle != "" {
		kept := rows[:0]
		for _, item := range rows {
			if rowMatches(item, in.needle) {
				kept = append(kept, item)
			}
		}
		rows = kept
	}
	grouped, groups := groupRows(rows, in.folded)
	lines := listLines(grouped, groups, sectionedRows(in.peering, grouped))
	return sidebarView{rows: grouped, groups: groups, lines: lines}
}

// rowGroup keys a row on the control session that created it, or that it created
// rows for, then on the peer a remote session involves, and on its repository
// when none of those holds.
func rowGroup(item row, children map[string]bool, roots map[string]string) string {
	if item.parent != "" {
		return byPrefix + item.parent
	}
	if children[item.name] {
		return byPrefix + item.name
	}
	if remote := item.remoteHost(); remote != "" {
		return hostPrefix + remote
	}
	return dirPrefix + groupKey(item.dir, roots)
}

// groupKey caches the walk to the repository, because the fold runs on every
// event and a session directory never changes.
func groupKey(dir string, roots map[string]string) string {
	if dir == "" {
		return ""
	}
	if root, ok := roots[dir]; ok {
		return root
	}
	root := repoRoot(dir)
	roots[dir] = root
	return root
}

// sectionedRows reports whether the sidebar draws section dividers: when this
// host has peers configured, or a hosted or streamed session is present. See
// docs/peers.md.
func sectionedRows(peering bool, rows []row) bool {
	if peering {
		return true
	}
	for _, item := range rows {
		if item.section != sectionLocal {
			return true
		}
	}
	return false
}
