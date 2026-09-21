package tui

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// barTickInterval is how often the interface looks for a custom bar element that
// is due to run. The interval is the floor of a refresh, so no element runs
// faster than this. See docs/config/bars.md.
const barTickInterval = config.MinBarRefresh

// barScriptTimeout stops a script that runs too long, so a slow script cannot
// hold the bar. See docs/config/bars.md.
const barScriptTimeout = 2 * time.Second

type barTickMsg struct{ t time.Time }

// barOutputMsg carries one script result back to the main loop.
type barOutputMsg struct {
	bar     string
	session string
	label   string
	text    string
	err     error
}

// barJob is one custom element that is due to run: the script, and the payload
// it reads on stdin.
type barJob struct {
	bar     string
	session string
	label   string
	script  string
	payload []byte
}

func barTick() tea.Cmd {
	return tea.Tick(barTickInterval, func(t time.Time) tea.Msg { return barTickMsg{t} })
}

// hasBarScripts reports whether any resolved bar holds a custom element, so the
// interface runs the ticker only when there is a script to run.
func (m Model) hasBarScripts() bool {
	for _, bar := range []string{config.BarSession, config.BarStatus} {
		spec := m.barSpec(bar)
		if anyCustom(spec.Left) || anyCustom(spec.Right) {
			return true
		}
	}
	return false
}

func anyCustom(elements []config.BarElement) bool {
	for _, e := range elements {
		if e.Custom() {
			return true
		}
	}
	return false
}

// handleBarTick runs the custom elements that are due, then schedules the next
// tick. It stops the loop when the settings hold no script.
func (m Model) handleBarTick(msg barTickMsg) (tea.Model, tea.Cmd) {
	if !m.hasBarScripts() {
		m.barTicking = false
		return m, nil
	}
	jobs := m.dueBarJobs(msg.t)
	cmds := make([]tea.Cmd, 0, len(jobs)+1)
	for _, job := range jobs {
		m.barRuns[barOutputKey(job.bar, job.session, job.label)] = msg.t
		cmds = append(cmds, runBarScript(job, m.configDir()))
	}
	cmds = append(cmds, barTick())
	return m, tea.Batch(cmds...)
}

// handleBarOutput writes a script result to the cache. A failure, or an empty
// line, keeps the last good output, so the bar does not flicker on a transient
// error. A first failure with no earlier output writes a one-line notice. See
// docs/config/bars.md.
func (m Model) handleBarOutput(msg barOutputMsg) (tea.Model, tea.Cmd) {
	key := barOutputKey(msg.bar, msg.session, msg.label)
	if msg.err != nil {
		if _, ok := m.barOutputs[key]; !ok {
			m.status = "bar element " + msg.label + ": " + firstLine([]byte(msg.err.Error()))
		}
		return m, nil
	}
	if msg.text == "" {
		return m, nil
	}
	m.barOutputs[key] = msg.text
	return m, nil
}

// dueBarJobs lists the custom elements that show now and whose refresh has
// elapsed: the elements of the selected session's bar, and of the status bar.
func (m Model) dueBarJobs(now time.Time) []barJob {
	var jobs []barJob
	if item, ok := m.selectedRow(); ok {
		payload := m.sessionPayload(item)
		spec := m.barSpec(config.BarSession)
		for _, side := range [][]config.BarElement{spec.Left, spec.Right} {
			jobs = m.appendDueJobs(jobs, config.BarSession, item.name, payload, side, now)
		}
	}
	payload := m.statusPayload()
	spec := m.barSpec(config.BarStatus)
	for _, side := range [][]config.BarElement{spec.Left, spec.Right} {
		jobs = m.appendDueJobs(jobs, config.BarStatus, "", payload, side, now)
	}
	return jobs
}

func (m Model) appendDueJobs(jobs []barJob, bar, session string, payload []byte, elements []config.BarElement, now time.Time) []barJob {
	for _, e := range elements {
		if !e.Custom() {
			continue
		}
		if !m.barDue(bar, session, e.Label, e.BarRefresh(), now) {
			continue
		}
		jobs = append(jobs, barJob{
			bar:     bar,
			session: session,
			label:   e.Label,
			script:  e.Script,
			payload: payload,
		})
	}
	return jobs
}

func (m Model) barDue(bar, session, label string, refresh time.Duration, now time.Time) bool {
	last, ok := m.barRuns[barOutputKey(bar, session, label)]
	return !ok || now.Sub(last) >= refresh
}

// configDir is the directory of the active settings file, which a relative
// script path resolves against.
func (m Model) configDir() string {
	if active := config.Active(m.opts.ConfigPaths...); active != "" {
		return filepath.Dir(active)
	}
	return ""
}

func runBarScript(job barJob, baseDir string) tea.Cmd {
	return func() tea.Msg {
		out := barOutputMsg{bar: job.bar, session: job.session, label: job.label}
		runner, ok := config.BarScriptRunner(job.script)
		if !ok {
			out.err = &barScriptError{"unknown script type"}
			return out
		}
		ctx, cancel := context.WithTimeout(context.Background(), barScriptTimeout)
		defer cancel()
		args := append(runner[1:], expandBarPath(job.script, baseDir))
		cmd := exec.CommandContext(ctx, runner[0], args...)
		cmd.Stdin = bytes.NewReader(job.payload)
		if baseDir != "" {
			cmd.Dir = baseDir
		}
		data, err := cmd.Output()
		if err != nil {
			out.err = err
			return out
		}
		out.text = firstLine(data)
		return out
	}
}

type barScriptError struct{ msg string }

func (e *barScriptError) Error() string { return e.msg }

// expandBarPath resolves a script path: a leading ~ against the home directory,
// and a relative path against the settings directory.
func expandBarPath(path, baseDir string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	if filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}

func firstLine(data []byte) string {
	text := string(data)
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	return strings.TrimSpace(text)
}
