package tui

import "encoding/json"

// The JSON a custom bar element reads on stdin. A session element reads the
// selected session, and a status element reads the totals. See
// docs/config/bars.md.
type barPayload struct {
	Bar     string          `json:"bar"`
	Session *barSessionData `json:"session,omitempty"`
	Totals  *barTotalsData  `json:"totals,omitempty"`
}

type barSessionData struct {
	Name     string           `json:"name"`
	Title    string           `json:"title"`
	Dir      string           `json:"dir"`
	WorkDir  string           `json:"workDir"`
	Model    string           `json:"model"`
	Mode     string           `json:"mode"`
	Effort   string           `json:"effort"`
	State    string           `json:"state"`
	Live     bool             `json:"live"`
	Control  bool             `json:"control"`
	Cost     float64          `json:"cost"`
	Tokens   barTokensData    `json:"tokens"`
	Diff     *barDiffData     `json:"diff,omitempty"`
	Jobs     int              `json:"jobs"`
	Queued   int              `json:"queued"`
	PRs      []barPRData      `json:"prs,omitempty"`
	WorkItem *barWorkItemData `json:"workItem,omitempty"`
}

type barTokensData struct {
	Input      int `json:"input"`
	Output     int `json:"output"`
	CacheRead  int `json:"cacheRead"`
	CacheWrite int `json:"cacheWrite"`
	Context    int `json:"context"`
}

type barDiffData struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
}

type barPRData struct {
	Provider   string `json:"provider"`
	Number     int    `json:"number"`
	State      string `json:"state"`
	Unresolved int    `json:"unresolved"`
}

type barWorkItemData struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
	Status   string `json:"status"`
}

type barTotalsData struct {
	Sessions   int     `json:"sessions"`
	Busy       int     `json:"busy"`
	Cost       float64 `json:"cost"`
	CostWindow string  `json:"costWindow"`
}

func (m Model) sessionPayload(item row) []byte {
	data := barSessionData{
		Name:    item.name,
		Title:   item.title,
		Dir:     item.dir,
		WorkDir: item.workDir,
		Model:   item.model,
		Mode:    item.mode,
		Effort:  item.effort,
		State:   item.label,
		Live:    item.live,
		Control: item.control,
		Cost:    item.cost,
		Tokens: barTokensData{
			Input:      item.input,
			Output:     item.output,
			CacheRead:  item.cacheRead,
			CacheWrite: item.cacheWrite,
			Context:    item.context,
		},
		Jobs:   item.jobs,
		Queued: item.queued,
	}
	if d, ok := m.diffs[item.name]; ok && d.anyRepo() && !d.stat().Empty() {
		stat := d.stat()
		data.Diff = &barDiffData{Added: stat.Insertions, Removed: stat.Deletions}
	}
	for _, pr := range item.prs {
		if pr.Number == 0 {
			continue
		}
		data.PRs = append(data.PRs, barPRData{
			Provider:   pr.Provider,
			Number:     pr.Number,
			State:      pr.State,
			Unresolved: pr.Unresolved,
		})
	}
	if item.workItem.Key != "" || item.workItem.Status != "" {
		data.WorkItem = &barWorkItemData{
			Provider: item.workItem.Provider,
			Key:      item.workItem.Key,
			Status:   item.workItem.Status,
		}
	}
	return marshalPayload(barPayload{Bar: "session", Session: &data})
}

func (m Model) statusPayload() []byte {
	live, busy := m.liveBusy()
	return marshalPayload(barPayload{Bar: "status", Totals: &barTotalsData{
		Sessions:   live,
		Busy:       busy,
		Cost:       m.cost,
		CostWindow: m.costWindow,
	}})
}

func marshalPayload(payload barPayload) []byte {
	data, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}")
	}
	return data
}
