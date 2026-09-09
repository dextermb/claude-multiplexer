package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

// Messages reads the recent conversation of a session from its transcript, so
// it works for a stored session as well as a live one. A streamed session has
// no local transcript, so it reads the messages from the peer.
func (m *Manager) Messages(name string, limit int) ([]mcp.Message, error) {
	if re := m.remote(name); re != nil {
		return re.client.Messages(context.Background(), re.remoteName, limit)
	}
	file, err := os.Open(transcriptPath(m.opts.Root, name))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSession, name)
	}
	defer file.Close()

	reader := protocol.NewReader(file)
	var out []mcp.Message
	for {
		ev, err := reader.Next()
		if errors.Is(err, protocol.ErrNotJSON) {
			continue
		}
		if err != nil {
			break
		}
		if message, ok := messageOf(ev); ok {
			out = append(out, message)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func messageOf(ev protocol.Event) (mcp.Message, bool) {
	switch ev.Type {
	case protocol.TypeUser:
		text := ev.Text()
		if text == "" {
			return mcp.Message{}, false
		}
		return mcp.Message{Role: "user", Text: text}, true
	case protocol.TypeAssistant:
		text := assistantText(ev)
		if text == "" {
			return mcp.Message{}, false
		}
		return mcp.Message{Role: "assistant", Text: text}, true
	case protocol.TypeResult:
		if ev.Result == nil {
			return mcp.Message{}, false
		}
		text := ev.Result.Result
		if text == "" {
			text = ev.Result.Subtype
		}
		return mcp.Message{Role: "result", Text: text}, true
	}
	return mcp.Message{}, false
}

// assistantText keeps the words and names each tool the model used, because the
// whole input of a tool call is large and it is rarely what a reader wants.
func assistantText(ev protocol.Event) string {
	var parts []string
	if text := ev.Text(); text != "" {
		parts = append(parts, text)
	}
	if ev.Message != nil {
		for _, block := range ev.Message.Content {
			if block.Type == "tool_use" && block.Name != "" {
				parts = append(parts, "[used "+block.Name+"]")
			}
		}
	}
	return strings.Join(parts, "\n")
}
