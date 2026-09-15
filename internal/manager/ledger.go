package manager

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CostEntry is one line of the cost ledger: what one prompt cost, and when the
// multiplexer saw it end. A Backfilled entry dates a whole stored session at its
// last activity, because the transcript cannot date a prompt. See docs/cost.md.
type CostEntry struct {
	At               time.Time `json:"at"`
	Session          string    `json:"session"`
	Cost             float64   `json:"cost"`
	InputTokens      int       `json:"input,omitempty"`
	OutputTokens     int       `json:"output,omitempty"`
	CacheReadTokens  int       `json:"cache_read,omitempty"`
	CacheWriteTokens int       `json:"cache_write,omitempty"`
	Backfilled       bool      `json:"backfilled,omitempty"`
}

// LedgerFile is the name of the ledger inside the state directory.
const LedgerFile = "cost.jsonl"

// ledgerKeep is how far back the ledger keeps a line. It covers the longest
// window a setting can name, with room to spare.
const ledgerKeep = 90 * 24 * time.Hour

// ledger appends one line per prompt, and sums the lines inside a window. It
// holds the lines in memory, and reloads them when another process has written
// to the file. See docs/cost.md.
type ledger struct {
	path string

	mu      sync.Mutex
	entries []CostEntry
	size    int64
}

func newLedger(root string) *ledger {
	return &ledger{path: filepath.Join(root, LedgerFile)}
}

func (l *ledger) append(entry CostEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	l.entries = append(l.entries, entry)
	if info, err := file.Stat(); err == nil {
		l.size = info.Size()
	}
	return nil
}

// sumSince adds the cost of every line at or after start.
func (l *ledger) sumSince(start time.Time) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reloadIfChanged()
	var total float64
	for _, entry := range l.entries {
		if !entry.At.Before(start) {
			total += entry.Cost
		}
	}
	return total
}

// reloadIfChanged reads the file again when its size does not match the bytes
// this process last wrote, so a second multiplexer on the same state directory
// does not go unseen.
func (l *ledger) reloadIfChanged() {
	info, err := os.Stat(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			l.entries, l.size = nil, 0
		}
		return
	}
	if info.Size() == l.size {
		return
	}
	l.readLocked()
}

func (l *ledger) load() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.readLocked()
}

func (l *ledger) readLocked() {
	file, err := os.Open(l.path)
	if err != nil {
		l.entries, l.size = nil, 0
		return
	}
	defer file.Close()
	var entries []CostEntry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		var entry CostEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	l.entries = entries
	if info, err := file.Stat(); err == nil {
		l.size = info.Size()
	}
}

// prune drops the lines older than before, and rewrites the file when it drops
// any.
func (l *ledger) prune(before time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reloadIfChanged()
	kept := make([]CostEntry, 0, len(l.entries))
	for _, entry := range l.entries {
		if !entry.At.Before(before) {
			kept = append(kept, entry)
		}
	}
	if len(kept) == len(l.entries) {
		return nil
	}
	return l.writeLocked(kept)
}

// backfill dates the lifetime cost of each stored session at its last activity.
// It runs once, when the ledger has no file, because a transcript cannot date a
// prompt. See docs/cost.md.
func (l *ledger) backfill(metas []Meta) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := os.Stat(l.path); err == nil {
		return nil
	}
	entries := make([]CostEntry, 0, len(metas))
	for _, meta := range metas {
		if meta.Cost == 0 || meta.LastActiveAt.IsZero() {
			continue
		}
		entries = append(entries, CostEntry{
			At:               meta.LastActiveAt.UTC(),
			Session:          meta.Name,
			Cost:             meta.Cost,
			InputTokens:      meta.InputTokens,
			OutputTokens:     meta.OutputTokens,
			CacheReadTokens:  meta.CacheReadTokens,
			CacheWriteTokens: meta.CacheWriteTokens,
			Backfilled:       true,
		})
	}
	return l.writeLocked(entries)
}

func (l *ledger) writeLocked(entries []CostEntry) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	temp := l.path + ".tmp"
	file, err := os.Create(temp)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		if _, err := writer.Write(append(line, '\n')); err != nil {
			file.Close()
			os.Remove(temp)
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		file.Close()
		os.Remove(temp)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(temp)
		return err
	}
	if err := os.Rename(temp, l.path); err != nil {
		os.Remove(temp)
		return err
	}
	l.entries = entries
	if info, err := os.Stat(l.path); err == nil {
		l.size = info.Size()
	}
	return nil
}
