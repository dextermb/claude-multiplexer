package protocol

import (
	"encoding/json"
	"io"
	"sync"
)

type userMessage struct {
	Type            string      `json:"type"`
	Message         innerUser   `json:"message"`
	ParentToolUseID interface{} `json:"parent_tool_use_id"`
	SessionID       string      `json:"session_id,omitempty"`
}

type innerUser struct {
	Role    string      `json:"role"`
	Content []textBlock `json:"content"`
}

type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type controlRequest struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
}

type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) SendUser(sessionID, text string) error {
	msg := userMessage{
		Type: string(TypeUser),
		Message: innerUser{
			Role:    "user",
			Content: []textBlock{{Type: "text", Text: text}},
		},
		SessionID: sessionID,
	}
	return w.writeJSON(msg)
}

func (w *Writer) SendInterrupt(requestID string) error {
	return w.SendControl(requestID, struct {
		Subtype string `json:"subtype"`
	}{"interrupt"})
}

func (w *Writer) SendControl(requestID string, request any) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return w.writeJSON(controlRequest{
		Type:      string(TypeControlRequest),
		RequestID: requestID,
		Request:   payload,
	})
}

func (w *Writer) writeJSON(v any) error {
	line, err := json.Marshal(v)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.w.Write(line)
	return err
}
