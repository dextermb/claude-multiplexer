package manager

import (
	"os"
	"strconv"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

const configWatchTick = time.Second

// StartConfigWatch starts the clock that reloads the settings when the settings
// file changes on disk, so an edit by hand takes effect without a restart. Call
// it once. See docs/config.md.
func (m *Manager) StartConfigWatch() {
	if m.configStop != nil {
		return
	}
	m.configStop = make(chan struct{})
	m.configPrint = configFingerprint(m.opts.ConfigPaths)
	m.configWG.Add(1)
	go m.configLoop()
}

func (m *Manager) configLoop() {
	defer m.configWG.Done()
	ticker := time.NewTicker(configWatchTick)
	defer ticker.Stop()
	for {
		select {
		case <-m.configStop:
			return
		case <-ticker.C:
			m.checkConfig()
		}
	}
}

// checkConfig reloads the settings when the fingerprint of the active settings
// file changes. It publishes a bare reload with no notice, so the interface
// reads the file again and keeps its own status line. See docs/config.md.
func (m *Manager) checkConfig() {
	print := configFingerprint(m.opts.ConfigPaths)
	if print == m.configPrint {
		return
	}
	m.configPrint = print
	m.bus.Publish(Event{Reload: true})
}

// configFingerprint gives a string that changes when the active settings file
// changes: its path, its modification time, and its size. An empty string means
// no settings file is there yet.
func configFingerprint(paths []string) string {
	path := config.Active(paths...)
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return path
	}
	return path + "|" + strconv.FormatInt(info.ModTime().UnixNano(), 10) + "|" + strconv.FormatInt(info.Size(), 10)
}
