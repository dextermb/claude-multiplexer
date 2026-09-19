package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Meta struct {
	Name             string     `json:"name"`
	Title            string     `json:"title,omitempty"`
	Dir              string     `json:"dir"`
	Model            string     `json:"model"`
	PermissionMode   string     `json:"permission_mode"`
	Effort           string     `json:"effort,omitempty"`
	ClaudeSessionID  string     `json:"claude_session_id"`
	CreatedAt        time.Time  `json:"created_at"`
	LastActiveAt     time.Time  `json:"last_active_at"`
	Turns            int        `json:"turns"`
	Cost             float64    `json:"cost_usd"`
	InputTokens      int        `json:"input_tokens"`
	CacheReadTokens  int        `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int        `json:"cache_write_tokens,omitempty"`
	OutputTokens     int        `json:"output_tokens"`
	WorkingDir       string     `json:"working_dir,omitempty"`
	WorkingDirs      []string   `json:"working_dirs,omitempty"`
	Locks            []string   `json:"locks,omitempty"`
	Layout           string     `json:"layout,omitempty"`
	Control          bool       `json:"control,omitempty"`
	Parent           string     `json:"parent,omitempty"`
	Scheduled        string     `json:"scheduled,omitempty"`
	Owner            string     `json:"owner,omitempty"`
	Hosted           bool       `json:"hosted,omitempty"`
	Lender           string     `json:"lender,omitempty"`
	TempDir          bool       `json:"temp_dir,omitempty"`
	Archived         bool       `json:"archived"`
	ArchivedAt       time.Time  `json:"archived_at,omitempty"`
	WorkItemProvider string     `json:"workitem_provider,omitempty"`
	WorkItemKey      string     `json:"workitem_key,omitempty"`
	WorkItemURL      string     `json:"workitem_url,omitempty"`
	WorkItemStatus   string     `json:"workitem_status,omitempty"`
	WorkItemStatusID string     `json:"workitem_status_id,omitempty"`
	WorkItemSyncedAt time.Time  `json:"workitem_synced_at,omitempty"`
	PRs              []PRMirror `json:"prs,omitempty"`
}

// PRMirror is the pull request of one code base of a session, keyed by its
// directory. A project holds one per code base. See docs/pull-requests.md.
type PRMirror struct {
	Dir        string    `json:"dir"`
	Branch     string    `json:"branch,omitempty"`
	Provider   string    `json:"provider,omitempty"`
	Number     int       `json:"number,omitempty"`
	URL        string    `json:"url,omitempty"`
	State      string    `json:"state,omitempty"`
	Title      string    `json:"title,omitempty"`
	Unresolved int       `json:"unresolved,omitempty"`
	SyncedAt   time.Time `json:"synced_at,omitempty"`
}

func (m Meta) sameAs(other Meta) bool {
	return m.ClaudeSessionID == other.ClaudeSessionID &&
		m.Title == other.Title &&
		m.Model == other.Model &&
		m.PermissionMode == other.PermissionMode &&
		m.Effort == other.Effort &&
		m.WorkingDir == other.WorkingDir &&
		sameStrings(m.WorkingDirs, other.WorkingDirs) &&
		sameStrings(m.Locks, other.Locks) &&
		m.Layout == other.Layout &&
		m.Control == other.Control &&
		m.Turns == other.Turns &&
		m.Cost == other.Cost &&
		m.InputTokens == other.InputTokens &&
		m.CacheReadTokens == other.CacheReadTokens &&
		m.CacheWriteTokens == other.CacheWriteTokens &&
		m.OutputTokens == other.OutputTokens &&
		m.Archived == other.Archived &&
		m.WorkItemProvider == other.WorkItemProvider &&
		m.WorkItemKey == other.WorkItemKey &&
		m.WorkItemStatus == other.WorkItemStatus &&
		samePRs(m.PRs, other.PRs)
}

func samePRs(a, b []PRMirror) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Dir != b[i].Dir ||
			a[i].Provider != b[i].Provider ||
			a[i].Number != b[i].Number ||
			a[i].State != b[i].State ||
			a[i].Unresolved != b[i].Unresolved ||
			a[i].Branch != b[i].Branch {
			return false
		}
	}
	return true
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

// mutateStoredMeta reads the record at path, applies fn, and writes it back. A
// stored session has no entry and no pump, so no lock is needed. See
// docs/manager.md.
func mutateStoredMeta(path string, fn func(*Meta) error) error {
	meta, err := ReadMeta(path)
	if err != nil {
		return err
	}
	if err := fn(&meta); err != nil {
		return err
	}
	return writeMeta(path, meta)
}

func ReadMeta(path string) (Meta, error) {
	var meta Meta
	data, err := os.ReadFile(path)
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	migratePR(data, &meta)
	return meta, nil
}

// migratePR folds the flat pr_* fields of an older meta.json into the PRs list,
// so a record written before the per-directory change still loads. See
// docs/pull-requests.md.
func migratePR(data []byte, meta *Meta) {
	if len(meta.PRs) > 0 {
		return
	}
	var legacy struct {
		Provider   string    `json:"pr_provider"`
		Number     int       `json:"pr_number"`
		URL        string    `json:"pr_url"`
		State      string    `json:"pr_state"`
		Title      string    `json:"pr_title"`
		Unresolved int       `json:"pr_unresolved"`
		Branch     string    `json:"pr_branch"`
		SyncedAt   time.Time `json:"pr_synced_at"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil || legacy.Number == 0 {
		return
	}
	meta.PRs = []PRMirror{{
		Dir:        effectiveDir(*meta),
		Branch:     legacy.Branch,
		Provider:   legacy.Provider,
		Number:     legacy.Number,
		URL:        legacy.URL,
		State:      legacy.State,
		Title:      legacy.Title,
		Unresolved: legacy.Unresolved,
		SyncedAt:   legacy.SyncedAt,
	}}
}

func metaPath(root, name string) string {
	return filepath.Join(root, "sessions", name, "meta.json")
}

func mcpConfigPath(root, name string) string {
	return filepath.Join(root, "sessions", name, "mcp.json")
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
