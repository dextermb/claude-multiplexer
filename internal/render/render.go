package render

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const defaultToolWidth = 100

type Class uint8

const (
	ClassPrompt Class = iota
	ClassText
	ClassThinking
	ClassToolUse
	ClassToolResult
	ClassMeta
	ClassStderr
	ClassError
)

type Line struct {
	Class Class
	Text  string
}

func Text(lines []Line) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, line.Text)
	}
	return out
}

type Renderer struct {
	Verbose   bool
	ToolWidth int
}

func (r Renderer) Lines(ev session.Event) []Line {
	switch ev.Kind {
	case session.KindProtocol:
		return r.protocolLines(ev.Protocol)
	case session.KindState:
		if !r.Verbose {
			return nil
		}
		return []Line{{ClassMeta, fmt.Sprintf("[%s -> %s]", ev.Prev, ev.State)}}
	case session.KindStderr:
		if strings.TrimSpace(ev.Line) == "" {
			return nil
		}
		return []Line{{ClassStderr, "stderr: " + ev.Line}}
	case session.KindError:
		if ev.Line != "" {
			return []Line{{ClassError, fmt.Sprintf("! %v: %s", ev.Err, r.clip(ev.Line))}}
		}
		return []Line{{ClassError, fmt.Sprintf("! %v", ev.Err)}}
	}
	return nil
}

func (r Renderer) protocolLines(ev protocol.Event) []Line {
	switch {
	case ev.IsInit() && ev.Init != nil:
		return []Line{{ClassMeta, fmt.Sprintf("● %s · %s · %d tools · %s",
			ev.Init.SessionID, ev.Init.Model, len(ev.Init.Tools), ev.Init.PermissionMode)}}
	case ev.Type == protocol.TypeAssistant && ev.Message != nil:
		return r.messageLines(ev.Message)
	case ev.Type == protocol.TypeUser && ev.IsReplay && ev.Message != nil:
		return promptLines(ev.Message.Content.Text())
	case ev.Type == protocol.TypeUser && ev.Message != nil:
		return r.messageLines(ev.Message)
	case ev.Type == protocol.TypeResult && ev.Result != nil:
		return []Line{{ClassMeta, r.resultLine(ev.Result)}}
	case ev.Type == protocol.TypeStreamEvent:
		return nil
	}
	if r.Verbose {
		return []Line{{ClassMeta, fmt.Sprintf("· %s", ev.Type)}}
	}
	return nil
}

func promptLines(text string) []Line {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	var out []Line
	for i, line := range strings.Split(text, "\n") {
		prefix := "  "
		if i == 0 {
			prefix = "› "
		}
		out = append(out, Line{ClassPrompt, prefix + line})
	}
	return out
}

func (r Renderer) messageLines(msg *protocol.Message) []Line {
	var out []Line
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if text := strings.TrimRight(block.Text, "\n"); text != "" {
				out = append(out, Line{ClassText, text})
			}
		case "thinking":
			if r.Verbose && block.Thinking != "" {
				out = append(out, Line{ClassThinking, "  thinking: " + r.clip(block.Thinking)})
			}
		case "tool_use":
			out = append(out, Line{ClassToolUse,
				fmt.Sprintf("→ %s %s", block.Name, r.clip(summariseInput(block.Input)))})
		case "tool_result":
			out = append(out, Line{ClassToolResult, r.toolResultLine(block)})
		}
	}
	return out
}

func (r Renderer) toolResultLine(block protocol.Block) string {
	text := block.Content.Text()
	mark := "←"
	if block.IsError {
		mark = "←!"
	}
	if r.Verbose {
		return fmt.Sprintf("%s %s", mark, text)
	}
	lines := strings.Count(strings.TrimRight(text, "\n"), "\n") + 1
	if strings.TrimSpace(text) == "" {
		return fmt.Sprintf("%s result", mark)
	}
	if lines == 1 {
		return fmt.Sprintf("%s %s", mark, r.clip(text))
	}
	return fmt.Sprintf("%s %d lines", mark, lines)
}

func (r Renderer) resultLine(res *protocol.Result) string {
	headline := "✓ " + res.Subtype
	if res.IsError {
		headline = "✗ error"
	}
	parts := []string{
		headline,
		formatDuration(res.DurationMS),
		"$" + strconv.FormatFloat(res.TotalCostUSD, 'f', 4, 64),
		plural(res.NumTurns, "turn"),
	}
	if res.Usage != nil {
		parts = append(parts, fmt.Sprintf("%d in / %d out",
			TotalInputTokens(res.Usage), res.Usage.OutputTokens))
	}
	return strings.Join(parts, " · ")
}

func (r Renderer) clip(text string) string {
	width := r.ToolWidth
	if width <= 0 {
		width = defaultToolWidth
	}
	text = strings.ReplaceAll(strings.TrimSpace(text), "\n", " ")
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return string(runes[:width]) + "…"
}

func summariseInput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return string(raw)
	}
	for _, key := range []string{"command", "file_path", "path", "pattern", "query", "prompt", "description"} {
		if value, ok := fields[key].(string); ok && value != "" {
			return value
		}
	}
	compact, err := json.Marshal(fields)
	if err != nil {
		return string(raw)
	}
	return string(compact)
}

func TotalInputTokens(usage *protocol.Usage) int {
	if usage == nil {
		return 0
	}
	return usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "0s"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}
