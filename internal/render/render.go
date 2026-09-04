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
	ClassBash
)

type Line struct {
	Class Class
	Text  string
	// Full holds the whole body when Text is only a collapsed summary. It is
	// empty for a plain line. See docs/tui/output.md.
	Full string
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
		return []Line{{Class: ClassMeta, Text: fmt.Sprintf("[%s -> %s]", ev.Prev, ev.State)}}
	case session.KindStderr:
		if strings.TrimSpace(ev.Line) == "" {
			return nil
		}
		return []Line{{Class: ClassStderr, Text: "stderr: " + ev.Line}}
	case session.KindError:
		if ev.Line != "" {
			return []Line{{Class: ClassError, Text: fmt.Sprintf("! %v: %s", ev.Err, r.clip(ev.Line))}}
		}
		return []Line{{Class: ClassError, Text: fmt.Sprintf("! %v", ev.Err)}}
	}
	return nil
}

func (r Renderer) protocolLines(ev protocol.Event) []Line {
	switch {
	case ev.IsInit() && ev.Init != nil:
		return []Line{{Class: ClassMeta, Text: fmt.Sprintf("● %s · %s · %d tools · %s",
			ev.Init.SessionID, ev.Init.Model, len(ev.Init.Tools), ev.Init.PermissionMode)}}
	case ev.Type == protocol.TypeAssistant && ev.Message != nil:
		return r.messageLines(ev.Message)
	case ev.Type == protocol.TypeUser && ev.IsReplay && ev.Message != nil:
		return PromptLines(ev.Message.Content.Text())
	case ev.Type == protocol.TypeUser && ev.Message != nil:
		return r.messageLines(ev.Message)
	case ev.Type == protocol.TypeResult && ev.Result != nil:
		return []Line{{Class: ClassMeta, Text: r.resultLine(ev.Result)}}
	case ev.Type == protocol.TypeSystem && ev.Task != nil:
		return taskLines(ev.Subtype, ev.Task)
	case ev.Type == protocol.TypeStreamEvent:
		return nil
	}
	if r.Verbose {
		return []Line{{Class: ClassMeta, Text: fmt.Sprintf("· %s", ev.Type)}}
	}
	return nil
}

// BashLines renders a bash command and its output for the session pane; see docs/tui/input.md.
func BashLines(command, output string, err error) []Line {
	out := []Line{{Class: ClassBash, Text: "! " + command}}
	if body := strings.TrimRight(output, "\n"); body != "" {
		for _, line := range strings.Split(body, "\n") {
			out = append(out, Line{Class: ClassBash, Text: line})
		}
	}
	if err != nil {
		out = append(out, Line{Class: ClassError, Text: "! " + err.Error()})
	}
	return out
}

// taskLines renders a background job lifecycle event for the session pane; see
// docs/tui/sessions.md. The start event carries the description; a later event
// carries only the id, so its line names the id.
func taskLines(subtype string, task *protocol.Task) []Line {
	switch subtype {
	case protocol.SubtypeTaskStarted:
		label := task.Description
		if label == "" {
			label = task.TaskID
		}
		return []Line{{Class: ClassMeta, Text: "⚙ started · " + label}}
	case protocol.SubtypeTaskUpdated:
		if task.Patch == nil || task.Patch.Status == "" {
			return nil
		}
		return []Line{{Class: ClassMeta, Text: "⚙ " + session.StatusWord(task.Patch.Status) + " · " + task.TaskID}}
	}
	return nil
}

func PromptLines(text string) []Line {
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
		out = append(out, Line{Class: ClassPrompt, Text: prefix + line})
	}
	return out
}

func (r Renderer) messageLines(msg *protocol.Message) []Line {
	var out []Line
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if text := strings.TrimRight(block.Text, "\n"); text != "" {
				out = append(out, Line{Class: ClassText, Text: text})
			}
		case "thinking":
			if r.Verbose && block.Thinking != "" {
				out = append(out, Line{Class: ClassThinking, Text: "  thinking: " + r.clip(block.Thinking)})
			}
		case "tool_use":
			out = append(out, Line{Class: ClassToolUse,
				Text: fmt.Sprintf("→ %s %s", block.Name, r.clip(summariseInput(block.Input)))})
		case "tool_result":
			text, full := r.toolResultLine(block)
			out = append(out, Line{Class: ClassToolResult, Text: text, Full: full})
		}
	}
	return out
}

// toolResultLine returns the line shown in the pane and, when that line is only
// a collapsed summary, the whole body so the pane can open it. A body that is
// shown in full returns an empty second value.
func (r Renderer) toolResultLine(block protocol.Block) (string, string) {
	text := block.Content.Text()
	mark := "←"
	if block.IsError {
		mark = "←!"
	}
	if r.Verbose {
		return fmt.Sprintf("%s %s", mark, text), ""
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Sprintf("%s result", mark), ""
	}
	trimmed := strings.TrimRight(text, "\n")
	lines := strings.Count(trimmed, "\n") + 1
	if lines == 1 {
		clipped := r.clip(text)
		if clipped != strings.TrimSpace(text) {
			return fmt.Sprintf("%s %s", mark, clipped), trimmed
		}
		return fmt.Sprintf("%s %s", mark, clipped), ""
	}
	return fmt.Sprintf("%s %d lines", mark, lines), trimmed
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
