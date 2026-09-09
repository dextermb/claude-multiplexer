// Package wire is the serialized form of a session event, streamed from the host
// that runs a session to the host that views it. It holds what the output pane
// consumes: the rendered lines, the snapshot, the partial text, the todos, and
// the questions. It carries no unmarshalable field, so a host encodes it to the
// stream and the peer decodes it. See docs/peers.md.
package wire

import (
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

// Event is one streamed session event. It mirrors the manager's own event, so
// the host that views a remote session republishes it to its bus under a local
// name, and the pane treats it as local.
type Event struct {
	Session    string              `json:"session"`
	Kind       int                 `json:"kind"`
	Lines      []render.Line       `json:"lines,omitempty"`
	Partial    string              `json:"partial,omitempty"`
	Snapshot   Snapshot            `json:"snapshot"`
	Closed     bool                `json:"closed,omitempty"`
	Questions  []protocol.Question `json:"questions,omitempty"`
	QuestionID string              `json:"question_id,omitempty"`
	Todos      []protocol.Todo     `json:"todos,omitempty"`
}

// Snapshot mirrors the session snapshot the pane reads, with the error as a
// string, so the whole snapshot marshals to the stream.
type Snapshot struct {
	Name            string  `json:"name"`
	Title           string  `json:"title,omitempty"`
	Dir             string  `json:"dir,omitempty"`
	Model           string  `json:"model,omitempty"`
	PermissionMode  string  `json:"permission_mode,omitempty"`
	Effort          string  `json:"effort,omitempty"`
	State           string  `json:"state"`
	ClaudeSessionID string  `json:"claude_session_id,omitempty"`
	Cost            float64 `json:"cost_usd"`
	Turns           int     `json:"turns"`
	Queued          int     `json:"queued"`
	LastDurationMS  int64   `json:"last_duration_ms"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ContextTokens   int     `json:"context_tokens"`
	Err             string  `json:"error,omitempty"`
	Jobs            []Job   `json:"jobs,omitempty"`
}

// Job mirrors one background job in a snapshot.
type Job struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	Status      string `json:"status"`
	Running     bool   `json:"running"`
}
