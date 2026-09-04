package protocol

import "encoding/json"

type Type string

const (
	TypeSystem          Type = "system"
	TypeAssistant       Type = "assistant"
	TypeUser            Type = "user"
	TypeResult          Type = "result"
	TypeStreamEvent     Type = "stream_event"
	TypeControlRequest  Type = "control_request"
	TypeControlResponse Type = "control_response"
)

const (
	SubtypeInit             = "init"
	SubtypeTaskStarted      = "task_started"
	SubtypeTaskUpdated      = "task_updated"
	SubtypeTaskNotification = "task_notification"
)

type Event struct {
	Type      Type
	Subtype   string
	SessionID string
	Raw       []byte
	Truncated bool
	IsReplay  bool

	Init    *Init
	Message *Message
	Result  *Result
	Delta   *Delta
	Task    *Task
}

type Init struct {
	SessionID      string      `json:"session_id"`
	Model          string      `json:"model"`
	CWD            string      `json:"cwd"`
	Tools          []string    `json:"tools"`
	PermissionMode string      `json:"permissionMode"`
	APIKeySource   string      `json:"apiKeySource"`
	MCPServers     []MCPServer `json:"mcp_servers"`
}

type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Message struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	Model      string  `json:"model"`
	Content    Content `json:"content"`
	StopReason string  `json:"stop_reason"`
	Usage      *Usage  `json:"usage"`
}

type Block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   Content         `json:"content"`
	IsError   bool            `json:"is_error"`
}

type Content []Block

func (c *Content) UnmarshalJSON(data []byte) error {
	trimmed := trimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*c = nil
		return nil
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}
		*c = Content{{Type: "text", Text: text}}
		return nil
	}
	var blocks []Block
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return err
	}
	*c = Content(blocks)
	return nil
}

func (c Content) Text() string {
	var out []byte
	for _, b := range c {
		if b.Type != "text" || b.Text == "" {
			continue
		}
		if len(out) > 0 {
			out = append(out, '\n')
		}
		out = append(out, b.Text...)
	}
	return string(out)
}

type Result struct {
	Subtype       string  `json:"subtype"`
	IsError       bool    `json:"is_error"`
	DurationMS    int64   `json:"duration_ms"`
	DurationAPIMS int64   `json:"duration_api_ms"`
	NumTurns      int     `json:"num_turns"`
	Result        string  `json:"result"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	SessionID     string  `json:"session_id"`
	Usage         *Usage  `json:"usage"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type Delta struct {
	Kind        string
	Index       int
	Text        string
	Thinking    string
	PartialJSON string
}

// Task carries a background job lifecycle event. Claude Code pushes one on
// task_started, task_updated, and task_notification; see docs/protocol.md.
type Task struct {
	TaskID      string     `json:"task_id"`
	ToolUseID   string     `json:"tool_use_id"`
	Description string     `json:"description"`
	TaskType    string     `json:"task_type"`
	Status      string     `json:"status"`
	Summary     string     `json:"summary"`
	OutputFile  string     `json:"output_file"`
	Patch       *TaskPatch `json:"patch"`
}

type TaskPatch struct {
	Status  string `json:"status"`
	EndTime int64  `json:"end_time"`
}

func (e Event) IsInit() bool {
	return e.Type == TypeSystem && e.Subtype == SubtypeInit
}

func (e Event) Text() string {
	if e.Message != nil {
		return e.Message.Content.Text()
	}
	if e.Delta != nil {
		return e.Delta.Text
	}
	return ""
}

func trimSpace(b []byte) []byte {
	start := 0
	for start < len(b) && isSpace(b[start]) {
		start++
	}
	end := len(b)
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
