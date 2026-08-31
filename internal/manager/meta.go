package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Meta struct {
	Name            string    `json:"name"`
	Dir             string    `json:"dir"`
	Model           string    `json:"model"`
	PermissionMode  string    `json:"permission_mode"`
	ClaudeSessionID string    `json:"claude_session_id"`
	CreatedAt       time.Time `json:"created_at"`
	LastActiveAt    time.Time `json:"last_active_at"`
	Turns           int       `json:"turns"`
	Cost            float64   `json:"cost_usd"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	Archived        bool      `json:"archived"`
	ArchivedAt      time.Time `json:"archived_at,omitempty"`
}

func (m Meta) sameAs(other Meta) bool {
	return m.ClaudeSessionID == other.ClaudeSessionID &&
		m.Model == other.Model &&
		m.PermissionMode == other.PermissionMode &&
		m.Turns == other.Turns &&
		m.Cost == other.Cost &&
		m.InputTokens == other.InputTokens &&
		m.OutputTokens == other.OutputTokens &&
		m.Archived == other.Archived
}

func writeMeta(path string, meta Meta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ReadMeta(path string) (Meta, error) {
	var meta Meta
	data, err := os.ReadFile(path)
	if err != nil {
		return meta, err
	}
	err = json.Unmarshal(data, &meta)
	return meta, err
}

func metaPath(root, name string) string {
	return filepath.Join(root, "sessions", name, "meta.json")
}

func transcriptPath(root, name string) string {
	return filepath.Join(root, "sessions", name, "transcript.jsonl")
}

func sessionDir(root, name string) string {
	return filepath.Join(root, "sessions", name)
}

func scanMeta(root string) ([]Meta, error) {
	entries, err := os.ReadDir(filepath.Join(root, "sessions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Meta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := ReadMeta(metaPath(root, entry.Name()))
		if err != nil {
			continue
		}
		if meta.Name == "" {
			meta.Name = entry.Name()
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastActiveAt.After(out[j].LastActiveAt)
	})
	return out, nil
}
