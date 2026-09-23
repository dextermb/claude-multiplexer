package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/keys"
	"github.com/dextermb/claude-multiplexer/internal/update"
)

// updateCheckInterval is how often the interface asks GitHub for the latest
// release. See docs/version-updates.md.
const updateCheckInterval = time.Hour

const updateFetchTimeout = 15 * time.Second

const updateDismissWindow = 24 * time.Hour

type updateTickMsg struct{}

type updateMsg struct {
	tag      string
	url      string
	outdated bool
	dismiss  time.Time
}

func updateTick() tea.Cmd {
	return tea.Tick(updateCheckInterval, func(time.Time) tea.Msg { return updateTickMsg{} })
}

// updateCheck reads the cached state, refreshes it from GitHub at most once an
// hour, and reports whether the running binary is out of date. It runs off the
// main loop. See docs/version-updates.md.
func (m Model) updateCheck() tea.Cmd {
	root := m.mgr.Root()
	version := m.version
	buildTime := m.buildTime
	enabled := m.checkUpdates
	return func() tea.Msg {
		path := update.StatePath(root)
		st, err := update.LoadState(path)
		if err != nil {
			return updateMsg{}
		}
		if !enabled {
			return updateMsg{dismiss: st.DismissedUntil}
		}
		if st.LastCheck.IsZero() || time.Since(st.LastCheck) >= updateCheckInterval {
			ctx, cancel := context.WithTimeout(context.Background(), updateFetchTimeout)
			defer cancel()
			rel, ok, err := update.Latest(ctx, nil)
			if err == nil {
				st.LastCheck = time.Now()
				if ok {
					st.Tag = rel.TagName
					st.URL = rel.HTMLURL
					st.PublishedAt = rel.PublishedAt
				}
				_ = update.SaveState(path, st)
			}
		}
		rel := update.Release{TagName: st.Tag, HTMLURL: st.URL, PublishedAt: st.PublishedAt}
		return updateMsg{
			tag:      st.Tag,
			url:      st.URL,
			outdated: update.Outdated(version, buildTime, rel),
			dismiss:  st.DismissedUntil,
		}
	}
}

func (m Model) handleUpdate(msg updateMsg) (tea.Model, tea.Cmd) {
	m.updateTag = msg.tag
	m.updateURL = msg.url
	m.updateOutdated = msg.outdated
	m.updateDismiss = msg.dismiss
	if m.updateTicking {
		return m, nil
	}
	m.updateTicking = true
	return m, updateTick()
}

func (m Model) handleUpdateTick() (tea.Model, tea.Cmd) {
	m.updateTicking = false
	return m, m.updateCheck()
}

// updateVisible reports whether the update banner shows now: the binary is out
// of date, the check is on, and the dismissal has run out.
func (m Model) updateVisible() bool {
	if !m.updateOutdated || !m.checkUpdates {
		return false
	}
	return time.Now().After(m.updateDismiss)
}

// dismissUpdate hides the banner for a day and stores that in the state file.
func (m Model) dismissUpdate() (tea.Model, tea.Cmd) {
	if !m.updateVisible() {
		return m, nil
	}
	until := time.Now().Add(updateDismissWindow)
	m.updateDismiss = until
	m.status = "update notice dismissed for a day"
	mgr := m.mgr
	return m, func() tea.Msg {
		path := update.StatePath(mgr.Root())
		st, err := update.LoadState(path)
		if err != nil {
			return nil
		}
		st.DismissedUntil = until
		_ = update.SaveState(path, st)
		return nil
	}
}

func (m Model) bannerHeight() int {
	if m.updateVisible() {
		return 1
	}
	return 0
}

func (m Model) updateBannerView() string {
	text := fmt.Sprintf("⚠ update available: %s is newer than %s. Press %s to dismiss for a day.",
		m.updateTag, m.versionLabel(), m.dismissUpdateKey())
	return updateBannerStyle.Width(m.width).Render(truncate(text, m.width-2))
}

// versionLabel names the running version for the banner: the tag of a released
// build, or a plain phrase for a build from source.
func (m Model) versionLabel() string {
	if update.IsSemver(m.version) {
		return m.version
	}
	return "this build"
}

func (m Model) dismissUpdateKey() string {
	if ks := m.keys.Keys(keys.GlobalDismissUpdate); len(ks) > 0 {
		return ks[0]
	}
	return "ctrl+g"
}
